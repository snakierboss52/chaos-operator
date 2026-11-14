# Arquitectura

El repositorio implementa un operador Kubernetes con `controller-runtime`.
Los tipos de API son `chaos.engineering.io/v1alpha1`: `PodChaos`,
`NetworkChaos`, `StressChaos` y `HTTPChaos`.

## Componentes

```text
cmd/manager
    └── internal/app: scheme, manager, registro de controladores y servidores
          ├── controllers/: reconciliadores por tipo de recurso
          ├── internal/domain/: selección y reglas de dominio
          ├── internal/infrastructure/: operaciones Kubernetes, métricas y reporte
          └── internal/api/: API HTTP local para crear/consultar recursos
api/v1alpha1/: tipos, validación, webhooks y deepcopy generado
config/: CRDs, RBAC, deployment y servicios
```

`cmd/manager` es el punto de entrada. `internal/app` construye el manager,
inyecta cliente, logger, scheme y reporter, y registra los cuatro
controladores. Los reconciliadores coordinan el ciclo de vida con Kubernetes;
la selección de targets y las reglas reutilizables viven fuera de ellos.

## Reconciliación

Cada controlador observa su recurso, valida precondiciones, selecciona pods,
aplica límites configurados, ejecuta el efecto, actualiza `status` y registra
métricas/eventos. El estado incluye fase, condiciones, tiempos y objetivos
afectados según el tipo. Los finalizers permiten retirar efectos durante la
eliminación del recurso.

La reconciliación puede repetirse: los cambios deben ser idempotentes, los
errores deben devolver reintento y la limpieza debe tolerar recursos que ya no
existen. Los detalles concretos se definen en cada controlador.

## API HTTP

`internal/api/` sirve rutas locales para crear recursos desde manifiestos o
requests simplificados y leerlos por namespace/nombre. Esta API escribe CRDs;
la ejecución sigue a cargo de los reconciliadores. Su namespace permitido se
configura con `API_ALLOWED_NAMESPACES` (por defecto `chaos-demo`). No incluye
autenticación, por lo que su exposición debe limitarse al laboratorio.

## Servidores y configuración

| Variable | Default | Uso |
| --- | --- | --- |
| `METRICS_ADDR` | `:8080` | Endpoint de métricas del manager |
| `HEALTH_PROBE_ADDR` | `:8081` | `/healthz` y `/readyz` |
| `API_ADDR` | `:8082` | API HTTP de experimentos |
| `API_ALLOWED_NAMESPACES` | `chaos-demo` | Namespaces permitidos por la API HTTP |
| `ENABLE_LEADER_ELECTION` | `false` en código | Leader election; deployment lo habilita |
| `ENABLE_WEBHOOKS` | `false` | Registro de defaults y validaciones webhook |

Los webhooks se registran solo si `ENABLE_WEBHOOKS=true`; la presencia de
validaciones en el código no implica que estén habilitados en cada despliegue.

## Límites

La API HTTP está diseñada para laboratorio y no tiene autenticación. `HTTPChaos`
está acoplado a la configuración de Nginx que el controlador espera en el
contenedor objetivo. Comprueba [Tipos de Chaos](CHAOS_TYPES.md),
[API HTTP](PODCHAOS_HTTP_API.md) y los manifiestos actuales antes de asumir
capacidades o garantías adicionales.
