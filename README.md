# goland-operator

[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.20%2B-326CE5?logo=kubernetes)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Operador de Kubernetes para ejecutar experimentos de resiliencia mediante los
recursos `PodChaos`, `NetworkChaos`, `StressChaos` y `HTTPChaos` del grupo
`chaos.engineering.io/v1alpha1`. El repositorio incluye controladores, CRDs,
ejemplos y un laboratorio local basado en kind.

## Estado y alcance

- `PodChaos` elimina pods o contenedores seleccionados.
- `NetworkChaos` aplica fallos de red compatibles con el entorno del pod.
- `StressChaos` genera carga de CPU y memoria en los pods seleccionados.
- `HTTPChaos` aplica fallos HTTP mediante configuración en el contenedor Nginx
  objetivo; requiere el layout que espera el controlador y no es un proxy HTTP
  independiente.
- Cada recurso mantiene su ciclo de ejecución y resultado en `status`; los
  controladores usan finalizers para limpiar efectos al eliminar recursos.
- La API HTTP local acepta manifiestos o bodies simplificados. Está pensada
  para el laboratorio y no incluye autenticación.

Consulta [Tipos de Chaos](docs/CHAOS_TYPES.md) para campos, acciones y límites
por tipo. Revisa los manifiestos de [samples](samples/README.md) antes de
ejecutar experimentos en un cluster.

## Inicio rápido con kind

Requisitos: Go 1.22+, Docker, `kubectl`, kind, Helm y Make. El cluster debe
tener disponible la configuración local del repositorio.

```bash
make kind-up
make docker-build
make kind-load-image
make install-crds
make deploy
make addons-install       # Prometheus, Grafana, Traefik y kube-state-metrics
make lab-deploy            # workloads de prueba en chaos-demo
```

Para crear un experimento:

```bash
kubectl apply -f samples/podchaos/basic-kill-one.yaml
kubectl get podchaos -A -w
kubectl describe podchaos basic-kill-one -n chaos-demo
```

El laboratorio local publica Prometheus y Grafana en `http://prometheus.localhost:30080`
y `http://grafana.localhost:30080`. La API HTTP local está en
`http://operator.localhost:30080`; ver [API HTTP](docs/PODCHAOS_HTTP_API.md) y
[monitoreo](docs/MONITORING.md).

Para retirar los workloads y el stack local usa `make lab-clean`,
`make addons-uninstall` y, si quieres borrar el cluster, `make kind-down`.
`make undeploy` retira el operador. `make uninstall-crds` elimina los CRDs y
los recursos personalizados asociados: úsalo solo si quieres borrar también
los experimentos almacenados.

## Desarrollo

```bash
go test ./...       # pruebas
make lint           # golangci-lint
make build          # bin/goland-operator
make run            # ejecutar manager localmente
make generate       # deepcopy generado
make generate-crds  # CRDs desde los tipos API
```

Para cambiar tipos en `api/v1alpha1/`, regenera deepcopy y CRDs y revisa los
manifiestos resultantes, webhooks, muestras y documentación. Los targets de
`make kind-*`, `make deploy`, `make install-crds`, addons y load test interactúan
con Docker o un cluster. Consulta [CONTRIBUTING.md](CONTRIBUTING.md) y la
[referencia de comandos](docs/COMMANDS.md).

## Estructura

| Ruta | Contenido |
| --- | --- |
| `api/v1alpha1/` | Tipos de recursos, validaciones, webhooks y deepcopy |
| `controllers/` | Reconciliación y ciclo de vida de cada experimento |
| `internal/app/` | Configuración del manager y registro de dependencias |
| `internal/domain/` | Selección de objetivos y lógica de dominio |
| `internal/infrastructure/` | Operaciones Kubernetes, métricas y reportes |
| `internal/api/` | API HTTP local para crear y consultar experimentos |
| `config/` | CRDs, RBAC, servicios y deployment |
| `samples/` | Ejemplos de recursos y workloads de laboratorio |
| `docs/`, `monitoring/`, `loadtest/` | Guías, dashboards y pruebas de carga |

## Documentación

- [Inicio y despliegue](docs/GETTING_STARTED.md)
- [Arquitectura](docs/ARCHITECTURE.md)
- [Tipos y campos](docs/CHAOS_TYPES.md)
- [API HTTP local](docs/PODCHAOS_HTTP_API.md)
- [Monitoreo](docs/MONITORING.md)
- [Comandos](docs/COMMANDS.md)
- [Muestras](samples/README.md)
- [Contribución](CONTRIBUTING.md)

## Licencia

Apache License 2.0. Consulta [LICENSE](LICENSE).
