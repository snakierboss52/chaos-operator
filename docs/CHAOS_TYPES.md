# Tipos de experimentos

La API vigente es `chaos.engineering.io/v1alpha1`. Los ejemplos ejecutables
están en [`samples/`](../samples/README.md); usa esos manifiestos como
referencia concreta y valida los campos contra los CRDs de `config/crd/`.

## Campos compartidos

Todos los recursos seleccionan pods con `spec.selector` y aceptan estos modos:

| `mode` | `value` |
| --- | --- |
| `one` | no requerido |
| `all` | no requerido |
| `fixed` | cantidad de pods |
| `fixed-percent` | porcentaje |
| `random-max-percent` | porcentaje máximo |

El selector puede incluir `namespaces`, `labelSelectors`, `fieldSelectors`,
`podPhaseSelectors`, `nodeSelectors` y una lista explícita `pods`. `duration`,
`scheduler.cron` y `blastRadius.maxPods`/`maxPercentage` están disponibles
según el recurso. La semántica exacta y las restricciones se implementan en
`api/v1alpha1/` y `internal/domain/`.

El estado compartido expone `phase` (`Pending`, `Running`, `Completed`,
`Failed`, `Stopped`), condiciones, pods afectados, tiempos, mensaje,
`observedGeneration` y resultados por objetivo. No infieras garantía de
protección de PDB o de namespaces concretos: consulta el selector y la lógica
actual del controlador.

## PodChaos

Acciones admitidas: `pod-kill`, `pod-failure` y `container-kill`. Puede
configurar `gracePeriod`, `force`, `containerNames`, `duration`, `scheduler` y
`blastRadius`.

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: basic-kill-one
  namespace: chaos-demo
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  action: pod-kill
  duration: 30s
```

## NetworkChaos

Acciones: `delay`, `loss`, `duplicate`, `corrupt`, `partition` y `bandwidth`.
Los parámetros correspondientes son `delay`, `loss`, `duplicate`, `corrupt` y
`bandwidth`. `direction` puede ser `from`, `to` o `both`; `target` y
`externalTargets` permiten especificar destinos según la acción.

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: NetworkChaos
metadata:
  name: delay
  namespace: chaos-demo
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  action: delay
  direction: to
  delay:
    latency: 100ms
    jitter: 10ms
  duration: 30s
```

## StressChaos

`spec.stressors` ofrece `cpu`, `memory`, `disk` y `io`; pueden combinarse.
La inyección depende de las herramientas/permisos disponibles en el contenedor
objetivo. `containerNames` permite elegir contenedores; de lo contrario, el
código selecciona uno según el comportamiento implementado por el controlador.

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: StressChaos
metadata:
  name: cpu-stress
  namespace: chaos-demo
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  stressors:
    cpu:
      workers: 2
      load: 80
  duration: 30s
```

## HTTPChaos

Permite definir `target` (`Request`, `Response` o `Both`), `port`, y filtros
opcionales `path` y `method`, junto con una acción `abort`, `delay`, `replace`
o `patch`. El controlador actual modifica configuración de Nginx dentro del
pod; no es una inyección genérica para cualquier servidor HTTP. Usa los
workloads preparados en `samples/workloads/` y revisa sus limitaciones antes
de aplicar el recurso.

```yaml
apiVersion: chaos.engineering.io/v1alpha1
kind: HTTPChaos
metadata:
  name: abort-503
  namespace: chaos-demo
spec:
  mode: one
  selector:
    labelSelectors:
      app: nginx
  target: Request
  port: 80
  path: "/*"
  method: GET
  abort:
    statusCode: 503
    percentage: 25
  duration: 30s
```

## Seguridad y validación

Empieza en un cluster aislado, con un selector acotado y duración breve.
`blastRadius` limita la selección donde se aplica; no reemplaza la revisión de
permisos, efectos secundarios y estrategia de recuperación de cada carga.
Webhooks de default/validación se registran cuando `ENABLE_WEBHOOKS=true`;
consulta la configuración del despliegue antes de asumir que están activos.
