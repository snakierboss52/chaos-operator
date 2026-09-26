# 🚀 Quick Start - Instalación en 5 Minutos

Esta guía te llevará desde cero hasta tener tu primer experimento de chaos corriendo.

## ⚡ Instalación con un comando

```bash
./install.sh
```

El script automáticamente:
- ✅ Verifica prerequisites
- ✅ Genera los CRDs
- ✅ Construye la imagen Docker
- ✅ Instala todo en tu cluster
- ✅ Detecta el tipo de cluster (minikube/kind/remoto)

## 📝 Pasos Manuales (Alternativa)

Si prefieres control total:

```bash
# 1. Generar CRDs
make generate-crds

# 2. Construir imagen
make docker-build

# 3. Instalar
make install-crds
make deploy

# Verificar
kubectl get pods -n chaos-system
```

## 🎯 Tu Primer Experimento

### 1. Crea una app de prueba

```bash
kubectl create deployment nginx --image=nginx --replicas=3
kubectl label deployment nginx app=nginx
```

### 2. Ejecuta un experimento simple

```bash
kubectl apply -f samples/podchaos/basic-kill-one.yaml
```

### 3. Observa qué sucede

```bash
# Ver pods (uno será eliminado)
kubectl get pods -w

# Ver el experimento
kubectl get podchaos

# Ver detalles
kubectl describe podchaos kill-one-pod
```

## 🔍 Verificación

### Ver el operador funcionando

```bash
# Status del operador
kubectl get deployment chaos-operator -n chaos-system

# Logs en tiempo real
kubectl logs -n chaos-system -l app=chaos-operator -f

# Métricas
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8080:8080
# Luego: curl http://localhost:8080/metrics
```

### Ver todos los tipos de chaos disponibles

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

## 📚 Ejemplos Disponibles

Explora más experimentos en el directorio `samples/`:

```bash
# PodChaos - Mata pods
samples/podchaos/
├── basic-kill-one.yaml          # Mata 1 pod
├── container-kill.yaml          # Mata un contenedor específico
└── kill-percentage.yaml         # Mata 25% de pods

# NetworkChaos - Inyecta fallos de red
samples/networkchaos/
├── delay.yaml                   # Añade latencia
├── packet-loss.yaml             # Pérdida de paquetes
└── partition.yaml               # Partición de red

# StressChaos - Estresa recursos
samples/stresschaos/
├── cpu-stress.yaml              # Estrés de CPU
├── memory-stress.yaml           # Estrés de memoria
└── combined-stress.yaml         # CPU + Memoria

# HTTPChaos - Fallos HTTP
samples/httpchaos/
├── abort-503.yaml               # Retorna 503
├── delay.yaml                   # Latencia HTTP
└── replace-response.yaml        # Modifica respuesta
```

## 🧪 Experimentos por Tipo de Cluster

### Minikube

```bash
# Inicia minikube
minikube start --cpus=4 --memory=8192

# Instala el operador
./install.sh

# Prueba un experimento
kubectl apply -f samples/podchaos/basic-kill-one.yaml
```

### Kind

```bash
# Crea un cluster
kind create cluster --name chaos-testing

# Instala el operador
./install.sh

# Prueba un experimento
kubectl apply -f samples/stresschaos/cpu-stress.yaml
```

### GKE/EKS/AKS (Clusters remotos)

```bash
# 1. Conecta a tu cluster
kubectl config use-context <your-cluster>

# 2. Build y push imagen
make docker-build IMG=<your-registry>/goland-operator:v1.0.0
make docker-push IMG=<your-registry>/goland-operator:v1.0.0

# 3. Actualiza el deployment
# Edita config/manager/deployment.yaml y cambia la imagen

# 4. Instala
make install-crds
make deploy
```

## 🛡️ Mejores Prácticas

### 1. Comienza pequeño

```yaml
spec:
  mode: one              # Solo 1 pod
  duration: "30s"        # Duración corta
```

### 2. Usa namespaces de prueba

```bash
kubectl create namespace chaos-test
# Aplica experimentos solo en ese namespace
```

### 3. Define límites de seguridad

```yaml
spec:
  blastRadius:
    maxPods: 2           # Máximo 2 pods
    maxPercentage: 25    # Máximo 25%
```

### 4. Monitorea activamente

```bash
# Terminal 1: Ver pods
kubectl get pods -w

# Terminal 2: Ver experimentos
kubectl get podchaos,networkchaos,stresschaos -w

# Terminal 3: Ver logs del operador
kubectl logs -n chaos-system -l app=chaos-operator -f
```

## 🐛 Solución Rápida de Problemas

| Problema | Solución |
|----------|----------|
| Operador no inicia | `kubectl logs -n chaos-system -l app=chaos-operator` |
| Imagen no encontrada (minikube) | `eval $(minikube docker-env) && make docker-build` |
| Imagen no encontrada (kind) | `make docker-build && kind load docker-image goland-operator:latest` |
| CRDs no se aplican | `make generate-crds && make install-crds` |
| Experimento en "Pending" | Verifica que los selectores coincidan con pods existentes |

## 🗑️ Desinstalación

```bash
# Opción 1: Script automático
./uninstall.sh

# Opción 2: Manual
make undeploy
make uninstall-crds
```

## 📖 Siguiente Nivel

- 📚 [Documentación completa](INSTALLATION.md)
- 🏗️ [Arquitectura del sistema](docs/ARCHITECTURE.md)
- 🎨 [Tipos de chaos detallados](docs/CHAOS_TYPES.md)
- 💡 [Patrones de diseño](docs/DESIGN_PATTERNS.md)

## ✅ Checklist de Instalación

- [ ] Prerequisites instalados (kubectl, docker, go)
- [ ] Cluster de Kubernetes funcionando
- [ ] CRDs generados
- [ ] Imagen Docker construida
- [ ] Operador desplegado en chaos-system namespace
- [ ] Pod del operador en estado "Running"
- [ ] Primer experimento ejecutado exitosamente

¡Listo para inyectar chaos! 🚀
