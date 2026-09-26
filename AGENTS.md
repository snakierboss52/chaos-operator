# Guía para agentes

## Proyecto

Este repositorio contiene un operador de Kubernetes escrito en Go (`go 1.22.5`) con `controller-runtime` 0.17 y Kubernetes client libraries 0.29. Su API es `chaos.engineering.io/v1alpha1` y define `PodChaos`, `NetworkChaos`, `StressChaos` y `HTTPChaos`.

## Estructura

- `cmd/manager/`: punto de entrada; el arranque y el registro de controladores están en `internal/app/`.
- `api/v1alpha1/`: tipos de recursos, esquemas, webhooks y deepcopy generado.
- `controllers/`: reconciliadores; coordinan las operaciones y actualizan el estado de los recursos.
- `internal/domain/`: selección de targets y lógica de ejecución de experimentos, desacoplada mediante interfaces.
- `internal/infrastructure/`: operaciones de Kubernetes, métricas y reporte de resultados.
- `config/`: CRDs y manifiestos de instalación, RBAC, webhooks y overlays de Kustomize.
- `samples/`: ejemplos de recursos para cada tipo de chaos.
- `docs/`, `monitoring/` y `loadtest/`: documentación, manifiestos de monitoreo y recursos de pruebas de carga.

Mantén la lógica de negocio fuera del punto de entrada y limita los controladores a coordinar el ciclo de reconciliación y sus dependencias.

## Comandos habituales

```bash
make build             # Compila bin/goland-operator
make run               # Ejecuta el manager localmente
make test              # Ejecuta go test ./... con cobertura en coverage/cover.out
make lint              # Ejecuta golangci-lint según .golangci.yml
make tidy              # Ejecuta go mod tidy
make generate          # Regenera deepcopy para api/...
make generate-crds     # Regenera config/crd/ desde los tipos de API
```

Las operaciones `make install-crds`, `make deploy`, `make undeploy` y los objetivos `load-*` interactúan con un clúster o con Docker/kind. Úsalos solo cuando la tarea requiera explícitamente instalación o pruebas de integración/carga.

## Cambios en la API y manifiestos

- Al cambiar tipos en `api/v1alpha1/`, revisa validaciones, defaults, webhooks y pruebas correspondientes.
- Regenera `zz_generated.deepcopy.go` con `make generate` cuando cambien tipos que requieren deepcopy.
- Si el esquema de un recurso cambia, ejecuta `make generate-crds` y revisa los YAML generados en `config/crd/`.
- Mantén los ejemplos pertinentes en `samples/` y la documentación en `docs/` alineados con el contrato de la API.
- Trata los cambios de API como compatibles hacia atrás salvo que la tarea indique una migración. Revisa efectos en CRDs, webhooks y estado antes de cambiar campos existentes.

## Convenciones de implementación

- Sigue el formato de Go del repositorio; `.golangci.yml` habilita `gofumpt`, `govet`, `staticcheck`, `revive`, `errcheck`, `gosec` y otros linters.
- Pasa `context.Context` a operaciones que interactúan con Kubernetes y propaga errores con contexto cuando sea útil.
- Diseña `Reconcile` para manejar reintentos y ejecuciones repetidas de forma segura; considera limpieza y finalizers al modificar el ciclo de vida de experimentos.
- Conserva la inyección de dependencias por interfaces en la lógica de dominio para facilitar pruebas.
- Valida entradas de recursos y conserva permisos RBAC mínimos al añadir llamadas o recursos Kubernetes.
- Usa logs estructurados y eventos Kubernetes para informar acciones relevantes del operador.
- Mantén los nombres y comentarios de Go en inglés, en línea con el código existente; la documentación para usuarios del proyecto está principalmente en español.

## Pruebas

Las pruebas Go están junto a los paquetes (`*_test.go`). Para una comprobación completa, `make test` ejecuta todos los paquetes; para cambios acotados, usa `go test ./ruta/del/paquete` o `go test -run NombreDelTest ./ruta/del/paquete`. `make lint` aplica la configuración del repositorio. Ejecuta las comprobaciones adecuadas al cambio y comunica si alguna no se pudo ejecutar.

## Documentación de referencia

Consulta `CLAUDE.md`, `docs/RULES.md`, `docs/EDITOR_RULES.md` y `README.md` para contexto adicional. Si alguna descripción general contradice el código o los manifiestos actuales, verifica el comportamiento en la implementación antes de modificarlo.
