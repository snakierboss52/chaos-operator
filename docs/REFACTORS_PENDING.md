# Pendientes técnicos y de evolución

## Propósito

Este archivo registra decisiones, funcionalidades y validaciones pendientes del proyecto `goland-operator`.

No representa funcionalidad implementada ni trabajo aprobado. Su objetivo es evitar que las brechas identificadas queden únicamente en documentos externos.

## Reglas de mantenimiento

- Estado inicial de todos los puntos: **No iniciado**.
- No mover un punto a completado sin código, pruebas y evidencia asociada.
- Las decisiones de arquitectura deben resolverse antes de crear tareas de implementación.
- Este archivo no autoriza despliegues en producción.

## Resumen

| ID | Pendiente | Tipo | Estado |
|---|---|---|---|
| PEND-001 | Ejecución programada mediante cron | Funcionalidad | No iniciado |
| PEND-002 | Decisión sobre CRD genéricos | Arquitectura | No iniciado |
| PEND-003 | IOChaos y TimeChaos | Funcionalidad | No iniciado |
| PEND-004 | Abortos automáticos | Seguridad operacional | No iniciado |
| PEND-005 | Namespaces permitidos y protegidos | Seguridad operacional | No iniciado |
| PEND-006 | HTTPChaos independiente del servidor | Arquitectura / portabilidad | No iniciado |
| PEND-007 | Preparación y certificación para producción | Validación | No iniciado |

## PEND-001 — Ejecución programada mediante cron

**Estado actual:** existe `SchedulerSpec.Cron` en el modelo de los recursos, pero no hay lógica que interprete la expresión, programe ejecuciones ni controle concurrencia.

**Decisiones pendientes:**

- Determinar si el scheduling será responsabilidad del operador o de un recurso externo como `CronJob`.
- Definir zona horaria, política de concurrencia, reintentos, historial y comportamiento ante periodos perdidos.
- Determinar cómo una ejecución programada crea o reinicia un experimento sin duplicar efectos.

**Criterio de cierre:** la alternativa arquitectónica queda documentada; las expresiones inválidas se rechazan; existen pruebas unitarias, de integración y end-to-end de programación, concurrencia y recuperación.

## PEND-002 — Decisión sobre CRD genéricos

**Estado actual:** el documento original describe `ChaosExperiment`, `ChaosSchedule` y `ChaosResult`. El código vigente implementa `PodChaos`, `NetworkChaos`, `StressChaos` y `HTTPChaos`, con resultados en `status` y en un `ConfigMap` de reporte.

**Decisiones pendientes:**

- Conservar el modelo actual de un CRD por tipo de falla o migrar a un modelo genérico.
- Definir compatibilidad, versionado y migración de recursos si se cambia el modelo.
- Determinar si `ChaosSchedule` y `ChaosResult` aportan valor suficiente como recursos independientes.

**Criterio de cierre:** existe una decisión de arquitectura registrada con ventajas, costos, compatibilidad y plan de migración. No deben coexistir dos modelos públicos sin una estrategia explícita.

## PEND-003 — IOChaos y TimeChaos

**Estado actual:** aparecen en documentación secundaria, pero no existen CRD, reconcilers ni operaciones específicas para `IOChaos` o `TimeChaos`.

**Decisiones pendientes:**

- Confirmar si ambos tipos pertenecen al alcance académico y técnico de una versión futura.
- Definir mecanismos de inyección compatibles con Kubernetes y los runtimes objetivo.
- Identificar privilegios, dependencias dentro del contenedor, límites de impacto y recuperación.

**Criterio de cierre:** cada tipo aceptado cuenta con diseño, CRD, validación, reconciler, recuperación, observabilidad, documentación y pruebas. Si se descarta, debe retirarse de la documentación de alcance.

## PEND-004 — Abortos automáticos

**Estado actual:** el documento original propone abortar por métricas, disponibilidad o tasa de error, pero esa capacidad no está implementada.

**Decisiones pendientes:**

- Definir fuentes de señal y comportamiento cuando Prometheus u otra dependencia no estén disponibles.
- Definir umbrales, ventana de evaluación, frecuencia, tolerancia y prevención de falsos positivos.
- Diferenciar aborto, fallo y finalización; establecer el proceso de recuperación para cada tipo de caos.
- Determinar cómo se registran la causa, las métricas evaluadas y el resultado del aborto.

**Criterio de cierre:** los criterios son configurables y validados; un umbral excedido detiene nuevas inyecciones, ejecuta la recuperación aplicable y deja evidencia auditable en `status`, Events, métricas y reporte.

## PEND-005 — Namespaces permitidos y protegidos

**Estado actual:** existe control cuantitativo de `blastRadius`, pero no una política configurable de namespaces permitidos/protegidos ni exclusión efectiva mediante una etiqueta de protección.

**Decisiones pendientes:**

- Definir una política de denegación por defecto o una lista explícita de namespaces permitidos.
- Establecer namespaces que nunca pueden ser objetivo, incluido el namespace del operador.
- Definir la clave y semántica de la etiqueta de protección.
- Determinar si la validación ocurre en admission, reconciliación o en ambas capas.

**Criterio de cierre:** ningún selector, lista explícita o combinación de campos puede eludir la política; los intentos rechazados quedan registrados; existen pruebas negativas para namespaces y pods protegidos.

## PEND-006 — HTTPChaos independiente del servidor

**Estado actual:** `HTTPChaos` modifica la configuración de Nginx dentro del contenedor objetivo. Esto acopla la funcionalidad a una tecnología y estructura de archivos concretas.

**Decisiones pendientes:**

- Elegir el mecanismo de interceptación: proxy sidecar, service mesh, eBPF u otra alternativa evaluada.
- Definir compatibilidad con distintos servidores y protocolos HTTP.
- Precisar la semántica real de `abort`, `delay`, `replace` y `patch`, incluidos porcentaje, método, ruta y dirección request/response.
- Definir aislamiento, restauración y validación estricta de todos los valores externos antes de generar comandos o configuración.

**Criterio de cierre:** la solución seleccionada no depende de Nginx, aplica completamente la especificación declarada, recupera el estado previo y cuenta con pruebas funcionales, de compatibilidad y seguridad.

## PEND-007 — Preparación y certificación para producción

**Estado actual:** la evidencia disponible corresponde principalmente a desarrollo local y pruebas automatizadas parciales. No existe certificación para producción, multi-cluster, pruebas de penetración ni alta disponibilidad real.

**Líneas de trabajo pendientes:**

- Validación end-to-end en un entorno de staging representativo.
- Definición y prueba de alta disponibilidad con múltiples réplicas y leader election.
- Pruebas multi-cluster y aislamiento de credenciales, contextos y resultados.
- Revisión de RBAC y mínimo privilegio.
- Pruebas de penetración y revisión específica de los datos de CR usados en `pods/exec`, shell y configuraciones dinámicas.
- Pruebas de carga, estrés prolongado, recuperación ante reinicios y degradación de dependencias.
- Runbooks de operación, rollback, incidentes y suspensión inmediata de experimentos.

**Criterio de cierre:** existen criterios de entrada y salida aprobados, evidencias reproducibles de todas las pruebas, cero hallazgos críticos o altos abiertos y aprobación explícita de las personas responsables del entorno productivo.

## Decisiones que deben resolverse primero

1. Modelo definitivo de CRD: recursos especializados o modelo genérico.
2. Responsabilidad del scheduling: operador o mecanismo externo.
3. Política de namespaces y recursos protegidos.
4. Mecanismo futuro de inyección HTTP.
5. Alcance de IOChaos, TimeChaos y despliegue multi-cluster.

## Historial

| Fecha | Cambio |
|---|---|
| 2026-08-20 | Creación del registro inicial de pendientes. No se realizaron cambios de código. |
