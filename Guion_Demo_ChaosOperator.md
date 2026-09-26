# Guion para grabación — Chaos Engineering Operator

**Proyecto:** goland-operator
**Duración objetivo:** 12–13 minutos (rango 10–15 min)
**Tono:** académico-conversacional, primera persona, ritmo pausado
**Convenciones del guion:**
- `[En pantalla: …]` → qué mostrar mientras hablás.
- `[Pausa]` → respirar y dejar respirar al espectador.
- *texto en cursiva* → énfasis al leer.
- Tiempos acumulados al inicio de cada bloque.

---

## 00:00 — Apertura (≈ 45 s)

[En pantalla: slide o terminal con el título "Chaos Engineering Operator para Kubernetes" y tu nombre.]

Hola. Soy Jorge Humberto Lozano Arenas, estudiante de Ingeniería de Sistemas en la Universidad Central, y en los próximos quince minutos quiero mostrarles el operador de ingeniería del caos para Kubernetes que desarrollé como proyecto de práctica.

La idea de este video es contarles *tres cosas*: primero, por qué decidí construir un operador desde cero en vez de usar una herramienta existente; segundo, cómo lo diseñé e implementé; y tercero, hacer una demo en vivo sobre un clúster local para que vean el comportamiento real, no slides bonitos.

[Pausa]

---

## 00:45 — ¿Qué es ingeniería del caos? (≈ 1 min 15 s)

La ingeniería del caos, por si alguno la escucha por primera vez, *no* es romper cosas porque sí. Es una disciplina formal —nacida en Netflix alrededor de 2010 con Chaos Monkey— que consiste en experimentar de forma controlada sobre un sistema para generar confianza en su capacidad de soportar condiciones turbulentas en producción.

La premisa es incómoda pero honesta: en un sistema distribuido las fallas son inevitables. En lugar de esperar a que ocurran un sábado a las tres de la mañana, las provocamos *nosotros* en horario laboral, con un alcance acotado, observabilidad encendida y un botón rojo de aborto a la mano. Si el sistema sobrevive, ganamos confianza. Si no, ya sabemos qué hay que reparar *antes* de que un cliente lo descubra.

Los cinco principios canónicos son: formular una hipótesis de estado estable, variar eventos del mundo real, ejecutar en condiciones lo más cercanas posibles a producción, automatizar para correr experimentos de forma continua, y *minimizar el radio de explosión*. Ese último principio es el más importante y va a aparecer varias veces en la demo.

---

## 02:00 — El problema y por qué un Operator (≈ 1 min 30 s)

[En pantalla: terminal con `kubectl get crds | grep chaos`.]

Hoy existen herramientas maduras como Chaos Mesh y LitmusChaos, pero la pregunta académica que me interesaba era: *¿cómo se construye una de estas por dentro?* ¿Qué patrones, qué garantías de seguridad, qué loop de control hay que escribir para integrar chaos engineering de forma nativa en Kubernetes?

La respuesta de la comunidad cloud-native a este tipo de problema es el patrón Operator. Un Operator extiende la API de Kubernetes con Custom Resource Definitions —recursos personalizados— y un controlador que reconcilia el estado deseado con el estado real. En vez de invocar scripts ad-hoc, uno declara *"quiero un experimento que mate uno de estos pods cada 30 segundos durante 5 minutos"* en YAML, y el operador se encarga del resto: selecciona los pods, ejecuta el experimento, expone métricas, emite eventos, limpia al final.

Esa abstracción declarativa es exactamente lo que necesita la ingeniería del caos: experimentos versionados en Git, ejecutables desde un pipeline CI, observables desde Prometheus y Grafana, y con garantías de cleanup integradas en el ciclo de vida del recurso.

[Pausa]

---

## 03:30 — Diseño y arquitectura (≈ 2 min 30 s)

[En pantalla: árbol de directorios del proyecto — `tree -L 2 -I 'bin|vendor'` o el README.]

El operador se llama `goland-operator` y está construido con `controller-runtime`, el SDK oficial del SIG API Machinery de Kubernetes.

Lo organicé siguiendo Clean Architecture, con cinco capas bien separadas:

- `cmd/manager` es el entry-point, sólo arma el binario.
- `api/v1alpha1` contiene los tipos de los CRDs y el código auto-generado de deep-copy.
- `controllers/` tiene los reconcilers, uno por cada CRD.
- `internal/domain` es la lógica de negocio pura, *sin importar nada de Kubernetes*. Ahí viven el selector de targets y los ejecutores de chaos.
- `internal/infrastructure` traduce la lógica de dominio a operaciones reales contra Kubernetes: matar pods, aplicar reglas `tc` para la red, inyectar stress de CPU.

Esta separación es importante porque el dominio no debería saber *cómo* se mata un pod —si por API, por `exec`, por SIGKILL—; solo debe describir el experimento. Eso hace los tests unitarios triviales: mockeo la infraestructura y verifico la decisión.

Definí *cuatro* CRDs en el API group `chaos.engineering.io/v1alpha1`: **PodChaos**, **NetworkChaos**, **StressChaos** y **HTTPChaos**. Cada uno cubre una familia de fallas: terminar pods o contenedores; introducir latencia, pérdida, particiones o limitar ancho de banda; saturar CPU, memoria, disco o I/O; y abortar, demorar, reemplazar o patchear respuestas HTTP. Es el mismo modelo que usa Chaos Mesh y resultó más cómodo que un único CRD genérico, porque cada tipo de falla tiene parámetros muy distintos.

Sobre los patrones de diseño: usé **Strategy Pattern** con una interfaz `ChaosExecutor` que tiene cuatro métodos —`Execute`, `Recover`, `Validate` y `GetType`— y una implementación por tipo de falla. Y un **Factory** que devuelve el ejecutor correcto según el CRD que esté reconciliando.

[Pausa]

---

## 06:00 — Implementación: el reconcile loop (≈ 2 min)

El corazón de cualquier Operator es el reconcile loop. En este caso, cada experimento atraviesa cuatro fases: `Pending` → `Running` → `Completed`, o `Failed` si algo sale mal. El controller-runtime me llama cada vez que cambia el CR o cuando le pido un re-encolado.

Tres decisiones de implementación que vale la pena destacar:

**Primero, finalizers.** Cuando el usuario hace `kubectl delete` de un experimento, *no* quiero que Kubernetes lo borre inmediatamente, porque puede haber reglas de red o procesos de stress corriendo en los pods objetivo. Entonces le pongo un finalizer al recurso; eso obliga a Kubernetes a esperar a que yo limpie el estado externo —llamando a `Recover()` del ejecutor— antes de eliminar el CR. Es la diferencia entre un cleanup confiable y dejar reglas `tc` huérfanas en producción.

**Segundo, webhooks.** Tengo un *mutating* webhook que asigna defaults seguros —si el usuario no especifica blast radius, le pongo `MaxPercentage=50`— y un *validating* webhook que rechaza specs malformadas antes de que lleguen al cluster: un `container-kill` sin `containerNames`, un `statusCode` HTTP fuera del rango 100–599, un `duration` que no parsea. Esto bloquea errores en el apply, no a las tres reconciliaciones.

**Tercero, blast radius enforcement.** El selector de targets *siempre* aplica los límites de `BlastRadius`. Aunque el usuario diga "modo all sobre 100 pods", si no configuró un máximo, el default es 50 %. Esa regla está en el dominio y existen tests unitarios y de seguridad que la verifican. Es la red de protección de toda la herramienta.

[Pausa]

---

## 08:00 — Demo en vivo (≈ 4 min)

[En pantalla: terminal limpia, cluster kind ya levantado.]

Vamos a la parte que más me gusta. Tengo un cluster `kind` de tres nodos corriendo localmente, con el operador y el stack de monitoring desplegados.

[En pantalla: `kubectl get nodes`]

Aquí están los tres nodos `Ready`.

[En pantalla: `kubectl get pods -n chaos-system`]

Este es el pod del operador, corriendo en su propio namespace. Si miramos el security context, está como non-root, con read-only filesystem y `allowPrivilegeEscalation: false` — todas las buenas prácticas de pod security.

[En pantalla: `kubectl get crds | grep chaos`]

Y aquí están los cuatro CRDs registrados.

Voy a usar como objetivo un Deployment llamado `nginx-target` que tiene cuatro réplicas en el namespace `chaos-demo`.

[En pantalla: `kubectl get pods -n chaos-demo`]

Cuatro pods Running. Ahora voy a aplicar un experimento de tipo PodChaos.

[En pantalla: `cat samples/podchaos/basic-kill-one.yaml`]

El YAML es muy corto: dice "modo `one`, action `pod-kill`, selector por label `app=nginx-target`, duration 30 segundos". *Nada más*. No estoy diciendo cómo matar el pod, solo qué quiero.

[En pantalla: `kubectl apply -f samples/podchaos/basic-kill-one.yaml`]

Aplico. Y ahora abro dos terminales: en una observo el CR; en la otra, los pods.

[En pantalla: split terminal con `kubectl get podchaos -n chaos-demo -w` arriba y `kubectl get pods -n chaos-demo -w` abajo.]

Vean la transición: el CR pasa de Pending a Running, se selecciona un pod aleatorio, lo mata, y el ReplicaSet de Kubernetes lo reemplaza automáticamente. Esa es la hipótesis: *el deployment debe sobrevivir la pérdida de una de sus réplicas*. Y sobrevive.

[En pantalla: `kubectl get events -n chaos-demo --sort-by='.lastTimestamp' | tail -15`]

Aquí están los eventos: `Initialized`, `Executing`, `WaveFired`, `Completed`. Esto es lo que verían en `kubectl describe` o en su sistema de observabilidad.

[En pantalla: `kubectl describe podchaos -n chaos-demo basic-kill-one`]

Y en el status del recurso ven `AffectedPods` con el namespace y nombre exactos del pod que se mató, `StartTime`, `CompletionTime`, y el `ExperimentResult` con los contadores.

[En pantalla: navegador con Grafana en `localhost:3000`.]

Y todo esto se está exportando como métricas Prometheus. En el dashboard ven `Active Experiments`, `Targets Affected`, `Reconciliation Duration p95`. Es el mismo conjunto de métricas que usan los SLOs del plan de pruebas de carga.

---

## 12:00 — Observabilidad y trazabilidad (≈ 1 min 30 s)

[En pantalla: Grafana o terminal con `kubectl logs deployment/chaos-operator -n chaos-system --tail=20`]

Quiero detenerme medio minuto en esto porque para mí es lo que diferencia un script de un *producto*. El operador expone:

- Métricas Prometheus en el puerto 8080: experimentos totales, activos, duración, pods afectados, errores de ejecución, y la duración del reconcile loop.
- Health probes en el 8081 —liveness y readiness— para que Kubernetes sepa cuándo reiniciarlo.
- Logs estructurados en JSON, con el namespace, el nombre del CR y el nivel en cada línea, listos para Loki o Elasticsearch.
- Eventos Kubernetes nativos en cada transición.
- Y un ConfigMap por experimento con el resumen final, útil para auditorías post-mortem.

Sumado al stack de Prometheus, Grafana e InfluxDB que viene en el repositorio bajo `monitoring/`, el usuario tiene visibilidad de extremo a extremo sin configurar nada extra.

[Pausa]

---

## 13:30 — Cierre (≈ 1 min)

Para cerrar: lo que construí es un operador cloud-native que convierte la práctica de chaos engineering en algo *declarativo, automatizable y observable*. Cuatro CRDs, un reconcile loop con finalizers, webhooks de validación, control de blast radius con un default seguro del 50 %, y un stack de observabilidad completo.

Está validado por un plan integral de pruebas con 123 casos —unitarios, de integración con `envtest`, end-to-end en `kind`, no funcionales y de carga con `k6`— que cubre el cien por ciento de los requerimientos funcionales y no funcionales.

¿Mi mayor aprendizaje? Que la ingeniería del caos *no* es destructiva: es una práctica de cuidado. Cada línea de código de este operador tiene como propósito último *proteger* al sistema bajo prueba mientras lo hacemos más resiliente. Por eso el blast radius es el primer ciudadano del dominio, por eso los finalizers son innegociables, y por eso los webhooks rechazan inputs ambiguos antes de tocar nada.

Gracias por acompañarme. Si tienen preguntas o quieren ver el código, el repositorio queda en mi perfil. Hasta la próxima.

[En pantalla: slide final con el nombre del proyecto, tu nombre y un QR o URL al repo.]

---

## Anexo — Comandos listos para copiar a la terminal de demo

```bash
# Estado del cluster
kubectl get nodes
kubectl get pods -n chaos-system
kubectl get crds | grep chaos.engineering.io
kubectl get pods -n chaos-demo

# Aplicar el experimento
cat samples/podchaos/basic-kill-one.yaml
kubectl apply -f samples/podchaos/basic-kill-one.yaml

# Observar (idealmente en split terminal)
kubectl get podchaos -n chaos-demo -w
kubectl get pods -n chaos-demo -w

# Eventos y status
kubectl get events -n chaos-demo --sort-by='.lastTimestamp' | tail -15
kubectl describe podchaos -n chaos-demo basic-kill-one

# Observabilidad
kubectl logs -n chaos-system deployment/chaos-operator --tail=20
kubectl port-forward -n monitoring svc/grafana 3000:3000   # usuario admin / pass admin
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Limpieza (cierre de demo)
kubectl delete -f samples/podchaos/basic-kill-one.yaml
```

## Anexo — Checklist pre-grabación

- [ ] Cluster `kind` arriba: `kind get clusters | grep chaos-testing-v2`.
- [ ] Operator Running: `kubectl get pods -n chaos-system` muestra 1/1 Ready.
- [ ] 4 réplicas Running en `chaos-demo`.
- [ ] Grafana abierto y logueado en el dashboard del operator.
- [ ] Terminal con fuente grande (16–18 pt) y prompt corto.
- [ ] Notificaciones del sistema en silencio.
- [ ] Micrófono probado; sin eco; ventanas cerradas si hay calle ruidosa.
- [ ] Limpiar historia de shell con `clear` antes de grabar.
