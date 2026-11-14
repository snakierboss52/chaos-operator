# Reglas de ingeniería

Estas convenciones complementan [CONTRIBUTING](../CONTRIBUTING.md) y las
instrucciones de [`AGENTS.md`](../AGENTS.md).

## Arquitectura y controladores

- Mantén lógica de negocio fuera de `cmd/manager/`; el punto de entrada delega
  el arranque en `internal/app/`.
- Mantén lógica de dominio en `internal/domain/` y llamadas concretas a
  Kubernetes/sistemas externos en infraestructura.
- Limita los controladores a coordinar reconciliación, dependencias y estado.
- Diseña `Reconcile` como operación repetible; maneja retries, errores,
  finalizers y recuperación de efectos.

## API y Kubernetes

- Conserva compatibilidad hacia atrás salvo que se acuerde una migración.
- Valida entradas y mantén defaults y webhooks alineados con los tipos.
- Regenera deepcopy y CRDs después de cambios API y revisa los archivos
  generados.
- Añade solo los permisos RBAC que requieren las llamadas nuevas.
- Pasa `context.Context` a operaciones Kubernetes y propaga errores con
  contexto útil.

## Calidad y observabilidad

- Sigue `.golangci.yml`, `gofumpt` y el estilo del código existente.
- Mantén pruebas junto a los paquetes afectados; ejecuta las comprobaciones
  pertinentes y reporta qué se ejecutó.
- Usa logging estructurado y eventos para acciones visibles del operador.
- No registres secretos ni datos sensibles.
- No hay un umbral de cobertura de pruebas configurado como requisito en el
  Makefile; evita tratar un porcentaje aspiracional como gate del proyecto.

## Documentación

Actualiza documentación y muestras cuando cambien la API, el comportamiento,
los comandos o los requisitos de ejecución. Escribe comentarios de Go en inglés
y documentación para usuarios principalmente en español.
