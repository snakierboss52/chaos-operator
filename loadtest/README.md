# Pruebas de carga — Chaos Engineering Operator

Quick reference. Para detalles paso a paso ver [`runbook.md`](./runbook.md).

```bash
make load-up        # crea kind si no existe
make load-deploy    # build + deploy operator + monitoring + workloads
make load-bin       # construye k6 con xk6-kubernetes (1 sola vez)
make load-prom-pf   # port-forward Prometheus a :9090

make load-smoke     # 30s, 1 VU
make load-steady    # 5min, 5 VUs (override: VUS=N DURATION=Nm)
make load-spike     # ramp 0→100 VUs en 10s, hold 30s

make load-clean     # borra CRs de carga
make load-down      # elimina workloads + port-forward
make load-teardown  # destruye el cluster kind
```

## Estructura

```
loadtest/
├── load.mk                  # targets de Make (incluido en /Makefile)
├── README.md                # este archivo
├── runbook.md               # guía detallada paso a paso
├── workloads/
│   └── load-target.yaml     # deployment nginx (50 réplicas) + ns chaos-load
└── scripts/
    ├── 00-smoke.js          # 1 VU x 30s
    ├── 01-steady.js         # 5 VUs x 5min
    ├── 02-spike.js          # ramping 0→100 VUs
    └── lib/
        ├── crds.js          # fábricas de manifests (PodChaos, NetworkChaos)
        └── prom.js          # consultas Prometheus con allowlist
```

## SLOs validados

| Métrica | Smoke | Steady | Spike |
|---------|-------|--------|-------|
| `chaos_reconciliation_duration_seconds` p95 | < 5 s | < 1.5 s | n/a |
| Error rate del operator | 0 | < 0.5/s | < 20 errores totales |
| `chaos_load_apply_duration_ms` p95 | n/a | < 3 s | < 2 s |
| Reconciliación a estado terminal | 100% | > 98% | n/a (validación post-test) |

## Aislamiento

Todos los CRs de carga viven en el namespace `chaos-load` con `blastRadius.maxPods=1` y selector estricto `app=load-target`, por lo que el impacto está siempre acotado.
