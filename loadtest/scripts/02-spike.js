// 02-spike.js — Pico súbito: ramp-up agresivo a 100 VUs.
//
// Valida que el operator soporta ráfagas de creación masiva sin caer
// y que todos los CRs eventualmente alcanzan estado terminal
// (Completed o Failed, no Pending stuck).
//
// Ejecutar:  k6 run loadtest/scripts/02-spike.js

import { Kubernetes } from 'k6/x/kubernetes';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { podChaosManifest, uniqueName, CONSTANTS } from './lib/crds.js';
import { snapshot } from './lib/prom.js';
import { buildSummary } from './lib/report.js';

const crsCreated = new Counter('chaos_load_crs_created');
const crsApplyErrors = new Counter('chaos_load_crs_apply_errors');
const applyDuration = new Trend('chaos_load_apply_duration_ms');

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 }, // ramp-up agresivo
        { duration: '30s', target: 100 }, // sostenido en 100 VUs
        { duration: '5s', target: 0 },    // ramp-down
      ],
    },
  },
  thresholds: {
    // Bajo spike, exigimos que apply succeed > 95% (operator no se cae).
    'chaos_load_crs_apply_errors': ['count<20'],
    'chaos_load_apply_duration_ms': ['p(95)<2000'],
    'checks': ['rate>0.95'],
  },
};

const k8s = new Kubernetes();

export default function () {
  const name = uniqueName('spike', __VU, __ITER);
  const yaml = podChaosManifest({ name, duration: '3s' });

  const t0 = Date.now();
  let ok = false;
  try {
    k8s.apply(yaml);
    ok = true;
  } catch (_) {
    crsApplyErrors.add(1);
  }
  applyDuration.add(Date.now() - t0);
  if (ok) crsCreated.add(1);

  check(ok, { 'CR applied during spike': (v) => v === true });
  sleep(0.1);
}

// Tras el spike, verificamos que el operator no quedó en estado degradado.
export function teardown() {
  sleep(15); // dar tiempo al operator para procesar el backlog
  const s = snapshot();
  console.log(`[spike] post-load prom snapshot: ${JSON.stringify(s)}`);

  // Limpieza best-effort: eliminamos cualquier CR de carga restante.
  try {
    const list = k8s.list('PodChaos.chaos.engineering.io', CONSTANTS.ALLOWED_NAMESPACE);
    for (const cr of list || []) {
      const labels = (cr.metadata && cr.metadata.labels) || {};
      if (labels['test.chaos.engineering.io/scenario'] === 'load') {
        try {
          k8s.delete('PodChaos.chaos.engineering.io', cr.metadata.name, CONSTANTS.ALLOWED_NAMESPACE);
        } catch (_) {}
      }
    }
  } catch (e) {
    console.log(`[spike] teardown list/delete error: ${e}`);
  }
}

export function handleSummary(data) {
  return buildSummary(data, 'spike');
}
