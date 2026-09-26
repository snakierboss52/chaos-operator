# Arquitectura del Operador de Chaos Engineering

## Visión General

Este operador de Kubernetes implementa principios de Chaos Engineering para validar la resiliencia y confiabilidad de aplicaciones cloud-native. El operador permite inyectar diferentes tipos de fallos controlados en el cluster para identificar debilidades antes de que se conviertan en incidentes de producción.

## Principios de Diseño

### 1. Cloud-Native First
- Diseñado específicamente para entornos Kubernetes
- Utiliza Custom Resource Definitions (CRDs) para definir experimentos de chaos
- Integración nativa con la API de Kubernetes
- Observabilidad completa mediante logs estructurados y métricas

### 2. Seguridad por Diseño
- Principio de privilegio mínimo mediante RBAC
- Validación exhaustiva de recursos antes de ejecución
- Capacidades de rollback automático
- Límites de blast radius (radio de explosión) configurables

### 3. Clean Architecture
- Separación clara de concerns mediante capas
- Domain-Driven Design para lógica de negocio
- Dependency Injection para testabilidad
- Interfaces bien definidas entre componentes

## Arquitectura de Componentes

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes API Server                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            │ Watch/List
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                    Chaos Operator Manager                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Controller Runtime Manager                   │  │
│  │  - Leader Election                                        │  │
│  │  - Metrics Server                                         │  │
│  │  - Health Probes                                          │  │
│  │  - Webhook Server (Admission Control)                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌───────────┐  ┌─────────────┐  ┌────────────┐               │
│  │  PodChaos │  │ NetworkChaos│  │StressChaos │  ...          │
│  │Controller │  │  Controller │  │ Controller │               │
│  └─────┬─────┘  └──────┬──────┘  └──────┬─────┘               │
│        │               │                │                       │
│        └───────────────┴────────────────┘                       │
│                        │                                         │
│        ┌───────────────▼──────────────────┐                    │
│        │    Chaos Execution Engine        │                    │
│        │  - Strategy Pattern              │                    │
│        │  - Factory Pattern               │                    │
│        │  - Command Pattern               │                    │
│        └───────────────┬──────────────────┘                    │
│                        │                                         │
│        ┌───────────────▼──────────────────┐                    │
│        │      Chaos Executors             │                    │
│        │  - Pod Killer                    │                    │
│        │  - Network Manipulator           │                    │
│        │  - Stress Injector               │                    │
│        │  - HTTP Fault Injector           │                    │
│        └──────────────────────────────────┘                    │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ Inject Chaos
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Target Workloads                              │
│  - Pods, Deployments, StatefulSets, etc.                        │
└──────────────────────────────────────────────────────────────────┘
```

## Capas de la Arquitectura

### Capa de API (api/)
Define los tipos de recursos personalizados (CRDs) que representan diferentes experimentos de chaos:
- `PodChaos` - Fallos a nivel de Pod
- `NetworkChaos` - Fallos de red
- `StressChaos` - Estrés de recursos
- `HTTPChaos` - Fallos HTTP
- `IOChaos` - Fallos de I/O
- `TimeChaos` - Desviaciones temporales

### Capa de Control (controllers/)
Implementa reconciliation loops para cada tipo de chaos:
- Observa cambios en CRDs
- Ejecuta lógica de reconciliación
- Actualiza status de recursos
- Maneja rollbacks y recuperación

### Capa de Dominio (internal/domain/)
Contiene la lógica de negocio pura:
- Modelos de dominio
- Reglas de negocio
- Validaciones
- Estrategias de ejecución

### Capa de Infraestructura (internal/infrastructure/)
Implementa interacciones con sistemas externos:
- Cliente de Kubernetes
- Ejecutores de chaos (inyección de fallos)
- Métricas y observabilidad
- Almacenamiento de estado

## Patrones de Diseño Implementados

### 1. Strategy Pattern
Define una familia de algoritmos (estrategias de chaos) intercambiables:
- `PodKillStrategy`
- `NetworkLatencyStrategy`
- `CPUStressStrategy`

### 2. Factory Pattern
Crea ejecutores de chaos basados en el tipo de experimento:
- `ChaosExecutorFactory` - Crea el ejecutor apropiado

### 3. Builder Pattern
Construye especificaciones complejas de chaos:
- `ChaosExperimentBuilder` - Configuración fluida de experimentos

### 4. Observer Pattern
Notifica cambios de estado:
- Controllers observan CRDs
- Webhooks validan cambios

### 5. Command Pattern
Encapsula operaciones de chaos como comandos:
- `ExecuteChaosCommand`
- `RollbackChaosCommand`

### 6. Singleton Pattern
Garantiza una única instancia del manager:
- Manager de controller-runtime

### 7. Repository Pattern
Abstrae acceso a datos:
- `ChaosExperimentRepository` - Acceso a CRDs

## Flujo de Ejecución

1. **Usuario crea un recurso de Chaos** (e.g., PodChaos)
2. **Webhook de validación** verifica la especificación
3. **Controller detecta el nuevo recurso** mediante watch
4. **Reconcile loop inicia**:
   - Valida pre-condiciones
   - Selecciona targets según selectores
   - Aplica limitaciones de blast radius
   - Ejecuta estrategia de chaos
   - Actualiza status del recurso
5. **Monitor continuo** durante la duración del experimento
6. **Cleanup y rollback** al finalizar o en caso de error

## Consideraciones de Seguridad

### RBAC
- ServiceAccount dedicado para el operador
- Permisos mínimos necesarios por tipo de chaos
- ClusterRole y RoleBinding específicos

### Blast Radius Control
- Límite máximo de pods afectados
- Namespaces permitidos/prohibidos
- Labels de protección (chaos.engineering.io/protected: "true")

### Admission Control
- ValidatingWebhook para validar experimentos
- MutatingWebhook para defaults seguros

## Observabilidad

### Logs Estructurados
- JSON format
- Niveles: Debug, Info, Warning, Error
- Contexto: namespace, name, kind

### Métricas (Prometheus)
- `chaos_experiments_total` - Total de experimentos
- `chaos_experiments_active` - Experimentos activos
- `chaos_experiments_duration_seconds` - Duración
- `chaos_targets_affected_total` - Targets afectados

### Health Checks
- `/healthz` - Liveness probe
- `/readyz` - Readiness probe
- `/metrics` - Prometheus metrics

## Estrategia de Testing

### Unit Tests
- Lógica de dominio pura
- Validaciones
- Estrategias de chaos

### Integration Tests
- Controller reconciliation
- Interacción con API de Kubernetes
- Webhook validation

### End-to-End Tests
- Experimentos completos en cluster real
- Validación de efectos
- Rollback scenarios

## Referencias

- [Principles of Chaos Engineering](https://principlesofchaos.org/)
- [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
