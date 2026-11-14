# Referencia de comandos

Los targets disponibles están definidos en el `Makefile` y `loadtest/load.mk`.
Los siguientes ejemplos reflejan el flujo mantenido en el repositorio.

## Desarrollo local

```bash
make build          # genera bin/goland-operator
make run            # ejecuta ./cmd/manager
make test           # pruebas de ./... con coverage/cover.out
make lint           # golangci-lint run ./...
make tidy           # go mod tidy
make generate       # deepcopy de api/...
make generate-crds  # manifiestos en config/crd/
```

También puedes ejecutar pruebas de un paquete con `go test ./internal/domain/...`
o `go test ./controllers/...`.

## Imagen y despliegue

```bash
make docker-build IMG=goland-operator:latest
make docker-push IMG=registry.example/goland-operator:tag
make install-crds
make deploy IMG=goland-operator:latest
make undeploy
make uninstall-crds
```

`install-crds` regenera y aplica CRDs al cluster del contexto activo.
`deploy` aplica namespace, RBAC, deployment y servicios y espera a que el
deployment esté disponible. `undeploy` elimina el operador, servicios y RBAC;
`uninstall-crds` elimina las definiciones y los recursos personalizados que
dependen de ellas.

## Laboratorio kind

```bash
make kind-up
make kind-status
make kind-load-image IMG=goland-operator:latest
make addons-install
make lab-deploy
make lab-clean
make addons-uninstall
make kind-down
```

`kind-load-image` espera que la imagen ya exista localmente. Los addons instalan
Traefik, Prometheus, Grafana y kube-state-metrics. `lab-deploy` prepara las
cargas objetivo en `chaos-demo`. Consulta [scripts/kind](../scripts/kind/README.md)
y [monitoreo](MONITORING.md).

## Pruebas de carga

Los targets de carga se documentan en [loadtest/runbook.md](../loadtest/runbook.md).
Usa `make load-help` para listar opciones; `load-up`, `load-deploy`, `load-*` y
`load-down` crean o modifican recursos de Docker/kind.

## Observar experimentos

```bash
kubectl get podchaos,networkchaos,stresschaos,httpchaos -A
kubectl describe podchaos <nombre> -n <namespace>
kubectl logs -n chaos-system deployment/chaos-operator -f
kubectl port-forward -n chaos-system svc/chaos-operator-metrics 8080:8080
curl http://localhost:8080/metrics
```

El servicio HTTP del operador usa el puerto `8082` por defecto y el de probes
`8081`; las rutas locales y su exposición se describen en
[API HTTP](PODCHAOS_HTTP_API.md). La configuración Prometheus/Grafana del
laboratorio se consulta en [Monitoreo](MONITORING.md).
