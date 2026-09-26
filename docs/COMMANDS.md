# 📋 Referencia Rápida de Comandos

Todos los comandos útiles para trabajar con el Chaos Operator.

## 🚀 Instalación

```bash
# Instalación automática (recomendado)
./install.sh

# Instalación manual completa
make install-all

# Instalación paso a paso
make generate-crds     # Generar CRDs
make docker-build      # Construir imagen
make install-crds      # Instalar CRDs
make deploy            # Desplegar operador
```

## 🛠️ Desarrollo

```bash
# Build local
make build

# Ejecutar tests
make test

# Linting
make lint

# Actualizar dependencias
make tidy

# Ejecutar operador localmente (sin Docker)
make run

# Generar código deepcopy
make generate
```

## 🐳 Docker

```bash
# Build imagen
make docker-build

# Build con tag personalizado
make docker-build IMG=myregistry/chaos-operator:v1.0.0

# Push a registry
make docker-push IMG=myregistry/chaos-operator:v1.0.0

# Para Minikube (usar Docker de minikube)
eval $(minikube docker-env)
make docker-build

# Para Kind (cargar imagen al cluster)
make docker-build
kind load docker-image goland-operator:latest
```

## 📦 Gestión de CRDs

```bash
# Generar CRDs desde código Go
make generate-crds

# Instalar CRDs en cluster
make install-crds

# Ver CRDs instalados
kubectl get crds | grep chaos.engineering.io

# Desinstalar CRDs
make uninstall-crds

# Ver definición de un CRD
kubectl get crd podchaos.chaos.engineering.io -o yaml
```

## 🎯 Deploy del Operador

```bash
# Deploy en cluster actual
make deploy

# Deploy con imagen personalizada
make deploy IMG=myregistry/chaos-operator:v1.0.0

# Ver status del operador
kubectl get deployment chaos-operator -n chaos-system

# Ver pods del operador
kubectl get pods -n chaos-system

# Ver logs
kubectl logs -n chaos-system -l app=chaos-operator -f

# Restart del operador
kubectl rollout restart deployment/chaos-operator -n chaos-system

# Eliminar operador
make undeploy
```

## 🔍 Monitoreo y Debug

```bash
# Ver logs en tiempo real
kubectl logs -n chaos-system -l app=chaos-operator -f

# Ver logs con marca de tiempo
kubectl logs -n chaos-system -l app=chaos-operator --timestamps=true

# Ver logs de los últimos N minutos
kubectl logs -n chaos-system -l app=chaos-operator --since=5m

# Ver eventos del operador
kubectl get events -n chaos-system --sort-by='.lastTimestamp'

# Describir deployment
kubectl describe deployment chaos-operator -n chaos-system

# Ver métricas
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8080:8080
curl http://localhost:8080/metrics

# Health checks
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

## 🧪 Gestión de Experimentos

### Listar experimentos

```bash
# Listar todos los experimentos
kubectl get podchaos,networkchaos,stresschaos,httpchaos --all-namespaces

# Listar por tipo
kubectl get podchaos -A
kubectl get networkchaos -A
kubectl get stresschaos -A
kubectl get httpchaos -A

# Con detalles adicionales
kubectl get podchaos -o wide
```

### Aplicar experimentos

```bash
# Aplicar un experimento
kubectl apply -f samples/podchaos/basic-kill-one.yaml

# Aplicar todos los experimentos de un directorio
kubectl apply -f samples/podchaos/

# Aplicar con dry-run (validar sin aplicar)
kubectl apply -f samples/podchaos/basic-kill-one.yaml --dry-run=client
```

### Ver detalles de experimentos

```bash
# Ver detalles completos
kubectl describe podchaos <name> -n <namespace>

# Ver como YAML
kubectl get podchaos <name> -n <namespace> -o yaml

# Ver solo el status
kubectl get podchaos <name> -n <namespace> -o jsonpath='{.status}'

# Watch en tiempo real
kubectl get podchaos -w
```

### Eliminar experimentos

```bash
# Eliminar un experimento específico
kubectl delete podchaos <name> -n <namespace>

# Eliminar todos de un namespace
kubectl delete podchaos --all -n <namespace>

# Eliminar todos los tipos de chaos
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all -n <namespace>

# Eliminar de todos los namespaces (¡cuidado!)
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all --all-namespaces
```

## 🎨 Ejemplos de Uso

### PodChaos

```bash
# Matar un pod aleatorio
kubectl apply -f samples/podchaos/basic-kill-one.yaml

# Matar múltiples pods por porcentaje
kubectl apply -f samples/podchaos/kill-percentage.yaml

# Matar contenedor específico
kubectl apply -f samples/podchaos/container-kill.yaml
```

### NetworkChaos

```bash
# Añadir latencia de red
kubectl apply -f samples/networkchaos/delay.yaml

# Pérdida de paquetes
kubectl apply -f samples/networkchaos/packet-loss.yaml

# Partición de red
kubectl apply -f samples/networkchaos/partition.yaml
```

### StressChaos

```bash
# Estrés de CPU
kubectl apply -f samples/stresschaos/cpu-stress.yaml

# Estrés de memoria
kubectl apply -f samples/stresschaos/memory-stress.yaml

# Estrés combinado
kubectl apply -f samples/stresschaos/combined-stress.yaml
```

### HTTPChaos

```bash
# Retornar error 503
kubectl apply -f samples/httpchaos/abort-503.yaml

# Añadir latencia HTTP
kubectl apply -f samples/httpchaos/delay.yaml

# Modificar respuesta
kubectl apply -f samples/httpchaos/replace-response.yaml
```

## 🧹 Limpieza

```bash
# Desinstalación automática
./uninstall.sh

# Desinstalación manual
make undeploy
make uninstall-crds

# Limpiar todos los experimentos
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all --all-namespaces

# Limpiar namespace completo
kubectl delete namespace chaos-system
```

## 🔧 Troubleshooting

```bash
# Verificar que kubectl está conectado
kubectl cluster-info

# Ver todos los recursos en chaos-system
kubectl get all -n chaos-system

# Ver permisos del ServiceAccount
kubectl auth can-i list pods --as=system:serviceaccount:chaos-system:chaos-operator

# Ver todos los roles del operador
kubectl get clusterrole chaos-operator-role -o yaml

# Ver binding de roles
kubectl get clusterrolebinding chaos-operator-rolebinding -o yaml

# Reiniciar todo el operador
kubectl rollout restart deployment/chaos-operator -n chaos-system
kubectl rollout status deployment/chaos-operator -n chaos-system

# Verificar CRDs
kubectl get crd | grep chaos
kubectl describe crd podchaos.chaos.engineering.io

# Ver eventos del cluster
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | grep -i chaos
```

## 📊 Kustomize (Avanzado)

```bash
# Ver manifiestos sin aplicar (dry-run)
kubectl kustomize config/default

# Deploy usando kustomize directamente
kubectl apply -k config/default

# Deploy ambiente de desarrollo
kubectl apply -k config/overlays/development

# Deploy ambiente de producción
kubectl apply -k config/overlays/production

# Build manifiestos para review
kubectl kustomize config/overlays/production > manifests.yaml
```

## 🔄 Actualización del Operador

```bash
# 1. Build nueva versión
make docker-build IMG=goland-operator:v2.0.0

# 2. Push a registry (si es necesario)
make docker-push IMG=your-registry/goland-operator:v2.0.0

# 3. Actualizar imagen
kubectl set image deployment/chaos-operator \
  manager=goland-operator:v2.0.0 \
  -n chaos-system

# 4. Verificar actualización
kubectl rollout status deployment/chaos-operator -n chaos-system

# 5. Actualizar CRDs si hay cambios
make generate-crds
make install-crds
```

## 📝 Comandos de Utilidad

```bash
# Ver versión de Kubernetes
kubectl version --short

# Ver nodos del cluster
kubectl get nodes

# Ver uso de recursos
kubectl top nodes
kubectl top pods -n chaos-system

# Crear namespace de prueba
kubectl create namespace chaos-test

# Crear deployment de prueba
kubectl create deployment nginx --image=nginx --replicas=3 -n chaos-test
kubectl label deployment nginx app=nginx -n chaos-test

# Port-forward a un pod
kubectl port-forward -n chaos-system pod/<pod-name> 8080:8080
```

## 🎓 Comandos para Demos

```bash
# Setup rápido para demo
kubectl create namespace demo
kubectl create deployment app --image=nginx --replicas=5 -n demo
kubectl label deployment app app=demo-app -n demo

# Aplicar chaos
cat <<EOF | kubectl apply -f -
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: demo-chaos
  namespace: demo
spec:
  mode: one
  selector:
    labelSelectors:
      app: demo-app
  action: pod-kill
  duration: "30s"
EOF

# Watch en acción
kubectl get pods -n demo -w

# Cleanup demo
kubectl delete namespace demo
```

---

💡 **Tip**: Agrega estos aliases a tu shell para comandos más rápidos:

```bash
alias kgc='kubectl get podchaos,networkchaos,stresschaos,httpchaos'
alias kcl='kubectl logs -n chaos-system -l app=chaos-operator -f'
alias kgco='kubectl get pods -n chaos-system'
```
