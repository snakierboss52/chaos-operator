# 📊 Guía de Monitoreo y Observabilidad

Este documento describe cómo usar el stack de monitoreo con Prometheus y Grafana para observar tus experimentos de Chaos Engineering.

## 🚀 Instalación Rápida

```bash
make kind-up
make addons-install
```

El stack local usa charts upstream de Helm para Prometheus, Grafana y
kube-state-metrics. Las versiones y valores de kind están en
`scripts/helms/`. kube-state-metrics se instala como release separado porque
los dashboards consultan métricas `kube_*`. InfluxDB permanece como manifiesto
local para soportar el streaming de k6.

Para quitar las releases: `make addons-uninstall`. La configuración de AWS se
añadirá como perfil separado; la configuración local usa credenciales de
desarrollo, por lo que no debe aplicarse a un cluster AWS.

## 🔗 Acceso a las Interfaces

### Para Kind (cluster local):

Inicia port-forwards en terminales separadas:

```bash
kubectl port-forward -n monitoring svc/prometheus-server 9090:80
kubectl port-forward -n monitoring svc/grafana 3000:80
```

O inicia ambos juntos con `./monitoring/access-monitoring.sh` y detén ambos con
Ctrl+C.

Luego abre:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

### Credenciales de Grafana

- **Usuario**: `admin`
- **Contraseña**: `admin`

## 📈 Métricas Disponibles

El operador exporta las siguientes métricas en Prometheus:

### Métricas de Experimentos

| Métrica | Tipo | Descripción | Labels |
|---------|------|-------------|--------|
| `chaos_experiments_total` | Counter | Total de experimentos creados | type, namespace, action |
| `chaos_experiments_active` | Gauge | Experimentos actualmente activos | type, namespace, phase |
| `chaos_experiments_duration_seconds` | Histogram | Duración de los experimentos | type, namespace, action |
| `chaos_targets_affected_total` | Counter | Total de targets (pods) afectados | type, namespace, action |
| `chaos_experiments_status_total` | Counter | Experimentos por estado final | type, namespace, status |

### Métricas de Control

| Métrica | Tipo | Descripción | Labels |
|---------|------|-------------|--------|
| `chaos_blast_radius_limit_total` | Counter | Veces que se alcanzó el límite de blast radius | type, namespace |
| `chaos_execution_errors_total` | Counter | Errores de ejecución | type, namespace, error_type |
| `chaos_reconciliation_duration_seconds` | Histogram | Duración del loop de reconciliación | type, namespace |

## 📊 Dashboard de Chaos Engineering

El dashboard incluye los siguientes paneles:

### 1. Vista General (Top Row)
- **Total Chaos Experiments**: Número total de experimentos creados
- **Active Experiments**: Experimentos actualmente en ejecución
- **Targets Affected**: Total de pods afectados
- **Execution Errors**: Errores de ejecución

### 2. Análisis Temporal
- **Experiments by Type Over Time**: Tasa de experimentos por tipo
- **Active Experiments by Phase**: Distribución de fases
- **Experiment Duration (p95)**: Percentil 95 de duración
- **Targets Affected by Action**: Targets por tipo de acción

### 3. Estado y Errores
- **Experiments by Status**: Distribución por estado (Completed/Failed)
- **Reconciliation Duration**: Tiempo de procesamiento del operador
- **Blast Radius Limits Hit**: Límites de seguridad alcanzados
- **Errors by Type**: Tabla de errores por categoría

## 🔍 Queries Útiles en Prometheus

### Ver experimentos activos:
```promql
chaos_experiments_active
```

### Tasa de experimentos por minuto:
```promql
rate(chaos_experiments_total[5m])
```

### Duración promedio de experimentos:
```promql
rate(chaos_experiments_duration_seconds_sum[5m]) / rate(chaos_experiments_duration_seconds_count[5m])
```

### Tasa de errores:
```promql
rate(chaos_execution_errors_total[5m])
```

### Pods afectados por namespace:
```promql
sum(chaos_targets_affected_total) by (namespace)
```

### Experimentos completados vs fallidos:
```promql
chaos_experiments_status_total{status="Completed"}
chaos_experiments_status_total{status="Failed"}
```

## 🎯 Ejemplo de Uso

### 1. Ejecutar un experimento:

```bash
kubectl apply -f samples/podchaos/basic-kill-one.yaml
```

### 2. Observar en Grafana:

Abre http://localhost:3000 y navega al dashboard "Chaos Engineering Dashboard"

Verás:
- ✅ **Total Experiments** incrementarse
- 📊 **Active Experiments** mostrar el experimento en ejecución
- 🎯 **Targets Affected** incrementarse cuando se ejecute
- ⏱️ **Duration** registrar la duración cuando complete

### 3. Verificar métricas en Prometheus:

Abre http://localhost:9090 y ejecuta:

```promql
chaos_experiments_total{namespace="chaos-demo"}
```

## 🔔 Configurar Alertas (Opcional)

Puedes agregar reglas de alertas en Prometheus:

```yaml
groups:
  - name: chaos_engineering
    interval: 10s
    rules:
      - alert: TooManyActiveExperiments
        expr: sum(chaos_experiments_active) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Demasiados experimentos activos"
          
      - alert: HighErrorRate
        expr: rate(chaos_execution_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Alta tasa de errores en experimentos"
```

## 📊 Monitoreo del Cluster

Además de las métricas del operador, puedes monitorear el estado del cluster:

### Métricas de Kubernetes (adicionales):

```promql
# Pods por fase
kube_pod_status_phase

# Uso de CPU
container_cpu_usage_seconds_total

# Uso de memoria
container_memory_usage_bytes

# Pods reiniciados
rate(kube_pod_container_status_restarts_total[5m])
```

## 🛠️ Troubleshooting

### Prometheus no muestra métricas del operador:

1. Verificar que el operador está corriendo:
```bash
kubectl get pods -n default -l app=chaos-operator
```

2. Verificar que las métricas están disponibles:
```bash
kubectl port-forward -n default <operator-pod> 8080:8080
curl http://localhost:8080/metrics | grep chaos_
```

3. Verificar configuración de Prometheus:
```bash
kubectl logs -n monitoring -l app=prometheus
```

### Grafana no muestra el dashboard:

1. Verificar que el ConfigMap existe:
```bash
kubectl get configmap grafana-dashboards -n monitoring
```

2. Reiniciar Grafana:
```bash
kubectl rollout restart deployment/grafana -n monitoring
```

3. Importar manualmente:
   - Ir a Dashboards → Import
   - Copiar el contenido de `monitoring/chaos-dashboard.json`
   - Importar

## 📚 Recursos Adicionales

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)

## 🎓 Mejores Prácticas

1. **Retención de Datos**: Por defecto, Prometheus guarda 15 días de datos
2. **Refresh Rate**: El dashboard se actualiza cada 10 segundos
3. **Alertas**: Configura alertas para eventos críticos
4. **Backups**: Haz backup de tus dashboards personalizados
5. **Resource Limits**: Ajusta los límites de memoria/CPU según tu uso

---

¿Preguntas? Consulta el [README.md](README.md) principal o abre un issue.
