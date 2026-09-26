# Patrones de Diseño Implementados

## Visión General

Este documento describe los patrones de diseño implementados en el operador de Chaos Engineering, sus motivaciones, y cómo contribuyen a un código limpio, mantenible y extensible.

## Principios SOLID Aplicados

### Single Responsibility Principle (SRP)

Cada componente tiene una única razón para cambiar:

- **Controllers**: Solo manejan la reconciliación de recursos
- **Executors**: Solo ejecutan el chaos específico
- **Selectors**: Solo seleccionan targets
- **Operations**: Solo interactúan con Kubernetes API

**Ejemplo**:
```go
// ❌ Malo: Responsabilidades mezcladas
type PodChaosController struct {
    // Reconciliation + Execution + Selection
}

// ✅ Bueno: Responsabilidades separadas
type PodChaosReconciler struct {
    // Solo reconciliation
}

type PodChaosExecutor struct {
    // Solo execution
}

type TargetSelector struct {
    // Solo selection
}
```

### Open/Closed Principle (OCP)

El sistema está abierto para extensión pero cerrado para modificación:

```go
// Interface permite añadir nuevos ejecutores sin modificar código existente
type ChaosExecutor interface {
    Execute(ctx context.Context, targets []corev1.Pod) error
    Recover(ctx context.Context, targets []corev1.Pod) error
    Validate(ctx context.Context) error
    GetType() string
}

// Nuevos tipos de chaos solo requieren nueva implementación
type IOChaosExecutor struct {}
type TimeChaosExecutor struct {}
```

### Liskov Substitution Principle (LSP)

Cualquier implementación de ChaosExecutor puede reemplazar a otra:

```go
func runChaos(executor ChaosExecutor, targets []corev1.Pod) error {
    // Funciona con cualquier implementación de ChaosExecutor
    return executor.Execute(context.Background(), targets)
}
```

### Interface Segregation Principle (ISP)

Interfaces específicas en lugar de interfaces gordas:

```go
// ✅ Interfaces segregadas
type PodOperations interface {
    KillPod(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error
    FailPod(ctx context.Context, pod corev1.Pod) error
}

type NetworkOperations interface {
    ApplyDelay(ctx context.Context, pod corev1.Pod, spec *DelaySpec) error
    ApplyLoss(ctx context.Context, pod corev1.Pod, spec *LossSpec) error
}

// ❌ Malo: Interface monolítica
type AllOperations interface {
    KillPod(...)
    FailPod(...)
    ApplyDelay(...)
    ApplyLoss(...)
    InjectCPUStress(...)
    // ...100 métodos más
}
```

### Dependency Inversion Principle (DIP)

Dependemos de abstracciones, no de implementaciones concretas:

```go
// ✅ Bueno: Depende de interfaces
type PodChaosExecutor struct {
    Spec   *v1alpha1.PodChaosSpec
    client PodOperations  // Interface
}

// ❌ Malo: Depende de implementación concreta
type PodChaosExecutor struct {
    Spec   *v1alpha1.PodChaosSpec
    client *kubernetes.Clientset  // Implementación concreta
}
```

## Patrones de Diseño Gang of Four

### 1. Strategy Pattern

**Propósito**: Definir una familia de algoritmos, encapsular cada uno, y hacerlos intercambiables.

**Implementación**: Selección de targets con diferentes estrategias

```go
type ChaosMode string

const (
    OnePodMode           ChaosMode = "one"
    AllMode              ChaosMode = "all"
    FixedMode            ChaosMode = "fixed"
    FixedPercentMode     ChaosMode = "fixed-percent"
    RandomMaxPercentMode ChaosMode = "random-max-percent"
)

type TargetSelector struct {
    mode ChaosMode
    // ...
}

func (s *TargetSelector) SelectTargets(ctx context.Context) ([]corev1.Pod, error) {
    switch s.mode {
    case OnePodMode:
        return s.selectOne(candidates)
    case AllMode:
        return candidates
    case FixedMode:
        return s.selectFixed(candidates)
    // ...
    }
}
```

**Beneficios**:
- Fácil añadir nuevas estrategias de selección
- Algoritmos intercambiables en runtime
- Evita condicionales complejos

### 2. Factory Pattern

**Propósito**: Crear objetos sin especificar la clase exacta.

**Implementación**: Creación de ejecutores de chaos

```go
type ChaosExecutorFactory struct {
    podOps     PodOperations
    networkOps NetworkOperations
    stressOps  StressOperations
    httpOps    HTTPOperations
}

func (f *ChaosExecutorFactory) CreateExecutor(chaosType string, spec interface{}) (ChaosExecutor, error) {
    switch chaosType {
    case "PodChaos":
        return NewPodChaosExecutor(spec.(*v1alpha1.PodChaosSpec), f.podOps), nil
    case "NetworkChaos":
        return NewNetworkChaosExecutor(spec.(*v1alpha1.NetworkChaosSpec), f.networkOps), nil
    case "StressChaos":
        return NewStressChaosExecutor(spec.(*v1alpha1.StressChaosSpec), f.stressOps), nil
    case "HTTPChaos":
        return NewHTTPChaosExecutor(spec.(*v1alpha1.HTTPChaosSpec), f.httpOps), nil
    default:
        return nil, fmt.Errorf("unknown chaos type: %s", chaosType)
    }
}
```

**Beneficios**:
- Centraliza lógica de creación
- Fácil añadir nuevos tipos de chaos
- Desacopla código cliente de implementaciones concretas

### 3. Builder Pattern

**Propósito**: Construir objetos complejos paso a paso.

**Implementación**: Construcción de experimentos de chaos

```go
type ChaosExperimentBuilder struct {
    name       string
    namespace  string
    mode       ChaosMode
    selector   PodSelector
    duration   string
    blastRadius *BlastRadiusControl
}

func NewChaosExperimentBuilder() *ChaosExperimentBuilder {
    return &ChaosExperimentBuilder{
        mode: OnePodMode, // defaults
    }
}

func (b *ChaosExperimentBuilder) WithName(name string) *ChaosExperimentBuilder {
    b.name = name
    return b
}

func (b *ChaosExperimentBuilder) WithNamespace(ns string) *ChaosExperimentBuilder {
    b.namespace = ns
    return b
}

func (b *ChaosExperimentBuilder) WithMode(mode ChaosMode) *ChaosExperimentBuilder {
    b.mode = mode
    return b
}

func (b *ChaosExperimentBuilder) WithDuration(duration string) *ChaosExperimentBuilder {
    b.duration = duration
    return b
}

func (b *ChaosExperimentBuilder) WithBlastRadius(maxPods int32, maxPercent int32) *ChaosExperimentBuilder {
    b.blastRadius = &BlastRadiusControl{
        MaxPods:       &maxPods,
        MaxPercentage: &maxPercent,
    }
    return b
}

func (b *ChaosExperimentBuilder) Build() (*PodChaos, error) {
    if b.name == "" {
        return nil, fmt.Errorf("name is required")
    }
    // Validación y construcción
    return &PodChaos{
        ObjectMeta: metav1.ObjectMeta{
            Name:      b.name,
            Namespace: b.namespace,
        },
        Spec: PodChaosSpec{
            Mode:        b.mode,
            Selector:    b.selector,
            Duration:    b.duration,
            BlastRadius: b.blastRadius,
        },
    }, nil
}

// Uso
chaos, err := NewChaosExperimentBuilder().
    WithName("my-experiment").
    WithNamespace("production").
    WithMode(FixedPercentMode).
    WithDuration("5m").
    WithBlastRadius(10, 25).
    Build()
```

**Beneficios**:
- API fluida y legible
- Validación centralizada
- Valores por defecto manejados de forma consistente

### 4. Observer Pattern

**Propósito**: Definir dependencia uno-a-muchos para que cuando un objeto cambie de estado, todos sus dependientes sean notificados.

**Implementación**: Reconciliation loop de Kubernetes

```go
// Controller-runtime ya implementa Observer Pattern
// El controller "observa" cambios en recursos

func (r *PodChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&chaosv1alpha1.PodChaos{}).  // Observa PodChaos
        Owns(&corev1.Pod{}).              // Observa Pods owned
        Complete(r)
}

// El reconcile loop es llamado cuando hay cambios
func (r *PodChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Reacciona a cambios
}
```

### 5. Command Pattern

**Propósito**: Encapsular una request como un objeto.

**Implementación**: Comandos de chaos como objetos

```go
type ChaosCommand interface {
    Execute(ctx context.Context) error
    Undo(ctx context.Context) error
    GetName() string
}

type ExecuteChaosCommand struct {
    executor ChaosExecutor
    targets  []corev1.Pod
}

func (c *ExecuteChaosCommand) Execute(ctx context.Context) error {
    return c.executor.Execute(ctx, c.targets)
}

func (c *ExecuteChaosCommand) Undo(ctx context.Context) error {
    return c.executor.Recover(ctx, c.targets)
}

func (c *ExecuteChaosCommand) GetName() string {
    return fmt.Sprintf("Execute %s", c.executor.GetType())
}

// Command Invoker
type ChaosCommandInvoker struct {
    history []ChaosCommand
}

func (i *ChaosCommandInvoker) Execute(cmd ChaosCommand) error {
    if err := cmd.Execute(context.Background()); err != nil {
        return err
    }
    i.history = append(i.history, cmd)
    return nil
}

func (i *ChaosCommandInvoker) UndoLast() error {
    if len(i.history) == 0 {
        return fmt.Errorf("no commands to undo")
    }
    lastCmd := i.history[len(i.history)-1]
    i.history = i.history[:len(i.history)-1]
    return lastCmd.Undo(context.Background())
}
```

**Beneficios**:
- Historial de comandos ejecutados
- Soporte para undo/rollback
- Queuing y scheduling de comandos

### 6. Singleton Pattern

**Propósito**: Asegurar que una clase tiene solo una instancia.

**Implementación**: Manager de controller-runtime

```go
// controller-runtime Manager es un Singleton
var managerInstance ctrl.Manager
var once sync.Once

func GetManager() ctrl.Manager {
    once.Do(func() {
        managerInstance, _ = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
    })
    return managerInstance
}
```

### 7. Repository Pattern

**Propósito**: Abstrae la capa de acceso a datos.

**Implementación**: Acceso a recursos de Kubernetes

```go
type ChaosRepository interface {
    Get(ctx context.Context, name, namespace string) (*PodChaos, error)
    List(ctx context.Context, namespace string) ([]PodChaos, error)
    Create(ctx context.Context, chaos *PodChaos) error
    Update(ctx context.Context, chaos *PodChaos) error
    Delete(ctx context.Context, name, namespace string) error
}

type KubernetesChaosRepository struct {
    client client.Client
}

func (r *KubernetesChaosRepository) Get(ctx context.Context, name, namespace string) (*PodChaos, error) {
    var chaos PodChaos
    if err := r.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, &chaos); err != nil {
        return nil, err
    }
    return &chaos, nil
}

// Más métodos...
```

**Beneficios**:
- Aísla lógica de negocio de detalles de persistencia
- Facilita testing con repositories mock
- Cambia implementación sin afectar lógica de negocio

## Patrones Arquitecturales

### Hexagonal Architecture (Ports and Adapters)

```
┌─────────────────────────────────────────┐
│           Domain Layer                  │
│  - ChaosExecutor (interface)            │
│  - TargetSelector                       │
│  - Business Logic                       │
└─────────────┬───────────────────────────┘
              │
    ┌─────────┴──────────┐
    │                    │
┌───▼────────┐    ┌─────▼──────────┐
│ Controllers│    │ Infrastructure │
│ (Adapters) │    │   (Adapters)   │
│            │    │                │
│ - PodChaos │    │ - K8s Client   │
│ - Network  │    │ - Metrics      │
└────────────┘    └────────────────┘
```

**Capas**:

1. **Domain (Core)**: Lógica de negocio pura, sin dependencias externas
2. **Ports**: Interfaces que definen contratos
3. **Adapters**: Implementaciones concretas (controllers, infrastructure)

**Beneficios**:
- Testing independiente de infraestructura
- Fácil cambiar implementaciones
- Lógica de negocio protegida

### Clean Architecture

```
┌──────────────────────────────────────────────┐
│  Frameworks & Drivers (Controller-runtime)   │
└─────────────────┬────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────┐
│  Interface Adapters (Controllers)            │
└─────────────────┬────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────┐
│  Application Business Rules (Use Cases)      │
└─────────────────┬────────────────────────────┘
                  │
┌─────────────────▼────────────────────────────┐
│  Enterprise Business Rules (Domain)          │
└──────────────────────────────────────────────┘
```

**Principios**:
- Dependency Rule: Las dependencias apuntan hacia adentro
- Independencia de frameworks
- Testeable sin UI, DB, o servicios externos

## Anti-Patterns Evitados

### 1. God Object

❌ **Evitado**: Un objeto que hace todo

```go
// ❌ Malo
type ChaosManager struct {
    // 50 métodos
    // 100 dependencias
}
```

✅ **Aplicado**: Responsabilidades distribuidas

```go
// ✅ Bueno
type PodChaosReconciler struct { }
type TargetSelector struct { }
type PodChaosExecutor struct { }
```

### 2. Spaghetti Code

❌ **Evitado**: Código con muchas interdependencias

✅ **Aplicado**: 
- Interfaces bien definidas
- Dependency injection
- Separación de concerns

### 3. Golden Hammer

❌ **Evitado**: Usar el mismo patrón para todo

✅ **Aplicado**: Cada problema usa el patrón más apropiado

### 4. Cargo Cult Programming

❌ **Evitado**: Copiar código sin entender

✅ **Aplicado**: Cada patrón tiene justificación documentada

## Testing con Patrones de Diseño

### Mock Objects

```go
type MockPodOperations struct {
    KillPodFunc func(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error
}

func (m *MockPodOperations) KillPod(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error {
    if m.KillPodFunc != nil {
        return m.KillPodFunc(ctx, pod, gracePeriod)
    }
    return nil
}

// Test
func TestPodChaosExecutor(t *testing.T) {
    mockOps := &MockPodOperations{
        KillPodFunc: func(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error {
            // Verificar llamada
            return nil
        },
    }
    
    executor := NewPodChaosExecutor(spec, mockOps)
    err := executor.Execute(context.Background(), targets)
    
    assert.NoError(t, err)
}
```

### Table-Driven Tests

```go
func TestTargetSelector(t *testing.T) {
    tests := []struct {
        name        string
        mode        ChaosMode
        value       string
        candidates  int
        expectCount int
    }{
        {"one pod", OnePodMode, "", 10, 1},
        {"all pods", AllMode, "", 10, 10},
        {"fixed 3", FixedMode, "3", 10, 3},
        {"50 percent", FixedPercentMode, "50", 10, 5},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

## Conclusiones

Los patrones de diseño implementados en este operador proporcionan:

1. **Extensibilidad**: Fácil añadir nuevos tipos de chaos
2. **Mantenibilidad**: Código organizado y fácil de entender
3. **Testabilidad**: Componentes aislados y mockables
4. **Reusabilidad**: Componentes pueden usarse en diferentes contextos
5. **Robustez**: Manejo de errores consistente

## Referencias

- **Design Patterns: Elements of Reusable Object-Oriented Software** - Gang of Four
- **Clean Architecture** - Robert C. Martin
- **Domain-Driven Design** - Eric Evans
- **Patterns of Enterprise Application Architecture** - Martin Fowler
