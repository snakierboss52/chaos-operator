# 🎉 Sistema de Monitoreo Configurado

¡El sistema completo de observabilidad está listo!

## ✅ Lo que se ha Instalado

### 1. Métricas en el Operador
- ✅ 8 métricas personalizadas de Prometheus
- ✅ Seguimiento de experimentos, targets, errores
- ✅ Métricas de rendimiento del operador

### 2. Stack de Monitoreo
- ✅ Prometheus (recolección de métricas)
- ✅ Grafana (visualización)
- ✅ Dashboard personalizado pre-configurado

### 3. Infraestructura
- ✅ Namespace `monitoring`
- ✅ RBAC configurado
- ✅ ServiceAccounts creados
- ✅ ConfigMaps con configuración

---

## 🚀 Cómo Usar

### Paso 1: Reiniciar el Operador con Métricas

El código ha sido actualizado. Necesitas recompilar y reiniciar:

```bash
# Terminal 1: Detén el operador actual (Ctrl+C)
# Luego recompila y ejecuta:
cd /Users/jlozanoarena/Documents/study/goland-operator
go build ./cmd/manager
./manager
```

### Paso 2: Acceder a las Interfaces

En otra terminal:

```bash
cd /Users/jlozanoarena/Documents/study/goland-operator/monitoring
./access-monitoring.sh
```

O manualmente:

```bash
# Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

### Paso 3: Abrir Grafana

1. Abre tu navegador en: **http://localhost:3000**
2. Login:
   - Usuario: `admin`
   - Contraseña: `admin`
3. Ve a: **Dashboards → Browse → Chaos Engineering Dashboard**

### Paso 4: Generar Métricas

Ejecuta algunos experimentos:

```bash
# Experimento simple
kubectl apply -f test-podchaos.yaml

# O usa los ejemplos
kubectl apply -f samples/podchaos/basic-kill-one.yaml
kubectl apply -f samples/podchaos/kill-percentage.yaml
```

### Paso 5: Observar en el Dashboard

Verás en tiempo real:
- 📊 Total de experimentos
- 🔄 Experimentos activos
- 🎯 Pods afectados
- ⏱️ Duración de experimentos
- ❌ Errores de ejecución

---

## 📊 Métricas Disponibles

### Principales Métricas:

| Métrica | Descripción |
|---------|-------------|
| `chaos_experiments_total` | Total de experimentos creados |
| `chaos_experiments_active` | Experimentos activos ahora |
| `chaos_experiments_duration_seconds` | Duración de experimentos |
| `chaos_targets_affected_total` | Pods afectados |
| `chaos_experiments_status_total` | Status (Completed/Failed) |
| `chaos_execution_errors_total` | Errores de ejecución |
| `chaos_blast_radius_limit_total` | Límites alcanzados |
| `chaos_reconciliation_duration_seconds` | Rendimiento del operador |

---

## 🧪 Probar el Sistema

Ejecuta el script de prueba:

```bash
cd monitoring
./test-metrics.sh
```

Este script:
1. Verifica que todo esté corriendo
2. Crea un experimento de prueba
3. Consulta las métricas
4. Limpia todo

---

## 🔍 Queries Útiles en Prometheus

Abre **http://localhost:9090** y prueba:

### Experimentos totales:
```promql
sum(chaos_experiments_total)
```

### Tasa de experimentos por minuto:
```promql
rate(chaos_experiments_total[5m])
```

### Pods afectados por namespace:
```promql
sum(chaos_targets_affected_total) by (namespace)
```

### Tasa de errores:
```promql
rate(chaos_execution_errors_total[5m])
```

### Duración promedio (segundos):
```promql
rate(chaos_experiments_duration_seconds_sum[5m]) / 
rate(chaos_experiments_duration_seconds_count[5m])
```

---

## 📈 Dashboard de Grafana

El dashboard incluye:

### Fila 1: KPIs Principales
- Total de experimentos
- Experimentos activos
- Targets afectados
- Errores de ejecución

### Fila 2: Análisis Temporal
- Experimentos por tipo (gráfico de líneas)
- Distribución por fase (gráfico circular)

### Fila 3: Rendimiento
- Duración de experimentos (percentil 95)
- Targets por acción

### Fila 4: Estado y Errores
- Experimentos por status
- Duración de reconciliación
- Tabla de errores por tipo

---

## 🛠️ Troubleshooting

### No veo métricas en Grafana:

1. **Verifica que el operador esté corriendo con la nueva versión:**
   ```bash
   ps aux | grep manager
   ```

2. **Verifica que las métricas estén disponibles:**
   ```bash
   kubectl port-forward <operator-pod> 8080:8080
   curl http://localhost:8080/metrics | grep chaos_
   ```

3. **Verifica Prometheus:**
   ```bash
   kubectl logs -n monitoring -l app=prometheus
   ```

4. **Recompila el operador:**
   ```bash
   cd /Users/jlozanoarena/Documents/study/goland-operator
   go build ./cmd/manager
   ./manager
   ```

### Grafana no muestra el dashboard:

1. **Verifica el ConfigMap:**
   ```bash
   kubectl get configmap grafana-dashboards -n monitoring -o yaml
   ```

2. **Reinicia Grafana:**
   ```bash
   kubectl rollout restart deployment/grafana -n monitoring
   ```

3. **Importa manualmente:**
   - En Grafana: Dashboards → Import
   - Usa el archivo: `monitoring/chaos-dashboard.json`

---

## 📚 Documentación Completa

- **[MONITORING.md](MONITORING.md)**: Guía completa de monitoreo
- **[README.md](README.md)**: Documentación principal
- **[QUICKSTART.md](QUICKSTART.md)**: Guía de inicio rápido

---

## 🎯 Workflow Completo

```bash
# Terminal 1: Operador
cd /Users/jlozanoarena/Documents/study/goland-operator
./manager

# Terminal 2: Port-forwards
cd monitoring
./access-monitoring.sh

# Terminal 3: Experimentos
kubectl apply -f samples/podchaos/basic-kill-one.yaml
kubectl get podchaos -w

# Navegador: Grafana
# http://localhost:3000
# Username: admin / Password: admin
# Dashboard: "Chaos Engineering Dashboard"
```

---

## 🎉 ¡Todo Listo!

Tu sistema de Chaos Engineering ahora tiene:
- ✅ Operador funcional con métricas
- ✅ Prometheus recolectando datos
- ✅ Grafana con dashboard personalizado
- ✅ Monitoreo en tiempo real
- ✅ Visualización de experimentos y su impacto

**¡Comienza a inyectar chaos y observa todo en tiempo real!** 🚀
