# Config - Manifiestos de Kubernetes

Este directorio contiene todos los manifiestos de Kubernetes necesarios para desplegar el Chaos Operator.

## 📂 Estructura

```
config/
├── crd/                    # Custom Resource Definitions (generados automáticamente)
│   ├── chaos.engineering.io_podchaos.yaml
│   ├── chaos.engineering.io_networkchaos.yaml
│   ├── chaos.engineering.io_stresschaos.yaml
│   └── chaos.engineering.io_httpchaos.yaml
├── rbac/                   # RBAC configuration
│   ├── service_account.yaml
│   ├── role.yaml
│   └── role_binding.yaml
├── manager/                # Operator deployment
│   ├── namespace.yaml
│   ├── deployment.yaml
│   └── service.yaml
├── default/                # Kustomize base configuration
│   └── kustomization.yaml
└── overlays/               # Environment-specific configurations
    ├── development/
    │   └── kustomization.yaml
    └── production/
        └── kustomization.yaml
```

## 🚀 Uso

### Instalación Directa

```bash
# Instalar CRDs
kubectl apply -f crd/

# Instalar RBAC y Deployment
kubectl apply -f manager/namespace.yaml
kubectl apply -f rbac/
kubectl apply -f manager/deployment.yaml
kubectl apply -f manager/service.yaml
```

### Usando Kustomize

**Desarrollo:**
```bash
kubectl apply -k overlays/development/
```

**Producción:**
```bash
kubectl apply -k overlays/production/
```

### Usando Make (Recomendado)

```bash
# Instalar todo
make install-crds
make deploy

# Desinstalar
make undeploy
make uninstall-crds
```

## 🔧 Personalización

### Cambiar la imagen del operador

Edita `manager/deployment.yaml`:

```yaml
containers:
  - name: manager
    image: your-registry/goland-operator:v1.0.0  # Cambiar aquí
```

O usa kustomize en `overlays/production/kustomization.yaml`:

```yaml
images:
  - name: goland-operator
    newName: your-registry/goland-operator
    newTag: v1.0.0
```

### Ajustar recursos

Edita `manager/deployment.yaml`:

```yaml
resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 256Mi
```

### Configurar replicas (Alta Disponibilidad)

Para producción con múltiples replicas:

```yaml
# En manager/deployment.yaml
spec:
  replicas: 3  # Cambiar de 1 a 3
```

El operador usa leader election automáticamente cuando `ENABLE_LEADER_ELECTION=true`.

### Agregar variables de entorno

En `manager/deployment.yaml`:

```yaml
env:
  - name: METRICS_ADDR
    value: ":8080"
  - name: HEALTH_PROBE_ADDR
    value: ":8081"
  - name: ENABLE_LEADER_ELECTION
    value: "true"
  - name: LOG_LEVEL  # Nueva variable
    value: "debug"
```

## 🔒 RBAC

El operador requiere los siguientes permisos:

- **Chaos CRDs**: Permiso completo sobre todos los recursos de chaos
- **Pods**: Get, List, Watch, Delete (para PodChaos)
- **Pods/exec**: Create, Get (para StressChaos y NetworkChaos)
- **Deployments/ReplicaSets/StatefulSets**: Get, List, Watch
- **Events**: Create, Patch (para registrar eventos)
- **Leases**: Todos (para leader election)

Ver `rbac/role.yaml` para la lista completa.

## 📝 Notas

- Los CRDs en `crd/` son **generados automáticamente** desde los tipos Go
- No edites los CRDs manualmente, usa `make generate-crds`
- El namespace por defecto es `chaos-system`
- El ServiceAccount usado es `chaos-operator`
