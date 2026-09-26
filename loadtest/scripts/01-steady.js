// 01-steady.js — Carga sostenida nominal: 5 VUs por 5 min.
//
// Cada VU crea un PodChaos cada ~6s ⇒ ~50 CRs/min global, ~250 CRs en total.
// Mide la latencia de reconciliación, error rate, y SLOs del operator.
//
// Ejecutar:  k6 run loadtest/scripts/01-steady.js
// Variables: VUS (default 5), DURATION (default 5m), PROM_URL.

import { Kubernetes } from 'k6/x/kubernetes';
import { check, sleep } from 'k6';
import { Counter, Trend, Gauge } from 'k6/metrics';
import { podChaosManifest, uniqueName, CONSTANTS } from './lib/crds.js';
import { query } from './lib/prom.js';
import { buildSummary } from './lib/report.js';

const crsCreated = new Counter('chaos_load_crs_created');
const crsReconciled = new Counter('chaos_load_crs_reconciled');
const crsTimeout = new Counter('chaos_load_crs_timeout');
const reconcileWait = new Trend('chaos_load_reconcile_wait_ms');
const operatorReconcileP95 = new Gauge('operator_reconcile_p95_seconds');
const operatorErrorRate = new Gauge('operator_error_rate_per_sec');

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-vus',
      vus: parseInt(__ENV.VUS || '5'),
      duration: __ENV.DURATION || '5m',
    },
  },
  thresholds: {
    // SLOs del operator bajo carga sostenida
    'chaos_load_reconcile_wait_ms': ['p(95)<3000', 'p(99)<6000'],
    'chaos_load_crs_timeout': ['count<5'],
    'checks': ['rate>0.98'],
    'operator_reconcile_p95_seconds': ['value<1.5'],
    'operator_error_rate_per_sec': ['value<0.5'],
  },
};

const k8s = new Kubernetes();

export default function () {
  const name = uniqueName('steady', __VU, __ITER);
  const yaml = podChaosManifest({ name, duration: '5s' });

  const t0 = Date.now();
  let applied = false;
  try {
    k8s.apply(yaml);
    applied = true;
  } catch (_) {
    // No fail() — queremos seguir midiendo aún con errores intermitentes.
  }
  if (applied) crsCreated.add(1);

  // Espera bounded de reconciliación (máx 10s).
  let phase = '';
  let attempts = 0;
  while (attempts < 50 && phase !== 'Running' && phase !== 'Completed') {
    sleep(0.2);
    attempts++;
    try {
      const obj = k8s.get('PodChaos.chaos.engineering.io', name, CONSTANTS.ALLOWED_NAMESPACE);
      phase = (obj && obj.status && obj.status.phase) || '';
    } catch (_) {}
  }
  reconcileWait.add(Date.now() - t0);

  if (phase === 'Running' || phase === 'Completed') {
    crsReconciled.add(1);
  } else {
    crsTimeout.add(1);
  }
  check(phase, { 'reconciled': (p) => p === 'Running' || p === 'Completed' });

  // Cleanup
  try {
    k8s.delete('PodChaos.chaos.engineering.io', name, CONSTANTS.ALLOWED_NAMESPACE);
  } catch (_) {}

  sleep(6);
}

// Snapshot Prometheus al cierre — emitido como gauges para que aparezcan
// en el reporte JSON/HTML/MD generado por buildSummary.
export function handleSummary(data) {
  const p95 = query('reconcile_p95');
  const err = query('execution_errors');
  if (p95 !== null) data.metrics['operator_reconcile_p95_seconds'] = {
    type: 'gauge', contains: 'default', values: { value: p95 },
  };
  if (err !== null) data.metrics['operator_error_rate_per_sec'] = {
    type: 'gauge', contains: 'default', values: { value: err },
  };
  return buildSummary(data, 'steady');
}
