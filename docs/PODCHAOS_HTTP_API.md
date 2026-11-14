# API HTTP local para experimentos Chaos

La API local acepta manifiestos `PodChaos`, `NetworkChaos`, `StressChaos` y
`HTTPChaos` en YAML o JSON, valida sus campos y crea el recurso Kubernetes. El
controller correspondiente reconcilia el recurso y ejecuta el experimento. El
endpoint no ejecuta el caos directamente.

## Requisitos

Despliega kind, el operator y Traefik en este orden:

```bash
make kind-up
make docker-build
make kind-load-image
make install-crds
make deploy
make addons-install
make lab-deploy
```

Las rutas locales son `/api/v1/podchaos`, `/api/v1/networkchaos`,
`/api/v1/stresschaos` y `/api/v1/httpchaos` bajo
`http://operator.localhost:30080`. Solo aceptan recursos del namespace
`chaos-demo` por defecto. La exposición usa el puerto local de kind ligado a
loopback. También está disponible `POST /api/v1/experiments`, que construye el
recurso adecuado a partir de un body simplificado.

## Endpoint unificado

Envía `type`, `namespace`, `mode`, `duration` y la acción propia del tipo. Se
puede enviar `appSelector: nginx` como atajo para seleccionar pods con la label
`app: nginx`; para selectores más complejos, usa `selector` en su lugar. Si se
omite `name`, la API genera uno. Por ejemplo:

```bash
curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @- http://operator.localhost:30080/api/v1/experiments <<'YAML'
type: PodChaos
namespace: chaos-demo
mode: one
appSelector: nginx
duration: 30s
action: pod-kill
YAML
```

Los otros tipos usan el mismo endpoint y agregan sus campos específicos:

```bash
curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @- http://operator.localhost:30080/api/v1/experiments <<'YAML'
type: NetworkChaos
namespace: chaos-demo
mode: all
appSelector: apache-target
duration: 10m
action: delay
delay:
  latency: 100ms
  jitter: 10ms
YAML

curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @- http://operator.localhost:30080/api/v1/experiments <<'YAML'
type: StressChaos
namespace: chaos-demo
mode: one
appSelector: nginx
duration: 15m
stressors:
  cpu:
    workers: 4
    load: 100
YAML

curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @- http://operator.localhost:30080/api/v1/experiments <<'YAML'
type: HTTPChaos
namespace: chaos-demo
mode: fixed-percent
value: "33"
appSelector: nginx
duration: 5m
target: Request
port: 80
method: GET
path: "/*"
abort:
  statusCode: 503
  percentage: 25
YAML
```

Cada respuesta `201 Created` incluye el nombre generado y un header `Location`
que apunta al recurso Kubernetes tipado. También se pueden seguir usando las
rutas específicas documentadas abajo.

## Crear un experimento desde un manifiesto

Cada ruta acepta el manifiesto correspondiente. Por ejemplo, para crear un
`PodChaos`:

```bash
curl --fail-with-body -i \
  -H 'Content-Type: application/yaml' \
  --data-binary @samples/podchaos/basic-kill-one.yaml \
  http://operator.localhost:30080/api/v1/podchaos
```

La respuesta exitosa es `201 Created` e incluye el recurso. El header `Location`
apunta a la ruta para consultar su estado:

```bash
curl --fail-with-body \
http://operator.localhost:30080/api/v1/podchaos/chaos-demo/basic-kill-one
```

Los otros tipos usan el mismo flujo, cambiando manifiesto y ruta:

```bash
curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @samples/networkchaos/delay.yaml \
  http://operator.localhost:30080/api/v1/networkchaos

curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @samples/stresschaos/cpu-stress.yaml \
  http://operator.localhost:30080/api/v1/stresschaos

curl --fail-with-body -i -H 'Content-Type: application/yaml' \
  --data-binary @samples/httpchaos/abort-503.yaml \
  http://operator.localhost:30080/api/v1/httpchaos
```

El estado lo actualiza el controller. Para un smoke test, elimina el recurso
después de confirmar que el controller afectó un Pod; así no quedan activas las
olas repetidas del ejemplo:

```bash
kubectl delete podchaos basic-kill-one -n chaos-demo
```

La API aplica los defaults y validaciones del webhook para cada tipo, exige un nombre y un
namespace permitido, restringe el selector al mismo namespace permitido (por
defecto `chaos-demo`), limita el body a 1 MiB y rechaza nombres duplicados con
`409 Conflict`. Si el selector omite namespaces, se usa el namespace del
recurso. En `NetworkChaos`, también valida el namespace de un selector destino
de partición si se especifica. En esta primera etapa es una API local de laboratorio; no se
debe publicar hacia una red compartida o AWS sin añadir autenticación y
autorización.
