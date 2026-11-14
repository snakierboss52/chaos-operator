# Monitoreo local

El stack del laboratorio kind incluye Traefik, Prometheus, Grafana y
kube-state-metrics. Para instalarlo:

```bash
make kind-up
make addons-install
```

La instalación del operador es independiente. Si aún no está instalada,
consulta [Instalación](INSTALLATION.md). La configuración de charts y recursos
locales está bajo `scripts/helms/` y `monitoring/`.

## Interfaces

- Prometheus: <http://prometheus.localhost:30080>
- Grafana: <http://grafana.localhost:30080>
- Métricas del operador: <http://operator.localhost:30080/metrics>, cuando
  están desplegados el operador y su ruta local.

El usuario/contraseña inicial configurado para Grafana local es `admin` /
`admin`. Cambia las credenciales antes de cualquier uso fuera del entorno
local. El cluster kind mapea el puerto `30080` al loopback; consulta
[`scripts/kind/README.md`](../scripts/kind/README.md) para recrear el cluster o
usar port-forward.

## Métricas del operador

El servicio de métricas es `chaos-operator-metrics` en `chaos-system`, puerto
8080. Para consultarlo directamente:

```bash
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8080:8080
curl http://localhost:8080/metrics
```

Métricas de experimento registradas por el código:

| Métrica | Descripción |
| --- | --- |
| `chaos_experiments_total` | Experimentos observados, etiquetados por tipo/namespace/acción |
| `chaos_experiments_active` | Experimentos activos por fase |
| `chaos_experiments_duration_seconds` | Histograma de duración |
| `chaos_targets_affected_total` | Targets afectados |
| `chaos_experiments_status_total` | Conteo por estado |
| `chaos_blast_radius_limit_total` | Veces que se limitó el blast radius |
| `chaos_execution_errors_total` | Errores de ejecución |
| `chaos_reconciliation_duration_seconds` | Duración de reconciliación |
| `chaos_experiment_info` | Información del experimento observado |
| `chaos_experiment_runtime_seconds` | Runtime del experimento |
| `chaos_experiment_affected_pod` | Pods afectados por experimento |

Ejemplos PromQL:

```promql
sum by (type, namespace) (chaos_experiments_active)
rate(chaos_execution_errors_total[5m])
sum by (namespace) (rate(chaos_targets_affected_total[5m]))
```

Los labels exactos dependen de cada métrica y están definidos en
`internal/infrastructure/metrics.go`.

## Salud del manager

Los probes del manager están en el puerto 8081:

```bash
kubectl port-forward -n chaos-system deployment/chaos-operator 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

## Grafana y Prometheus

Los dashboards provisionados están en `monitoring/`. Las consultas deben
coincidir con los nombres y labels actuales de las métricas; ejecuta primero
las expresiones en Prometheus si adaptas un dashboard.

Para retirar las releases locales ejecuta `make addons-uninstall`. Esa
operación no elimina el operador ni sus CRDs.
