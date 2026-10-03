# API HTTP local para PodChaos

La API local acepta un manifiesto `PodChaos` en YAML o JSON, valida sus campos
y crea el recurso Kubernetes. El controller existente reconcilia ese recurso y
ejecuta el experimento. El endpoint no ejecuta el caos directamente.

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

La ruta local queda en `http://operator.localhost:30080/api/v1/podchaos` y solo
acepta recursos del namespace `chaos-demo` por defecto. La exposición usa el
puerto local de kind ligado a loopback.

## Crear un experimento desde un manifiesto

El ejemplo existente elimina un Pod nginx y configura olas cada dos minutos
durante veinte minutos. Envíalo con:

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

El estado lo actualiza el controller. Para un smoke test, elimina el recurso
después de confirmar que el controller afectó un Pod; así no quedan activas las
olas repetidas del ejemplo:

```bash
kubectl delete podchaos basic-kill-one -n chaos-demo
```

La API aplica los defaults y validación de `PodChaos`, exige un nombre y un
namespace permitido, restringe el selector al mismo namespace permitido (por
defecto `chaos-demo`), limita el body a 1 MiB y rechaza nombres duplicados con
`409 Conflict`. Si el selector omite namespaces, se usa el namespace del
recurso. En esta primera etapa es una API local de laboratorio; no se
debe publicar hacia una red compartida o AWS sin añadir autenticación y
autorización.
