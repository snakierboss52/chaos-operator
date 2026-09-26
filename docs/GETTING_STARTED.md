# Getting Started with Chaos Engineering Operator

## Tabla de Contenidos

- [Prerequisitos](#prerequisitos)
- [Instalación](#instalación)
- [Primeros Pasos](#primeros-pasos)
- [Ejemplos Básicos](#ejemplos-básicos)
- [Validación](#validación)
- [Troubleshooting](#troubleshooting)

## Prerequisitos

### Software Requerido

- **Kubernetes Cluster**: v1.20+
  - Minikube, Kind, o cluster real
- **kubectl**: Configurado para acceder al cluster
- **Go**: 1.22+ (para desarrollo)
- **Make**: Para ejecutar comandos de build

### Conocimientos

- Conocimientos básicos de Kubernetes (Pods, Deployments, Services)
- Familiaridad con conceptos de Chaos Engineering
- Entendimiento de YAML y manifiestos de Kubernetes

## Instalación

### Opción 1: Instalación Rápida con Manifiestos

```bash
# Clonar el repositorio
git clone https://github.com/your-org/goland-operator.git
cd goland-operator

# Instalar CRDs
kubectl apply -f config/crd/

# Instalar el operador
kubectl apply -f config/manager/

# Verificar instalación
kubectl get pods -n chaos-system
```

### Opción 2: Instalación Desde Código

```bash
# Clonar el repositorio
git clone https://github.com/your-org/goland-operator.git
cd goland-operator

# Descargar dependencias
go mod download

# Generar manifiestos
make manifests

# Instalar CRDs en el cluster
make install

# Ejecutar el operador localmente (para desarrollo)
make run

# O construir y desplegar en cluster
make docker-build docker-push IMG=your-registry/chaos-operator:v1.0.0
make deploy IMG=your-registry/chaos-operator:v1.0.0
```

### Opción 3: Helm Chart (Recomendado para Producción)

```bash
# Agregar el repositorio Helm
helm repo add chaos-operator https://your-org.github.io/goland-operator
helm repo update

# Instalar con valores por defecto
helm install chaos-operator chaos-operator/chaos-operator

# O con valores personalizados
helm install chaos-operator chaos-operator/chaos-operator \
  --set rbac.create=true \
  --set blastRadius.maxPercentageDefault=25
```

## Primeros Pasos

### 1. Verificar Instalación

```bash
# Verificar que los CRDs están instalados
kubectl get crds | grep chaos.engineering.io

# Deberías ver:
# httpchaos.chaos.engineering.io
# networkchaos.chaos.engineering.io
# podchaos.chaos.engineering.io
# stresschaos.chaos.engineering.io

# Verificar el operador está corriendo
kubectl get pods -n chaos-system
```

### 2. Preparar Aplicación de Prueba

Despliega una aplicación simple para testear:

```bash
kubectl create namespace demo-app

# Crear un deployment de ejemplo
kubectl create deployment nginx --image=nginx --replicas=3 -n demo-app
kubectl expose deployment nginx --port=80 -n demo-app

# Verificar pods
kubectl get pods -n demo-app
```

### 3. Tu Primer Experimento de Chaos

Crea un archivo `first-chaos.yaml`:

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: kill-one-nginx-pod
  namespace: demo-app
spec:
  mode: one
  selector:
    namespaces:
      - demo-app
    labelSelectors:
      app: nginx
  action: pod-kill
  duration: "30s"
```

Aplica el experimento:

```bash
kubectl apply -f first-chaos.yaml

# Observa qué sucede
kubectl get pods -n demo-app -w

# Verifica el estado del experimento
kubectl get podchaos -n demo-app
kubectl describe podchaos kill-one-nginx-pod -n demo-app
```

## Ejemplos Básicos

### Ejemplo 1: PodChaos - Eliminar Porcentaje de Pods

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: kill-25-percent
  namespace: production
spec:
  mode: fixed-percent
  value: "25"
  selector:
    namespaces:
      - production
    labelSelectors:
      app: backend-api
  action: pod-kill
  duration: "5m"
  blastRadius:
    maxPercentage: 30  # No más del 30%
```

### Ejemplo 2: NetworkChaos - Añadir Latencia

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: network-delay
  namespace: production
spec:
  mode: all
  selector:
    namespaces:
      - production
    labelSelectors:
      app: frontend
  action: delay
  direction: to
  delay:
    latency: "100ms"
    jitter: "10ms"
  target:
    selector:
      labelSelectors:
        app: backend-api
  duration: "10m"
```

### Ejemplo 3: StressChaos - Estrés de CPU

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: StressChaos
metadata:
  name: cpu-stress
  namespace: production
spec:
  mode: one
  selector:
    namespaces:
      - production
    labelSelectors:
      app: worker
  stressors:
    cpu:
      workers: 2
      load: 80
  duration: "3m"
```

### Ejemplo 4: HTTPChaos - Simular Errores 503

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: HTTPChaos
metadata:
  name: http-503-errors
  namespace: production
spec:
  mode: all
  selector:
    namespaces:
      - production
    labelSelectors:
      app: api-gateway
  target: Request
  port: 8080
  method: GET
  path: "/api/v1/*"
  abort:
    statusCode: 503
    percentage: 25
  duration: "5m"
```

## Validación

### Verificar Estado de Experimentos

```bash
# Listar todos los experimentos
kubectl get podchaos,networkchaos,stresschaos,httpchaos --all-namespaces

# Ver detalles de un experimento
kubectl describe podchaos <name> -n <namespace>

# Ver logs del operador
kubectl logs -n chaos-system -l app=chaos-operator -f
```

### Métricas y Observabilidad

El operador expone métricas Prometheus:

```bash
# Port-forward al puerto de métricas
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8080:8080

# Acceder a métricas
curl http://localhost:8080/metrics | grep chaos
```

Métricas disponibles:
- `chaos_experiments_total` - Total de experimentos ejecutados
- `chaos_experiments_active` - Experimentos actualmente activos
- `chaos_experiments_duration_seconds` - Duración de experimentos
- `chaos_targets_affected_total` - Total de targets afectados

## Mejores Prácticas Iniciales

### 1. Comienza en Entornos No Productivos

```yaml
# Usa namespaces dedicados para chaos
apiVersion: v1
kind: Namespace
metadata:
  name: chaos-testing
  labels:
    chaos.engineering.io/allowed: "true"
```

### 2. Define Blast Radius Conservador

```yaml
spec:
  blastRadius:
    maxPods: 3
    maxPercentage: 20
```

### 3. Empieza con Duraciones Cortas

```yaml
spec:
  duration: "1m"  # 1 minuto es suficiente para empezar
```

### 4. Usa Selectores Específicos

```yaml
spec:
  selector:
    namespaces:
      - specific-namespace
    labelSelectors:
      app: specific-app
      version: v1
      chaos.engineering.io/protected: "false"
```

### 5. Monitorea Activamente

```bash
# Terminal 1: Observa pods
kubectl get pods -n demo-app -w

# Terminal 2: Observa logs de la aplicación
kubectl logs -f deployment/your-app -n demo-app

# Terminal 3: Observa logs del operador
kubectl logs -f -n chaos-system -l app=chaos-operator
```

## Troubleshooting

### Problema: CRDs No se Instalan

```bash
# Verificar versión de Kubernetes
kubectl version

# Re-instalar CRDs
kubectl delete crd podchaos.chaos.engineering.io --ignore-not-found
make install
```

### Problema: Operador No Inicia

```bash
# Verificar logs
kubectl logs -n chaos-system -l app=chaos-operator

# Verificar RBAC
kubectl get clusterrole,clusterrolebinding | grep chaos

# Verificar recursos
kubectl describe pod -n chaos-system -l app=chaos-operator
```

### Problema: Experimento en Estado Pending

```bash
# Ver detalles del experimento
kubectl describe podchaos <name> -n <namespace>

# Verificar que hay pods que coinciden con el selector
kubectl get pods -n <namespace> -l <your-label-selector>

# Ver eventos
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
```

### Problema: No Hay Pods Afectados

Causas comunes:
1. **Selector incorrecto**: Verifica que los labels coincidan
2. **Namespace incorrecto**: Asegúrate de usar el namespace correcto
3. **Blast Radius muy restrictivo**: Aumenta los límites
4. **Pods protegidos**: Verifica labels de protección

```bash
# Debug del selector
kubectl get pods --show-labels -n <namespace>

# Verificar el status del chaos
kubectl get podchaos <name> -n <namespace> -o yaml
```

### Problema: Permissions Denied

```bash
# Verificar ServiceAccount
kubectl get sa -n chaos-system

# Verificar RoleBindings
kubectl describe clusterrolebinding chaos-operator-binding

# Re-aplicar RBAC
kubectl apply -f config/rbac/
```

## Próximos Pasos

1. **Explora Tipos de Chaos Avanzados**: Lee [CHAOS_TYPES.md](./CHAOS_TYPES.md)
2. **Entiende la Arquitectura**: Consulta [ARCHITECTURE.md](./ARCHITECTURE.md)
3. **Aprende Patrones Comunes**: Ve [PATTERNS.md](./PATTERNS.md)
4. **Integra con CI/CD**: Revisa [CI_CD_INTEGRATION.md](./CI_CD_INTEGRATION.md)
5. **Automatiza con Game Days**: Consulta [GAME_DAYS.md](./GAME_DAYS.md)

## Recursos Adicionales

- **Documentación Completa**: `docs/`
- **Ejemplos**: `samples/`
- **API Reference**: `docs/api/`
- **Community**: GitHub Discussions
- **Issues**: GitHub Issues

## Obtener Ayuda

- **Slack**: #chaos-engineering
- **Stack Overflow**: Tag `kubernetes-chaos-operator`
- **GitHub Issues**: Para bugs y feature requests
- **Email**: support@chaos-operator.io
