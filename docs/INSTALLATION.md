# Instalación

El flujo documentado aquí instala el operador en el cluster activo de
`kubectl`. Comprueba `kubectl config current-context` antes de continuar.

## Requisitos

- Kubernetes y `kubectl` configurado.
- Docker si vas a construir la imagen localmente.
- Go 1.22.5 para desarrollo; Make para los targets del repositorio.
- Para laboratorio local: kind y Helm.

## Laboratorio kind

```bash
make kind-up
make docker-build
make kind-load-image
make install-crds
make deploy
make addons-install
make lab-deploy
```

El comando de carga toma la imagen `goland-operator:latest` por defecto. Si
usas otro tag, pásalo igual a ambos comandos:

```bash
make docker-build IMG=goland-operator:dev
make kind-load-image IMG=goland-operator:dev
make deploy IMG=goland-operator:dev
```

`make deploy` instala namespace, RBAC, deployment y servicios en
`chaos-system`. `make addons-install` prepara el stack de observabilidad local;
`make lab-deploy` añade workloads de prueba en `chaos-demo`.

## Cluster remoto

Construye y publica una imagen en un registry accesible por los nodos y aplica
los CRDs/despliegue usando el contexto correcto:

```bash
make docker-build IMG=registry.example/goland-operator:tag
make docker-push IMG=registry.example/goland-operator:tag
make install-crds
make deploy IMG=registry.example/goland-operator:tag
```

Revisa y adapta los manifiestos de `config/manager/` y `config/rbac/` a la
política de tu cluster antes de un despliegue remoto. No existe un chart Helm
del operador mantenido en este repositorio; Helm se usa para los addons del
laboratorio local.

## Verificar

```bash
kubectl get crd | grep chaos.engineering.io
kubectl get pods -n chaos-system
kubectl get services -n chaos-system
kubectl logs -n chaos-system deployment/chaos-operator
```

Los endpoints y variables de configuración están descritos en
[Arquitectura](ARCHITECTURE.md). Consulta
[Primeros pasos](GETTING_STARTED.md) para ejecutar un experimento.

## Retirar

```bash
make lab-clean
make addons-uninstall
make undeploy
make uninstall-crds
```

`make uninstall-crds` también elimina recursos personalizados almacenados.
Para el laboratorio kind, `make kind-down` borra el cluster completo.
