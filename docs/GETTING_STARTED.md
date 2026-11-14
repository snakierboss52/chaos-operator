# Primeros pasos

Esta guía usa el laboratorio kind mantenido en este repositorio. Para requisitos
y detalles de comandos, consulta [Installation](INSTALLATION.md) y
[COMMANDS](COMMANDS.md).

## Requisitos

Go 1.22.5, Docker, `kubectl`, kind, Helm y Make. El acceso a internet puede ser
necesario para descargar imágenes y charts.

## Levantar el laboratorio

Desde la raíz del repositorio:

```bash
make kind-up
make docker-build
make kind-load-image
make install-crds
make deploy
make addons-install
make lab-deploy
```

Esto instala el operador y sus CRDs, observabilidad local y workloads de
prueba en `chaos-demo`. Para detalles del stack consulta
[Monitoreo](MONITORING.md).

## Ejecutar un experimento

Los ejemplos canónicos están en `samples/`. Por ejemplo:

```bash
kubectl apply -f samples/podchaos/basic-kill-one.yaml
kubectl get podchaos -n chaos-demo -w
kubectl describe podchaos basic-kill-one -n chaos-demo
kubectl get events -n chaos-demo --sort-by=.lastTimestamp
```

El reconciliador actualiza `status` durante la ejecución. Al borrar el recurso,
el finalizer intenta limpiar los efectos asociados. Consulta también
[Tipos de Chaos](CHAOS_TYPES.md) y [muestras](../samples/README.md).

La API HTTP del laboratorio crea el mismo tipo de recurso Kubernetes y el
controller es quien lo ejecuta. Sus rutas, namespace permitido y limitaciones
están en [API HTTP local](PODCHAOS_HTTP_API.md).

## Limpieza

```bash
make lab-clean
make addons-uninstall
make undeploy
make kind-down
```

`make uninstall-crds` elimina también los CRDs; al hacerlo, Kubernetes elimina
los recursos de esos tipos almacenados en el cluster.

## Solución de problemas

- Comprueba que el contexto de `kubectl` apunta al cluster esperado con
  `kubectl config current-context`.
- Revisa `kubectl get pods -A` y los logs con
  `kubectl logs -n chaos-system deployment/chaos-operator`.
- Si el recurso no selecciona objetivos, verifica labels, namespace y campos
  `spec.selector` con `kubectl describe`.
- Verifica el estado y eventos del recurso antes de volver a aplicar un
  experimento.
