// 00-smoke.js — Sanity check: 1 VU durante 30s creando PodChaos cada ~3s.
//
// Objetivo: validar que el setup completo (kind + operator + workload + k6)
// funciona y que el operator responde a una creación nominal.
//
// Ejecutar:  k6 run loadtest/scripts/00-smoke.js
// Requiere:  KUBECONFIG apuntando al cluster kind chaos-testing-v2.

import { Kubernetes } from 'k6/x/kubernetes';
import { check, sleep, fail } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { podChaosManifest, uniqueName, CONSTANTS } from './lib/crds.js';
import { snapshot } from './lib/prom.js';
import { buildSummary } from './lib/report.js';

const crsCreated = new Counter('chaos_load_crs_created');
const crsCompleted = new Counter('chaos_load_crs_completed');
const crsFailed = new Counter('chaos_load_crs_failed');
const reconcileWait = new Trend('chaos_load_reconcile_wait_ms');

export const options = {
  vus: 1,
  duration: '30s',
  thresholds: {
    'chaos_load_crs_created': ['count>=8'],
    'chaos_load_crs_failed': ['count==0'],
    'chaos_load_reconcile_wait_ms': ['p(95)<5000'],
    'checks': ['rate==1.0'],
  },
};

const k8s = new Kubernetes();

export default function () {
  const name = uniqueName('smoke', __VU, __ITER);
  const yaml = podChaosManifest({ name, duration: '5s' });

  const t0 = Date.now();
  try {
    k8s.apply(yaml);
  } catch (e) {
    crsFailed.add(1);
    fail(`apply failed: ${e}`);
  }
  crsCreated.add(1);

  // Esperar hasta que el operator avance a Running o Completed (max 5s).
  let phase = '';
  let attempts = 0;
  while (attempts < 25 && phase !== 'Running' && phase !== 'Completed') {
    sleep(0.2);
    attempts++;
    try {
      const obj = k8s.get('PodChaos.chaos.engineering.io', name, CONSTANTS.ALLOWED_NAMESPACE);
      phase = (obj && obj.status && obj.status.phase) || '';
    } catch (_) {
      // El recurso aún puede estar siendo aceptado por el webhook.
    }
  }
  reconcileWait.add(Date.now() - t0);

  const reconciled = check(phase, {
    'phase reached Running/Completed': (p) => p === 'Running' || p === 'Completed',
  });
  if (reconciled) crsCompleted.add(1);
  else crsFailed.add(1);

  // Cleanup: eliminamos el CR para no acumular estado.
  try {
    k8s.delete('PodChaos.chaos.engineering.io', name, CONSTANTS.ALLOWED_NAMESPACE);
  } catch (_) {
    // Si el CR ya fue garbage-collected por el operator, no es error.
  }

  sleep(2);
}

export function teardown() {
  // Snapshot final de Prometheus para visibilidad en logs.
  const s = snapshot();
  console.log(`[smoke] prom snapshot: ${JSON.stringify(s)}`);
}

export function handleSummary(data) {
  return buildSummary(data, 'smoke');
}
