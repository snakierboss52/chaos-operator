# Contribuir

Gracias por contribuir a goland-operator. Esta guía describe el flujo local
para cambios de código, API y documentación.

## Preparar el entorno

Necesitas Go 1.22.5 (o una versión compatible con `go.mod`) y Make. Docker,
kind, Helm y `kubectl` solo hacen falta para trabajar con el laboratorio o el
cluster.

```bash
git clone <url-del-repositorio>
cd goland-operator
go mod download
```

## Flujo de trabajo

1. Crea una rama para el cambio.
2. Mantén la lógica de negocio en `internal/domain/`; deja que los controladores
   coordinen reconciliación, estado y dependencias.
3. Añade o adapta pruebas junto al paquete afectado.
4. Actualiza los documentos y ejemplos que describan el comportamiento cambiado.
5. Revisa el diff y ejecuta las comprobaciones adecuadas antes de abrir el PR.

```bash
gofmt -w archivo.go
go test ./...
make lint
git diff --check
```

No todos los cambios requieren todas las comprobaciones; indica en el PR qué
ejecutaste y cualquier limitación. Los objetivos `make test`, `make lint` y
`make build` no requieren un cluster. Los objetivos `make kind-*`, `make
deploy`, `make install-crds`, addons y `load-*` sí pueden interactuar con
Docker o Kubernetes.

## Cambios en la API

Al modificar `api/v1alpha1/`:

- Mantén compatibilidad hacia atrás salvo que el cambio solicite una migración.
- Actualiza validaciones/defaults, webhooks y pruebas relacionadas.
- Ejecuta `make generate` para regenerar deepcopy.
- Si cambia el esquema, ejecuta `make generate-crds` y revisa los YAML en
  `config/crd/`.
- Actualiza ejemplos en `samples/` y la referencia en `docs/CHAOS_TYPES.md`.
- Comprueba que el registro de controladores y RBAC sigan cubriendo el cambio.

## Convenciones

- Código, nombres y comentarios de Go en inglés; documentación para usuarios
  principalmente en español.
- Sigue el formato existente y las reglas de `.golangci.yml`.
- Propaga `context.Context` a llamadas Kubernetes y errores con contexto.
- Mantén reconciliación idempotente y considera finalizers y limpieza de efectos.
- Usa permisos RBAC mínimos; no añadas credenciales ni datos sensibles.
- En cambios de comportamiento, explica el impacto y la forma de observarlo.

## Pull requests

Incluye un resumen del problema y la solución, los recursos o paquetes
afectados, las pruebas ejecutadas y capturas/logs solo cuando aclaren un cambio
observable. Mantén cada PR enfocado y describe cualquier paso manual o impacto
en despliegues existentes.
