# Chaos Engineering Operator for Kubernetes

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.20+-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Operador de Kubernetes cloud-native para ejecutar experimentos de Chaos Engineering, validar resiliencia de aplicaciones, y mejorar la confiabilidad de sistemas distribuidos.

## 🎯 Características

- **Múltiples Tipos de Chaos**: PodChaos, NetworkChaos, StressChaos, HTTPChaos
- **Control de Blast Radius**: Límites configurables para experimentos seguros
- **Selectores Flexibles**: Múltiples modos de selección de targets
- **Scheduling**: Experimentos recurrentes con cron
- **Observabilidad Completa**: Métricas Prometheus, logs estructurados, health checks
- **Clean Architecture**: Código modular, testeable y mantenible
- **Patrones de Diseño**: Strategy, Factory, Builder, Repository patterns
- **Seguridad por Diseño**: RBAC, validación exhaustiva, privilegios mínimos

## 📋 Tipos de Chaos Soportados

### PodChaos
Inyecta fallos a nivel de Pod para simular terminaciones, crashes y reinicios.

**Acciones**:
- `pod-kill`: Elimina pods
- `pod-failure`: Simula fallos de pod
- `container-kill`: Elimina contenedores específicos

**Ejemplo**:
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: kill-one-pod
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  action: pod-kill
  duration: "30s"
```

### NetworkChaos
Inyecta fallos de red: latencia, pérdida de paquetes, particiones.

**Acciones**:
- `delay`: Añade latencia de red
- `loss`: Causa pérdida de paquetes
- `duplicate`: Duplica paquetes
- `corrupt`: Corrompe paquetes
- `partition`: Crea particiones de red
- `bandwidth`: Limita ancho de banda

**Ejemplo**:
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: network-delay
spec:
  mode: all
  selector:
    labelSelectors:
      app: frontend
  action: delay
  delay:
    latency: "100ms"
    jitter: "10ms"
  duration: "10m"
```

### StressChaos
Genera estrés artificial en CPU y memoria.

**Tipos**:
- CPU stress con workers configurables
- Memory stress con tamaño configurable
- Estrés combinado

**Ejemplo**:
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: StressChaos
metadata:
  name: cpu-stress
spec:
  mode: one
  selector:
    labelSelectors:
      app: api-server
  stressors:
    cpu:
      workers: 2
      load: 80
  duration: "3m"
```

### HTTPChaos
Inyecta fallos HTTP: errores, latencias, modificación de contenido.

**Acciones**:
- `abort`: Aborta requests con códigos de error
- `delay`: Añade latencia a requests/responses
- `replace`: Reemplaza contenido HTTP
- `patch`: Modifica headers/body

**Ejemplo**:
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: HTTPChaos
metadata:
  name: http-503
spec:
  mode: all
  selector:
    labelSelectors:
      app: api-gateway
  target: Request
  port: 8080
  abort:
    statusCode: 503
    percentage: 25
  duration: "5m"
```

## 🚀 Quick Start

### Prerequisitos

- Kubernetes cluster v1.20+ (minikube, kind, GKE, EKS, AKS)
- kubectl configurado
- Docker
- Go 1.22+ (para desarrollo)

### Instalación Rápida (Recomendado)

**Instalación con un comando:**

```bash
./install.sh
```

El script detecta automáticamente tu tipo de cluster y configura todo. Ver [QUICKSTART.md](QUICKSTART.md) para más detalles.

**Instalación Manual:**

```bash
# 1. Generar CRDs
make generate-crds

# 2. Construir imagen Docker
make docker-build

# 3. Instalar CRDs
make install-crds

# 4. Desplegar operador
make deploy

# 5. Verificar instalación
kubectl get pods -n chaos-system
```

📚 **Guía completa de instalación**: [INSTALLATION.md](INSTALLATION.md)

### Primer Experimento

1. **Crear aplicación de prueba**:
```bash
kubectl create deployment nginx --image=nginx --replicas=3
kubectl label deployment nginx app=nginx
```

2. **Aplicar experimento**:
```bash
kubectl apply -f samples/podchaos/basic-kill-one.yaml
```

3. **Observar**:
```bash
kubectl get pods -w
kubectl get podchaos
kubectl describe podchaos kill-one-pod
```

## 📁 Estructura del Proyecto

```
goland-operator/
├── api/v1alpha1/              # CRD type definitions
│   ├── groupversion_info.go
│   ├── common_types.go
│   ├── podchaos_types.go
│   ├── networkchaos_types.go
│   ├── stresschaos_types.go
│   └── httpchaos_types.go
├── controllers/               # Reconcilers
│   └── podchaos_controller.go
├── internal/
│   ├── app/                  # Application initialization
│   ├── domain/               # Business logic (clean architecture)
│   │   ├── chaos_executor.go
│   │   └── target_selector.go
│   └── infrastructure/       # External integrations
│       └── kubernetes_pod_operations.go
├── cmd/manager/              # Entry point
├── config/                   # Kubernetes manifests
├── docs/                     # Documentation
│   ├── ARCHITECTURE.md       # Arquitectura del sistema
│   ├── CHAOS_TYPES.md        # Tipos de chaos detallados
│   ├── GETTING_STARTED.md    # Guía de inicio
│   └── DESIGN_PATTERNS.md    # Patrones implementados
├── samples/                  # Ejemplos YAML
│   ├── podchaos/
│   ├── networkchaos/
│   ├── stresschaos/
│   └── httpchaos/
└── pkg/                      # Shared packages
    └── version/
```

## 🏗️ Arquitectura

El operador sigue principios de **Clean Architecture** y **Domain-Driven Design**:

### Capas

1. **Domain Layer** (`internal/domain/`): Lógica de negocio pura
   - ChaosExecutor interfaces (Strategy Pattern)
   - TargetSelector (múltiples estrategias de selección)
   - Business rules y validaciones

2. **Application Layer** (`internal/app/`): Inicialización y orquestación
   - Manager setup
   - Controller registration
   - Configuration

3. **Infrastructure Layer** (`internal/infrastructure/`): Integraciones externas
   - Kubernetes client operations
   - Metrics y observabilidad
   - External systems

4. **Interface Layer** (`controllers/`): Adaptadores
   - Reconciliation loops
   - Event handling
   - Status updates

### Patrones de Diseño

- **Strategy Pattern**: Diferentes estrategias de selección y ejecución
- **Factory Pattern**: Creación de ejecutores de chaos
- **Builder Pattern**: Construcción fluida de experimentos
- **Repository Pattern**: Abstracción de acceso a datos
- **Observer Pattern**: Reconciliation loops (controller-runtime)

Ver [DESIGN_PATTERNS.md](docs/DESIGN_PATTERNS.md) para detalles completos.

## 🔧 Desarrollo

### Build

```bash
# Compilar
make build

# Ejecutar tests
make test

# Ejecutar linting
make lint

# Generar CRDs
make manifests

# Generar código (deepcopy, etc.)
make generate
```

### Testing

```bash
# Unit tests
go test ./internal/domain/... -v

# Integration tests (requiere envtest)
go test ./controllers/... -v

# Coverage
make test-coverage
```

## 📊 Observabilidad

### Métricas Prometheus

El operador expone métricas en `:8080/metrics`:

```
# Experimentos totales
chaos_experiments_total{type="PodChaos",namespace="default"}

# Experimentos activos
chaos_experiments_active{type="NetworkChaos"}

# Duración de experimentos
chaos_experiments_duration_seconds{type="StressChaos"}

# Targets afectados
chaos_targets_affected_total{type="HTTPChaos"}
```

### Logs Estructurados

Logs en formato JSON con contexto:
```json
{
  "level": "info",
  "ts": "2025-10-05T19:36:37-05:00",
  "msg": "Executing PodChaos",
  "podchaos": "kill-one-pod",
  "namespace": "default",
  "targets": 1
}
```

### Health Checks

- `/healthz`: Liveness probe
- `/readyz`: Readiness probe

## 🔒 Seguridad

### RBAC

El operador requiere permisos mínimos:
- Leer/escribir CRDs de chaos
- Leer pods, deployments, services
- Eliminar pods (para PodChaos)
- Exec en pods (para StressChaos, NetworkChaos)

### Blast Radius Control

Límites de seguridad por defecto:
- Máximo 50% de pods (configurable)
- Respeta PodDisruptionBudgets
- Namespaces protegidos (kube-system, etc.)

### Pod Security

- Run as non-root
- Read-only root filesystem
- No privilege escalation
- Capabilities dropped

## 📚 Documentación

- **[Getting Started](docs/GETTING_STARTED.md)**: Guía completa de inicio
- **[Architecture](docs/ARCHITECTURE.md)**: Arquitectura detallada del sistema
- **[Chaos Types](docs/CHAOS_TYPES.md)**: Descripción completa de tipos de chaos
- **[Design Patterns](docs/DESIGN_PATTERNS.md)**: Patrones implementados y mejores prácticas
- **[Samples](samples/README.md)**: Ejemplos YAML de experimentos

## 🤝 Mejores Prácticas

### 1. Comienza Pequeño
```yaml
spec:
  mode: one  # Un solo pod
  duration: "1m"  # Duración corta
```

### 2. Define Blast Radius
```yaml
spec:
  blastRadius:
    maxPods: 3
    maxPercentage: 25
```

### 3. Usa Selectores Específicos
```yaml
spec:
  selector:
    namespaces:
      - production
    labelSelectors:
      app: my-app
      version: v2
      chaos.engineering.io/protected: "false"
```

### 4. Monitorea Activamente
- Observa logs de aplicación
- Monitorea métricas de sistema
- Define alertas apropiadas

### 5. Documenta Experimentos
- Registra hipótesis
- Documenta resultados
- Comparte learnings

## 🎓 Principios de Chaos Engineering

1. **Define Steady State**: Establece comportamiento normal del sistema
2. **Hypothesize**: Formula hipótesis sobre comportamiento bajo fallo
3. **Run Experiment**: Inyecta chaos en producción
4. **Verify**: Comprueba si el sistema mantiene steady state
5. **Learn & Improve**: Itera basado en resultados

Ver [Principles of Chaos Engineering](https://principlesofchaos.org/)

## 🛠️ Comandos Útiles

```bash
# Listar todos los experimentos
kubectl get podchaos,networkchaos,stresschaos,httpchaos --all-namespaces

# Ver estado de un experimento
kubectl describe podchaos <name> -n <namespace>

# Ver logs del operador
kubectl logs -n chaos-system -l app=chaos-operator -f

# Port-forward métricas
kubectl port-forward -n chaos-system svc/chaos-operator 8080:8080

# Eliminar todos los experimentos
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all -n <namespace>
```

## 🐛 Troubleshooting

### Experimento en Estado Pending

```bash
# Verificar selector
kubectl get pods -n <namespace> --show-labels

# Ver eventos
kubectl describe podchaos <name> -n <namespace>

# Revisar logs del operador
kubectl logs -n chaos-system -l app=chaos-operator
```

### No Hay Pods Afectados

Verifica:
1. Labels del selector coinciden con pods
2. Namespace correcto
3. Blast radius no es demasiado restrictivo
4. Pods no están protegidos

## 📄 Licencia

Apache License 2.0 - ver [LICENSE](LICENSE)

## 🌟 Contribuir

Contribuciones son bienvenidas! Ver [CONTRIBUTING.md](CONTRIBUTING.md)

## 📮 Soporte

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Docs**: [docs/](docs/)

## 🙏 Referencias

- [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Principles of Chaos Engineering](https://principlesofchaos.org/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Chaos Mesh](https://chaos-mesh.org/)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
