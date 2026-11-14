# Reglas para editores

Estas pautas aplican a GoLand, VS Code y editores similares; no requieren una
configuración específica del editor.

## Formato

- Usa Go 1.22.5 según `go.mod`.
- Formatea Go con `gofumpt` y respeta las reglas de `.golangci.yml`.
- Mantén la indentación existente: tabs para Go y dos espacios en YAML.
- Escribe nombres y comentarios Go en inglés.

## Organización

- `cmd/manager/`: punto de entrada.
- `internal/app/`: bootstrap y composición de dependencias.
- `internal/domain/`: reglas de negocio y selección desacoplada.
- `internal/infrastructure/`: interacción con Kubernetes y otros sistemas.
- `controllers/`: coordinación del ciclo de reconciliación.
- `api/v1alpha1/`: tipos versionados, validación y deepcopy.
- `config/` y `samples/`: manifiestos de instalación y ejemplos.

## Prácticas

- Pasa `context.Context` primero en funciones que realizan llamadas externas.
- Mantén reconciliación idempotente, errores explícitos y limpieza mediante
  finalizers cuando corresponda.
- Inyecta dependencias mediante interfaces donde facilite pruebas.
- Actualiza pruebas y documentación junto con el comportamiento.
- Ejecuta `make generate` y, si el esquema cambia, `make generate-crds` para
  cambios en los tipos API.

La configuración de lint vive en `.golangci.yml`; el repositorio no impone una
longitud máxima de línea ni un porcentaje fijo de cobertura en el Makefile.
