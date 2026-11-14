# Registro de despliegue

Esta página correspondía a una ejecución puntual y sus URLs ya no describen
necesariamente el estado local actual. Usa las guías mantenidas:

- [Instalación](INSTALLATION.md)
- [Monitoreo](MONITORING.md)
- [Comandos](COMMANDS.md)

Para observar un despliegue actual, consulta el estado del cluster con
`kubectl get pods -n chaos-system` y las métricas en el puerto 8080 del servicio
`chaos-operator-metrics`.
