# 📊 Dashboard Simple - Kubernetes Cluster Overview

Dashboard eficiente con solo 2 visualizaciones para monitorear tu cluster de Kubernetes.

## 🎯 Widgets del Dashboard

### 1️⃣ **Total de Pods en el Cluster** (Stat)
- **Métrica**: `count(kube_pod_info)`
- **Descripción**: Muestra el número total de pods corriendo en todo el cluster
- **Colores**:
  - 🟢 Verde: 0-49 pods
  - 🟡 Amarillo: 50-99 pods
  - 🔴 Rojo: 100+ pods
- **Visualización**: Número grande con gráfico de área

### 2️⃣ **Réplicas por Deployment** (Time Series)
- **Métricas**:
  - `kube_deployment_status_replicas` - Réplicas deseadas (línea azul punteada)
  - `kube_deployment_status_replicas_available` - Réplicas disponibles (línea verde)
  - `kube_deployment_status_replicas_unavailable` - Réplicas no disponibles (línea roja)
- **Descripción**: Gráfico de series temporales que muestra la evolución de réplicas
- **Leyenda**: Muestra namespace/deployment con valores Last, Max, Min
- **Refresh**: Cada 10 segundos
- **Rango**: Última hora (ajustable)

---

## 🚀 Cómo Acceder al Dashboard

### Paso 1: Port-forward a Grafana

```bash
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

### Paso 2: Abrir Grafana

1. Abre tu navegador en: **http://localhost:3000**
2. Login:
   - Usuario: `admin`
   - Contraseña: `admin`

### Paso 3: Importar el Dashboard

**Opción A - Importar desde archivo:**
1. Menu (☰) → Dashboards → Import
2. Click "Upload JSON file"
3. Selecciona: `/Users/jlozanoarena/Documents/study/goland-operator/monitoring/simple-dashboard.json`
4. Click "Import"

**Opción B - Debería aparecer automáticamente:**
1. Menu (☰) → Dashboards → Browse
2. Busca: "Kubernetes Cluster Overview"
3. Click para abrir

---

## 📊 Queries PromQL Utilizadas

### Total de Pods
```promql
count(kube_pod_info)
```
Esta query cuenta todos los pods en el cluster.

### Réplicas Deseadas
```promql
kube_deployment_status_replicas
```
Muestra cuántas réplicas debería tener cada deployment.

### Réplicas Disponibles
```promql
kube_deployment_status_replicas_available
```
Muestra cuántas réplicas están actualmente disponibles y funcionando.

### Réplicas No Disponibles
```promql
kube_deployment_status_replicas_unavailable
```
Muestra cuántas réplicas no están disponibles (en estado de error o iniciando).

---

## 🎨 Personalización

### Cambiar Colores

Edita el dashboard y ajusta los colores en:
- Panel → Edit → Overrides
- Selecciona la serie y cambia el color

### Cambiar Rango de Tiempo

Usa el selector de tiempo en la parte superior derecha:
- Last 5 minutes
- Last 15 minutes
- Last 1 hour
- Last 6 hours
- Last 24 hours
- Custom range

### Agregar Filtros

Puedes agregar variables para filtrar por namespace:

1. Dashboard settings (⚙️) → Variables
2. Add variable:
   - Name: `namespace`
   - Type: Query
   - Query: `label_values(kube_pod_info, namespace)`
3. Actualiza las queries para usar `{namespace="$namespace"}`

---

## 🔍 Verificar que las Métricas Funcionan

### En Prometheus

1. Abre Prometheus: **http://localhost:9090** (con port-forward)
2. Prueba estas queries:

```promql
# Ver todos los pods
kube_pod_info

# Ver deployments
kube_deployment_status_replicas

# Contar pods
count(kube_pod_info)
```

Si no ves datos, verifica que kube-state-metrics esté corriendo:

```bash
kubectl get pods -n monitoring -l app=kube-state-metrics
```

### Verificar Scraping

En Prometheus:
1. Status → Targets
2. Busca el job `kube-state-metrics`
3. Estado debe ser **"UP"** ✅

---

## 📊 Ejemplos de Uso

### Ver Impacto de Chaos Engineering

Cuando ejecutes experimentos de chaos:

```bash
# Ejecutar experimento que mata pods
kubectl apply -f test-podchaos.yaml

# Ver en el dashboard:
# - Widget 1: Total de pods bajará temporalmente
# - Widget 2: Réplicas disponibles bajará, luego subirá cuando Kubernetes las recree
```

### Monitorear Escalado

Cuando escales un deployment:

```bash
# Escalar deployment
kubectl scale deployment nginx --replicas=5 -n chaos-demo

# Ver en el dashboard:
# - Widget 1: Total de pods incrementará
# - Widget 2: Verás la línea azul (deseadas) subir a 5
# - Widget 2: Verás la línea verde (disponibles) subir gradualmente a 5
```

---

## 🎯 Ventajas de Este Dashboard

1. **Simple**: Solo 2 widgets, fácil de entender
2. **Eficiente**: Se actualiza cada 10 segundos
3. **Informativo**: Muestra información crítica del cluster
4. **Visual**: Gráfico de series temporales muestra tendencias
5. **Colorido**: Colores distintivos para cada estado

---

## 🔧 Troubleshooting

### No veo datos en el dashboard

1. **Verifica kube-state-metrics:**
   ```bash
   kubectl get pods -n monitoring -l app=kube-state-metrics
   kubectl logs -n monitoring -l app=kube-state-metrics
   ```

2. **Verifica Prometheus targets:**
   - http://localhost:9090 → Status → Targets
   - Busca `kube-state-metrics` y verifica que esté "UP"

3. **Prueba las queries en Prometheus:**
   ```promql
   kube_pod_info
   ```

4. **Reinicia Prometheus:**
   ```bash
   kubectl rollout restart deployment/prometheus -n monitoring
   ```

### El dashboard no aparece

1. **Verifica el ConfigMap:**
   ```bash
   kubectl get configmap grafana-simple-dashboard -n monitoring
   ```

2. **Reinicia Grafana:**
   ```bash
   kubectl rollout restart deployment/grafana -n monitoring
   ```

3. **Importa manualmente:**
   - Usa el archivo JSON directamente desde la UI de Grafana

---

## 📈 Métricas Adicionales Disponibles

Si quieres expandir el dashboard, puedes agregar:

```promql
# Pods por namespace
count(kube_pod_info) by (namespace)

# Pods por fase (Running, Pending, Failed)
count(kube_pod_status_phase) by (phase)

# CPU request por deployment
sum(kube_pod_container_resource_requests{resource="cpu"}) by (namespace, pod)

# Memoria request por deployment
sum(kube_pod_container_resource_requests{resource="memory"}) by (namespace, pod)

# Containers en estado waiting
count(kube_pod_container_status_waiting) by (namespace)

# Containers que han reiniciado
sum(kube_pod_container_status_restarts_total) by (namespace, pod)
```

---

## 🎓 Aprende Más

### PromQL Básico

- `count()` - Cuenta elementos
- `sum()` - Suma valores
- `by (label)` - Agrupa por etiqueta
- `rate()` - Calcula tasa por segundo
- `increase()` - Calcula incremento en un rango

### Grafana Tips

- **Shift + Click**: Selecciona múltiples series
- **Ctrl + S**: Guarda el dashboard
- **d + k**: Ver atajos de teclado
- **?**: Ayuda

---

## 🚀 Script de Acceso Rápido

```bash
#!/bin/bash
# Archivo: quick-dashboard.sh

echo "🚀 Accediendo a Grafana..."

# Port-forward
kubectl port-forward -n monitoring svc/grafana 3000:3000 > /dev/null 2>&1 &
GRAFANA_PID=$!

sleep 3

echo "✅ Grafana disponible en: http://localhost:3000"
echo "   Usuario: admin"
echo "   Contraseña: admin"
echo ""
echo "Dashboard: 'Kubernetes Cluster Overview'"
echo ""
echo "Presiona Ctrl+C para cerrar"

# Wait
trap "kill $GRAFANA_PID 2>/dev/null; exit 0" INT TERM
wait
```

---

¡Tu dashboard simple y eficiente está listo! 🎉
