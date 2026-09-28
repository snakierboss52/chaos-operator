# Addons locales con Helm

`install-local.sh` instala releases Helm independientes para kube-state-metrics,
Prometheus y Grafana en `monitoring`. Los charts y sus versiones están fijados
en el script; los valores específicos del entorno están junto a cada addon.

```bash
make addons-install
make addons-uninstall
```

La instalación requiere que el contexto activo sea `kind-chaos-testing-v2`.
Para otro nombre, definir `KIND_CLUSTER=<nombre>` tanto al crear el cluster como
al instalar addons. Para acceder, inicia port-forwards a localhost:9090
(Prometheus) y localhost:3000 (Grafana). Ambos Services son ClusterIP.

Grafana obtiene los JSON de `monitoring/*.json` mediante su sidecar. Prometheus
conserva los scrape jobs del operador, kube-state-metrics, pods anotados, kubelet
y cAdvisor. kube-state-metrics es una dependencia explícita porque los dashboards
consultan métricas `kube_*`.

InfluxDB se conserva para enviar resultados k6 en vivo. Todavía se instala desde
el manifiesto `monitoring/influxdb-deployment.yaml`; no forma parte de los tres
charts Helm upstream.

## Perfil futuro para AWS

La configuración actual es exclusiva para kind. Antes de usar addons en AWS se
deben añadir valores separados para ese entorno y definir persistencia, acceso
autenticado, exposición privada, recursos, retención y almacenamiento de
secretos. No se deben reutilizar los valores ni las credenciales locales.
