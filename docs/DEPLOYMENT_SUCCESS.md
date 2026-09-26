# 🎉 ¡Despliegue Exitoso del Operador!

El operador de Chaos Engineering ahora está corriendo en tu cluster de Kind con todas las métricas funcionando.

## ✅ Estado del Sistema

### Operador Desplegado en Cluster
- **Namespace**: `chaos-system`
- **Pod**: `chaos-operator-58f59d4475-gzk28`
- **Status**: Running ✅
- **Métricas**: Activas en puerto 8080 ✅

### Stack de Monitoreo
- **Prometheus**: Running en namespace `monitoring` ✅
- **Grafana**: Running en namespace `monitoring` ✅
- **Configuración**: Actualizada para scrapear `chaos-system` ✅

### Experimentos de Prueba
- **Namespace**: `chaos-demo`
- **Aplicación**: 3 pods de nginx
- **Experimento activo**: `kill-one-nginx` ✅

---

## 🔗 Acceso a las Interfaces

### Prometheus
```bash
# Port-forward Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
```
- **URL**: http://localhost:9090
- **Query de prueba**: `chaos_experiments_total`

### Grafana
```bash
# Port-forward Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000
```
- **URL**: http://localhost:3000
- **Usuario**: `admin`
- **Contraseña**: `admin`

### Script Rápido (Port-forwards simultáneos)
```bash
cd /Users/jlozanoarena/Documents/study/goland-operator/monitoring
./access-monitoring.sh
```

---

## 📊 Verificar que Todo Funciona

### 1. Ver Métricas en el Operador
```bash
# Desde dentro del cluster
kubectl exec -n monitoring $(kubectl get pod -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}') -- \
  wget -qO- http://chaos-operator-metrics.chaos-system.svc.cluster.local:8080/metrics | grep chaos_experiments_total
```

### 2. Ver en Prometheus
1. Abre http://localhost:9090 (después del port-forward)
2. Query: `chaos_experiments_total`
3. Presiona "Execute"
4. Deberías ver: `chaos_experiments_total{action="pod-kill",namespace="chaos-demo",type="PodChaos"} 1`

### 3. Ver en Grafana
1. Abre http://localhost:3000
2. Login con `admin`/`admin`
3. Dashboards → Browse
4. Importa manualmente el dashboard desde: `monitoring/chaos-dashboard.json`

---

## 🧪 Ejecutar Más Experimentos

### Experimento Simple
```bash
# Eliminar el anterior
kubectl delete podchaos kill-one-nginx -n chaos-demo

# Aplicar nuevo
kubectl apply -f test-podchaos.yaml

# Ver estado
kubectl get podchaos -n chaos-demo -w
```

### Ver Métricas en Tiempo Real
```bash
# Ver logs del operador
kubectl logs -n chaos-system -l app=chaos-operator -f

# Ver métricas en loop
while true; do
  kubectl exec -n monitoring $(kubectl get pod -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}') -- \
    wget -qO- http://chaos-operator-metrics.chaos-system.svc.cluster.local:8080/metrics 2>/dev/null | \
    grep "chaos_experiments_total\|chaos_experiments_active\|chaos_targets_affected" | grep -v "^#"
  sleep 5
  echo "---"
done
```

---

## 📈 Métricas Disponibles

### Counters
- `chaos_experiments_total` - Total de experimentos creados
- `chaos_targets_affected_total` - Total de pods afectados
- `chaos_experiments_status_total` - Experimentos por status (Completed/Failed)
- `chaos_execution_errors_total` - Errores de ejecución
- `chaos_blast_radius_limit_total` - Límites de blast radius alcanzados

### Gauges
- `chaos_experiments_active` - Experimentos actualmente activos

### Histograms
- `chaos_experiments_duration_seconds` - Duración de experimentos
- `chaos_reconciliation_duration_seconds` - Tiempo de reconciliación

---

## 🎯 Queries Útiles en Prometheus

```promql
# Total de experimentos
sum(chaos_experiments_total)

# Tasa de experimentos por minuto
rate(chaos_experiments_total[5m])

# Experimentos activos por fase
sum(chaos_experiments_active) by (phase)

# Pods afectados por namespace
sum(chaos_targets_affected_total) by (namespace)

# Duración promedio (segundos)
rate(chaos_experiments_duration_seconds_sum[5m]) / 
rate(chaos_experiments_duration_seconds_count[5m])

# Tasa de errores
rate(chaos_execution_errors_total[5m])

# Percentil 95 de duración
histogram_quantile(0.95, 
  sum(rate(chaos_experiments_duration_seconds_bucket[5m])) by (le, type)
)
```

---

## 🔧 Comandos de Gestión

### Ver Estado del Operador
```bash
# Pods
kubectl get pods -n chaos-system

# Logs
kubectl logs -n chaos-system -l app=chaos-operator -f

# Describe
kubectl describe deployment chaos-operator -n chaos-system

# Restart (si es necesario)
kubectl rollout restart deployment/chaos-operator -n chaos-system
```

### Ver Targets en Prometheus
1. Abre Prometheus: http://localhost:9090
2. Status → Targets
3. Busca el job `chaos-operator`
4. Deberías ver el endpoint del operador como "UP"

### Actualizar el Operador
```bash
# Si haces cambios en el código:
cd /Users/jlozanoarena/Documents/study/goland-operator

# 1. Rebuild la imagen
docker build -t goland-operator:latest .

# 2. Cargar en Kind
kind load docker-image goland-operator:latest --name chaos-test-cluster

# 3. Restart del deployment
kubectl rollout restart deployment/chaos-operator -n chaos-system

# 4. Verificar
kubectl rollout status deployment/chaos-operator -n chaos-system
```

---

## 🎨 Importar Dashboard en Grafana

1. **Abre Grafana**: http://localhost:3000
2. **Login**: admin / admin
3. **Menu → Dashboards → Import**
4. **Click "Upload JSON file"**
5. **Selecciona**: `monitoring/chaos-dashboard.json`
6. **O copia y pega el contenido del archivo**
7. **Click "Import"**

El dashboard incluye:
- KPIs principales (Total, Activos, Targets, Errores)
- Gráficos temporales de experimentos
- Distribución por fase y tipo
- Métricas de rendimiento
- Tabla de errores

---

## 🐛 Troubleshooting

### Prometheus no muestra métricas
1. **Verifica que el operador esté corriendo**:
   ```bash
   kubectl get pods -n chaos-system
   ```

2. **Verifica los targets en Prometheus**:
   - http://localhost:9090 → Status → Targets
   - El job `chaos-operator` debe estar "UP"

3. **Verifica que las métricas estén disponibles**:
   ```bash
   kubectl exec -n monitoring $(kubectl get pod -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}') -- \
     wget -qO- http://chaos-operator-metrics.chaos-system.svc.cluster.local:8080/metrics 2>/dev/null | head -50
   ```

4. **Si no funciona, reinicia Prometheus**:
   ```bash
   kubectl rollout restart deployment/prometheus -n monitoring
   ```

### El operador no inicia
1. **Ver logs**:
   ```bash
   kubectl logs -n chaos-system -l app=chaos-operator
   ```

2. **Verificar RBAC**:
   ```bash
   kubectl get clusterrole chaos-operator-role
   kubectl get clusterrolebinding chaos-operator-rolebinding
   ```

3. **Verificar la imagen**:
   ```bash
   kubectl describe pod -n chaos-system -l app=chaos-operator
   ```

### Dashboard no carga en Grafana
- El formato del JSON tiene un problema conocido
- **Solución**: Importar manualmente desde la UI de Grafana
- O crear paneles manualmente con las queries de arriba

---

## 🚀 Siguiente Nivel

### Ejecutar Diferentes Tipos de Chaos
```bash
# StressChaos - Estrés de CPU
kubectl apply -f samples/stresschaos/cpu-stress.yaml

# NetworkChaos - Latencia de red
kubectl apply -f samples/networkchaos/delay.yaml

# HTTPChaos - Errores HTTP
kubectl apply -f samples/httpchaos/abort-503.yaml
```

### Monitorear Múltiples Experimentos
```bash
# Ver todos los tipos
kubectl get podchaos,stresschaos,networkchaos,httpchaos -A

# Ver métricas agregadas
# En Prometheus: sum(chaos_experiments_total) by (type)
```

### Automatizar Experimentos
```bash
# Loop continuo de experimentos
while true; do
  kubectl delete podchaos kill-one-nginx -n chaos-demo 2>/dev/null || true
  sleep 5
  kubectl apply -f test-podchaos.yaml
  sleep 60
done
```

---

## 📚 Documentación

- **[README.md](README.md)**: Documentación principal
- **[MONITORING.md](MONITORING.md)**: Guía completa de métricas
- **[SETUP_COMPLETE.md](SETUP_COMPLETE.md)**: Configuración inicial
- **[QUICKSTART.md](QUICKSTART.md)**: Inicio rápido

---

## ✅ Checklist Final

- [x] Operador desplegado en cluster Kind
- [x] Métricas expuestas en puerto 8080
- [x] Prometheus configurado para scrapear chaos-system
- [x] Grafana accesible
- [x] Primer experimento ejecutado exitosamente
- [x] Métricas visibles en Prometheus
- [x] Sistema listo para producción

---

## 🎉 ¡Felicidades!

Tienes un sistema completo de Chaos Engineering funcionando con:
- ✅ Operador en Kubernetes
- ✅ Métricas de Prometheus
- ✅ Dashboards de Grafana
- ✅ Monitoreo en tiempo real
- ✅ Múltiples tipos de chaos disponibles

**¡Comienza a inyectar chaos y mejora la resiliencia de tus aplicaciones!** 🚀
