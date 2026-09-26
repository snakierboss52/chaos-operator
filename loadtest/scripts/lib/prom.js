// prom.js — helper para consultar Prometheus desde k6.
//
// Acepta solo la URL configurada vía variable de entorno PROM_URL.
// No interpolamos URLs ni queries con input externo no validado.

import http from 'k6/http';
import { check } from 'k6';

const PROM_URL = __ENV.PROM_URL || 'http://localhost:9090';

// Allowlist de queries permitidas — defense in depth contra injection
// en PromQL. Las queries son hardcoded, no parametrizables por usuario.
const ALLOWED_QUERIES = new Set([
  'reconcile_p95',
  'experiments_active',
  'execution_errors',
  'goroutines',
  'memory_resident',
]);

const QUERIES = {
  reconcile_p95:
    'histogram_quantile(0.95, rate(chaos_reconciliation_duration_seconds_bucket[1m]))',
  experiments_active: 'sum(chaos_experiments_active)',
  execution_errors: 'sum(rate(chaos_execution_errors_total[1m]))',
  goroutines: 'go_goroutines{job=~"chaos-operator.*"}',
  memory_resident: 'process_resident_memory_bytes{job=~"chaos-operator.*"}',
};

export function query(name) {
  if (!ALLOWED_QUERIES.has(name)) {
    throw new Error(`Query no permitida: ${name}`);
  }
  const q = QUERIES[name];
  const url = `${PROM_URL}/api/v1/query?query=${encodeURIComponent(q)}`;
  const res = http.get(url, { timeout: '5s' });
  const ok = check(res, {
    [`prom ${name} 200`]: (r) => r.status === 200,
  });
  if (!ok) return null;
  try {
    const body = JSON.parse(res.body);
    if (body.status !== 'success' || !body.data || !body.data.result) return null;
    const result = body.data.result;
    if (result.length === 0) return 0;
    const v = parseFloat(result[0].value[1]);
    return Number.isFinite(v) ? v : null;
  } catch (_) {
    return null;
  }
}

export function snapshot() {
  return {
    reconcile_p95_seconds: query('reconcile_p95'),
    experiments_active: query('experiments_active'),
    execution_errors_per_sec: query('execution_errors'),
    goroutines: query('goroutines'),
    memory_bytes: query('memory_resident'),
  };
}
