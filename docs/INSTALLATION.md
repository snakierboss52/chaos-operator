# Guía de Instalación del Chaos Operator

Esta guía te ayudará a instalar el operador de Chaos Engineering en tu cluster de Kubernetes.

## 📋 Prerequisitos

- Kubernetes cluster v1.20+ (minikube, kind, GKE, EKS, AKS, etc.)
- `kubectl` configurado y conectado a tu cluster
- Docker (para construir la imagen del operador)
- Go 1.22+ (para desarrollo)

### Verificar prerequisitos

```bash
# Verificar conexión al cluster
kubectl cluster-info

# Verificar versión de Kubernetes
kubectl version --short

# Verificar Docker
docker version
```

## 🚀 Instalación Rápida

### Opción 1: Instalación completa con un comando

```bash
make install-all
```

Este comando ejecutará automáticamente:
1. Generación de CRDs
2. Build de la imagen Docker
3. Instalación de CRDs en el cluster
4. Deploy del operador

### Opción 2: Instalación paso a paso

#### 1. Generar los CRDs

```bash
make generate-crds
```

Este comando generará los manifiestos de Custom Resource Definitions en `config/crd/`.

#### 2. Construir la imagen Docker

```bash
# Build con tag por defecto
make docker-build

# O especificar tu registry
make docker-build IMG=your-registry/goland-operator:v1.0.0
```

#### 3. (Opcional) Subir imagen a registry

Si estás usando un cluster remoto (GKE, EKS, AKS), necesitarás subir la imagen:

```bash
# Docker Hub
make docker-push IMG=your-dockerhub-user/goland-operator:v1.0.0

# Google Container Registry
make docker-push IMG=gcr.io/your-project/goland-operator:v1.0.0

# AWS ECR
make docker-push IMG=123456789.dkr.ecr.us-east-1.amazonaws.com/goland-operator:v1.0.0
```

Luego actualiza el deployment:
```bash
# Editar config/manager/deployment.yaml y cambiar la imagen
kubectl apply -f config/manager/deployment.yaml
```

#### 4. Instalar CRDs en el cluster

```bash
make install-crds
```

Verificar que los CRDs se instalaron correctamente:
```bash
kubectl get crds | grep chaos.engineering.io
```

Deberías ver:
```
httpchaos.chaos.engineering.io
networkchaos.chaos.engineering.io
podchaos.chaos.engineering.io
stresschaos.chaos.engineering.io
```

#### 5. Deploy del operador

```bash
make deploy
```

Este comando desplegará:
- Namespace `chaos-system`
- ServiceAccount
- ClusterRole y ClusterRoleBinding (RBAC)
- Deployment del operador
- Service para métricas

#### 6. Verificar instalación

```bash
# Ver el pod del operador
kubectl get pods -n chaos-system

# Ver logs del operador
kubectl logs -n chaos-system -l app=chaos-operator -f

# Verificar que el operador está listo
kubectl get deployment chaos-operator -n chaos-system
```

## 🧪 Probar la instalación

### 1. Crear una aplicación de prueba

```bash
kubectl create namespace test-chaos
kubectl create deployment nginx --image=nginx --replicas=3 -n test-chaos
kubectl label deployment nginx app=nginx -n test-chaos

# Esperar que los pods estén listos
kubectl wait --for=condition=ready pod -l app=nginx -n test-chaos --timeout=60s
```

### 2. Aplicar un experimento de chaos

```bash
cat <<EOF | kubectl apply -f -
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: test-kill-one
  namespace: test-chaos
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  action: pod-kill
  duration: "30s"
EOF
```

### 3. Observar el experimento

```bash
# Ver el recurso PodChaos
kubectl get podchaos -n test-chaos

# Ver los pods (uno será terminado y recreado)
kubectl get pods -n test-chaos -w

# Ver detalles del experimento
kubectl describe podchaos test-kill-one -n test-chaos
```

### 4. Limpiar

```bash
kubectl delete podchaos test-kill-one -n test-chaos
kubectl delete namespace test-chaos
```

## 🔧 Configuración Avanzada

### Cambiar recursos del operador

Edita `config/manager/deployment.yaml`:

```yaml
resources:
  limits:
    cpu: 1000m      # Aumentar CPU
    memory: 512Mi   # Aumentar memoria
  requests:
    cpu: 200m
    memory: 256Mi
```

Luego aplica los cambios:
```bash
kubectl apply -f config/manager/deployment.yaml
```

### Habilitar Leader Election

Para múltiples réplicas del operador:

```yaml
# En config/manager/deployment.yaml
replicas: 3  # Cambiar de 1 a 3

# La variable ENABLE_LEADER_ELECTION ya está en "true"
```

### Configurar métricas de Prometheus

Si tienes Prometheus Operator instalado:

```bash
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: chaos-operator
  namespace: chaos-system
spec:
  selector:
    matchLabels:
      app: chaos-operator
  endpoints:
  - port: metrics
    interval: 30s
EOF
```

## 🐛 Troubleshooting

### El operador no inicia

```bash
# Ver logs detallados
kubectl logs -n chaos-system -l app=chaos-operator --tail=100

# Ver eventos del pod
kubectl describe pod -n chaos-system -l app=chaos-operator

# Verificar permisos RBAC
kubectl auth can-i list pods --as=system:serviceaccount:chaos-system:chaos-operator
```

### CRDs no se instalan

```bash
# Verificar que los CRDs existen
ls config/crd/

# Re-generar CRDs
make generate-crds

# Re-instalar
make install-crds
```

### Imagen no encontrada

Si usas minikube o kind, carga la imagen localmente:

```bash
# Para minikube
eval $(minikube docker-env)
make docker-build
kubectl rollout restart deployment chaos-operator -n chaos-system

# Para kind, cargar la imagen construida y actualizar el deployment
make kind-load-image IMG=goland-operator:latest
make deploy IMG=goland-operator:latest
```

### Permisos insuficientes

Si ves errores de permisos:

```bash
# Verificar RBAC
kubectl get clusterrole chaos-operator-role -o yaml
kubectl get clusterrolebinding chaos-operator-rolebinding -o yaml

# Re-aplicar RBAC
kubectl apply -f config/rbac/
```

## 🔄 Actualización del operador

```bash
# 1. Build nueva imagen
make docker-build IMG=goland-operator:v2.0.0

# 2. Actualizar deployment
kubectl set image deployment/chaos-operator manager=goland-operator:v2.0.0 -n chaos-system

# 3. Verificar
kubectl rollout status deployment/chaos-operator -n chaos-system
```

## 🗑️ Desinstalación completa

```bash
# 1. Eliminar todos los experimentos de chaos
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all --all-namespaces

# 2. Desinstalar operador
make undeploy

# 3. Desinstalar CRDs
make uninstall-crds
```

## 📚 Siguientes pasos

- Lee la [documentación de tipos de chaos](docs/CHAOS_TYPES.md)
- Revisa los [ejemplos](samples/README.md)
- Consulta las [mejores prácticas](README.md#-mejores-prácticas)
- Explora la [arquitectura](docs/ARCHITECTURE.md)

## 🆘 Soporte

Si encuentras problemas:
1. Revisa esta guía y el [README.md](README.md)
2. Consulta los [logs del operador](#el-operador-no-inicia)
3. Abre un issue en GitHub
