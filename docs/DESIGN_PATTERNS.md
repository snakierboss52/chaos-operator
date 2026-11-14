# Patrones y decisiones de diseño

Este documento describe patrones que se ven en la implementación actual; no
es una lista de patrones nominales que deban imponerse a cada cambio.

## Reconciliación declarativa

Los cuatro controladores (`PodChaos`, `NetworkChaos`, `StressChaos` y
`HTTPChaos`) observan recursos Kubernetes y coordinan su ciclo de vida. Deben
soportar reconciliación repetida, actualización de estado y limpieza al borrar
el recurso. Consulta cada controlador para ver sus fases y comportamiento
concreto.

## Inyección de dependencias

Los controladores reciben cliente, logger, configuración y reporter; la lógica
de dominio usa interfaces para desacoplar selección y operaciones. El manager y
el registro de controladores se componen en `internal/app/`.

## Estrategias de selección

`api/v1alpha1/` define los modos y selectores de objetivos. La selección se
implementa fuera del punto de entrada para permitir validación y pruebas
aisladas. Cambia esa lógica solo manteniendo los límites documentados por los
tipos y controladores.

## Estado y observabilidad

Los tipos comparten campos de status y resultados; la infraestructura expone
métricas y reportes. Eventos y logs aportan contexto operativo.

## Evitar documentación especulativa

No atribuyas al proyecto patrones como Repository, Builder, Singleton,
`ChaosExecutorFactory` o Strategy de ejecución si no están presentes en el
código vigente. Añade a esta página decisiones verificables y enlaza los
archivos de implementación correspondientes.
