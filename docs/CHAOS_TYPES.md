# Tipos de Chaos Engineering

## Visión General

Este documento describe los diferentes tipos de experimentos de chaos soportados por el operador, sus casos de uso, y mejores prácticas de implementación.

## 1. PodChaos

### Descripción
Inyecta fallos a nivel de Pod para simular terminaciones inesperadas, crashes, y reinicios.

### Tipos de Fallos

#### 1.1 Pod Kill
Elimina pods de forma aleatoria o determinística.

**Casos de Uso:**
- Validar que aplicaciones recuperan correctamente de fallos de pod
- Verificar configuración de livenessProbes y readinessProbes
- Testear políticas de restart
- Validar que no hay pérdida de datos en aplicaciones stateful

**Ejemplo de Especificación:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: kill-random-pods
  namespace: default
spec:
  action: pod-kill
  mode: random-one  # one, fixed, random-max-percent, fixed-percent
  selector:
    namespaces:
      - production
    labelSelectors:
      app: my-service
  duration: "30s"
  scheduler:
    cron: "@every 5m"
```

#### 1.2 Pod Failure
Hace que pods fallen sin eliminarlos (exit code != 0).

**Casos de Uso:**
- Simular fallos de aplicación
- Verificar logs y alertas
- Testear crash loop backoff

#### 1.3 Container Kill
Elimina containers específicos dentro de pods multi-container.

**Casos de Uso:**
- Testear sidecars (service mesh, logging agents)
- Validar comportamiento de init containers

### Parámetros Configurables
- `gracePeriod`: Tiempo de gracia antes de terminar (segundos)
- `force`: Forzar terminación sin grace period
- `targetContainer`: Container específico a terminar

### Estrategias de Selección

#### Mode: one
Selecciona exactamente un pod.

#### Mode: fixed
Selecciona un número fijo de pods (especificado en `value`).

#### Mode: random-max-percent
Selecciona hasta un porcentaje aleatorio (especificado en `value`).

#### Mode: fixed-percent
Selecciona un porcentaje fijo de pods.

### Limitaciones y Seguridad
- Máximo 50% de pods por defecto (configurable)
- Respeta PodDisruptionBudgets
- No afecta pods del sistema (kube-system, kube-public)

---

## 2. NetworkChaos

### Descripción
Inyecta fallos de red para simular condiciones adversas de red.

### Tipos de Fallos

#### 2.1 Network Delay
Añade latencia artificial a conexiones de red.

**Casos de Uso:**
- Simular redes lentas o saturadas
- Validar timeouts configurados
- Testear experiencia de usuario en condiciones degradadas

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: network-delay
spec:
  action: delay
  mode: all
  selector:
    labelSelectors:
      app: frontend
  delay:
    latency: "100ms"
    correlation: "25"  # % correlación entre paquetes
    jitter: "10ms"     # variación
  direction: to  # from, to, both
  target:
    selector:
      labelSelectors:
        app: backend
  duration: "5m"
```

#### 2.2 Network Loss
Descarta paquetes de forma aleatoria.

**Casos de Uso:**
- Simular conexiones inestables
- Validar protocolos de retry
- Testear circuit breakers

**Parámetros:**
- `loss`: Porcentaje de pérdida (0-100)
- `correlation`: Correlación entre paquetes perdidos

#### 2.3 Network Corruption
Corrompe paquetes introduciendo errores de bits.

**Casos de Uso:**
- Validar checksums y validaciones
- Testear detección de errores

#### 2.4 Network Partition
Crea particiones de red entre grupos de pods.

**Casos de Uso:**
- Simular split-brain scenarios
- Validar comportamiento en particiones de red
- Testear sistemas distribuidos y quorum

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: network-partition
spec:
  action: partition
  mode: all
  selector:
    labelSelectors:
      app: database
  direction: both
  target:
    selector:
      labelSelectors:
        app: application
  duration: "2m"
```

#### 2.5 Network Bandwidth Limitation
Limita el ancho de banda disponible.

**Casos de Uso:**
- Simular throttling
- Validar comportamiento bajo recursos de red limitados

### Direccionalidad
- `from`: Tráfico saliente desde el selector
- `to`: Tráfico entrante al selector
- `both`: Bidireccional

### Implementación Técnica
Utiliza:
- **tc (traffic control)** - Linux network shaping
- **iptables** - Filtrado de paquetes
- **Sidecar injection** - Para manipulación de tráfico (opcional)

---

## 3. StressChaos

### Descripción
Genera estrés artificial en recursos del sistema (CPU, memoria).

### Tipos de Fallos

#### 3.1 CPU Stress
Consume CPU para simular alta carga.

**Casos de Uso:**
- Validar límites de CPU (limits/requests)
- Testear throttling de CPU
- Verificar auto-scaling basado en CPU

**Ejemplo:**
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
      workers: 2      # Número de workers
      load: 80        # % carga por worker
  duration: "3m"
```

#### 3.2 Memory Stress
Consume memoria para simular memory leaks o alta utilización.

**Casos de Uso:**
- Validar límites de memoria
- Testear OOMKiller behavior
- Verificar memory leaks detection

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: StressChaos
metadata:
  name: memory-stress
spec:
  mode: fixed
  value: "2"
  selector:
    labelSelectors:
      app: worker
  stressors:
    memory:
      workers: 1
      size: "512MB"   # Cantidad de memoria
  duration: "2m"
```

### Parámetros Avanzados
- `oomScoreAdj`: Ajuste de prioridad OOM
- `options`: Opciones adicionales para stress-ng

### Implementación Técnica
Utiliza:
- **stress-ng** - Herramienta de stress testing
- **cgroups** - Límites de recursos

---

## 4. HTTPChaos

### Descripción
Inyecta fallos a nivel HTTP para simular errores de API, latencias, y comportamientos anómalos.

### Tipos de Fallos

#### 4.1 HTTP Abort
Aborta requests HTTP con códigos de error específicos.

**Casos de Uso:**
- Simular errores 500, 503, 404
- Validar retry logic
- Testear error handling

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: HTTPChaos
metadata:
  name: http-abort
spec:
  mode: random-one
  selector:
    labelSelectors:
      app: api-gateway
  target: Request
  port: 8080
  method: GET
  path: "/api/v1/*"
  abort:
    statusCode: 503
    percentage: 25  # % requests afectados
  duration: "5m"
```

#### 4.2 HTTP Delay
Introduce latencia en requests/responses HTTP.

**Casos de Uso:**
- Simular APIs lentas
- Validar timeouts
- Testear experiencia de usuario

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: HTTPChaos
metadata:
  name: http-delay
spec:
  mode: all
  selector:
    labelSelectors:
      app: payment-service
  target: Request
  port: 8080
  path: "/checkout"
  delay:
    latency: "2s"
    percentage: 50
  duration: "10m"
```

#### 4.3 HTTP Patch
Modifica cuerpo, headers, o status de HTTP.

**Casos de Uso:**
- Simular datos corruptos
- Validar sanitización de inputs
- Testear manejo de responses inesperados

### Targets
- `Request`: Afecta requests entrantes
- `Response`: Afecta responses salientes

### Implementación Técnica
- **Service Mesh Integration** (Istio, Linkerd)
- **HTTP Proxy Sidecar**
- **eBPF** para interceptación de tráfico

---

## 5. IOChaos

### Descripción
Inyecta fallos relacionados con I/O de disco.

### Tipos de Fallos

#### 5.1 IO Delay
Añade latencia a operaciones de lectura/escritura.

**Casos de Uso:**
- Simular discos lentos
- Validar configuraciones de storage
- Testear aplicaciones I/O intensive

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: IOChaos
metadata:
  name: io-delay
spec:
  mode: one
  selector:
    labelSelectors:
      app: database
  volumePath: /var/lib/mysql
  action: latency
  delay: "100ms"
  percentage: 50
  duration: "5m"
```

#### 5.2 IO Errno
Inyecta errores de sistema de archivos.

**Casos de Uso:**
- Simular disco lleno (ENOSPC)
- Simular I/O errors (EIO)
- Validar error handling

#### 5.3 IO Attribute Override
Modifica permisos o atributos de archivos.

**Casos de Uso:**
- Simular permission denied
- Testear manejo de permisos

### Implementación Técnica
- **FUSE (Filesystem in Userspace)**
- **System call interception**

---

## 6. TimeChaos

### Descripción
Manipula la percepción del tiempo del sistema para simular desfases de reloj.

### Tipos de Fallos

#### 6.1 Clock Skew
Desplaza el reloj del sistema adelante o atrás.

**Casos de Uso:**
- Validar expiración de certificados
- Testear drift de relojes en sistemas distribuidos
- Verificar sincronización NTP

**Ejemplo:**
```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: TimeChaos
metadata:
  name: time-skew
spec:
  mode: all
  selector:
    labelSelectors:
      app: time-sensitive-app
  timeOffset: "-2h"  # Atrasar 2 horas
  clockIds:
    - CLOCK_REALTIME
  duration: "10m"
```

### Implementación Técnica
- **libfaketime** - Intercepta llamadas de tiempo
- **LD_PRELOAD** - Inyección de librería

---

## Mejores Prácticas

### 1. Comenzar con Blast Radius Pequeño
- Empezar con `mode: one`
- Incrementar gradualmente el alcance

### 2. Definir Hipótesis Clara
Antes de ejecutar un experimento:
- ¿Qué esperamos que pase?
- ¿Cómo medimos el éxito?
- ¿Cuáles son los criterios de rollback?

### 3. Monitoreo Continuo
- Métricas de aplicación
- Logs agregados
- Alertas configuradas

### 4. Blast Radius Controls
```yaml
spec:
  selector:
    labelSelectors:
      app: my-app
      chaos.engineering.io/protected: "false"
  blastRadius:
    maxPods: 5
    maxPercentage: 25
```

### 5. Duración Apropiada
- Corta para experimentos iniciales (1-2 min)
- Incrementar gradualmente
- Considerar ritmos circadianos (evitar horas pico)

### 6. Documentación
- Documentar cada experimento
- Registrar resultados y learnings
- Compartir con el equipo

### 7. Automatización
- Integrar con CI/CD
- Chaos como parte de testing
- GameDays programados

---

## Matriz de Experimentos Recomendados

| Tipo de Aplicación | Experimentos Primarios | Experimentos Secundarios |
|-------------------|------------------------|--------------------------|
| **API REST** | HTTPChaos (abort, delay), NetworkChaos (delay) | PodChaos (kill), StressChaos (CPU) |
| **Database** | PodChaos (kill), NetworkChaos (partition) | IOChaos (latency), StressChaos (memory) |
| **Message Queue** | NetworkChaos (loss, delay), PodChaos (kill) | StressChaos (memory) |
| **Frontend SPA** | HTTPChaos (delay, abort) | NetworkChaos (bandwidth) |
| **Batch Jobs** | PodChaos (kill), StressChaos (CPU, memory) | IOChaos (errors) |
| **Microservices** | NetworkChaos (partition, delay), HTTPChaos (abort) | PodChaos (kill) |

---

## Referencias

- [Chaos Engineering Principles](https://principlesofchaos.org/)
- [Google SRE Book - Testing for Reliability](https://sre.google/sre-book/testing-reliability/)
- [Netflix Chaos Engineering](https://netflixtechblog.com/tagged/chaos-engineering)
- [Chaos Mesh Documentation](https://chaos-mesh.org/docs/)
