# Runbook — Pruebas de carga del Chaos Engineering Operator

Guía paso a paso para levantar el cluster, desplegar el operador y ejecutar los escenarios de carga con k6.

## Prerrequisitos

| Herramienta | Versión mínima | Verificación |
|-------------|----------------|--------------|
| Docker | 24.x | `docker version` |
| kind | 0.20+ | `kind version` |
| kubectl | 1.28+ | `kubectl version --client` |
| Go | 1.22+ | `go version` |
| make | 4.x | `make --version` |

> Si falta alguna, el target `make load-up` aborta con mensaje explícito antes de tocar nada.

## Flujo recomendado (un comando por paso)

```bash
make load-up        # 1. Crea kind si no existe; aprueba CSRs; nodos Ready
make load-deploy    # 2. Build + deploy operator + monitoring (Prom + Grafana + InfluxDB) + workloads
make load-bin       # 3. Construye k6 con xk6-kubernetes en loadtest/bin/k6 (1ª vez)
make load-prom-pf   # 4. Port-forward Prometheus :9090
make load-grafana   # 5. Port-forward Grafana + abre navegador

# Streaming en vivo a Grafana (recomendado):
make load-influx-pf                       # 6. Port-forward InfluxDB :8086
make load-steady INFLUX=true VUS=10       # 7. k6 streamea métricas a Grafana en vivo
                                          #    Dashboard: "k6 Load Tests — Chaos Operator"

# Reportes históricos navegables:
make load-reports-open                    # 8. Despliega nginx + publica HTMLs + abre :8088

make load-clean     # 9. Borra CRs de carga
make load-down      # 10. Elimina workloads + todos los port-forwards
```

Cada target es idempotente: re-ejecutar uno sin cambios no rompe el estado anterior.

## ¿Qué hace cada paso?

### `make load-up`

Detecta si el cluster `chaos-testing-v2` ya existe (`kind get clusters`). Si no, lo crea con `kind-cluster.yaml`, aprueba los CSRs pendientes del kubelet (necesarios porque `kind-cluster.yaml` activa `serverTLSBootstrap: true`), cambia el contexto kubectl al cluster y espera a que todos los nodos estén Ready (timeout 120s).

### `make load-deploy`

Encadena: `make generate generate-crds`, `docker build -t goland-operator:latest .`, `kind load docker-image`, `kubectl apply -f config/crd/`, `kubectl apply -f config/rbac/`, deploy del operador, deploy de Prometheus + Grafana, y deploy de los workloads de carga (`loadtest/workloads/load-target.yaml`). Hace `kubectl rollout restart` del operador para que tome la imagen recién construida.

### `make load-bin`

Instala `xk6` con `go install go.k6.io/xk6/cmd/xk6@latest` si no existe, y construye un binario `k6` con la extensión `xk6-kubernetes` en `loadtest/bin/k6`. El binario sólo se reconstruye si no existe.

### `make load-prom-pf`

Inicia (o reutiliza) un port-forward de Prometheus a `localhost:9090` en background. Guarda el PID en `/tmp/.chaos-prom-pf.pid` para poder reusar/limpiar.

### Escenarios

| Escenario | VUs | Duración | Carga aproximada | SLOs |
|-----------|-----|----------|------------------|------|
| `load-smoke`  | 1   | 30 s     | ~10 CRs           | p95 reconcile < 5 s, 0 fallos |
| `load-steady` | 5   | 5 min    | ~250 CRs (50/min) | p95 reconcile < 3 s, error rate < 0.5/s, p95 prom < 1.5 s |
| `load-spike`  | 0→100→0 | ~45 s | ~200 CRs en 30 s  | apply errors < 20, p95 apply < 2 s |

Variables overridables: `make load-steady VUS=10 DURATION=10m PROM_URL=http://localhost:9090`.

### Visualización de resultados

Hay tres formas de ver lo que pasó durante un test, complementarias:

**Tiempo real en Grafana (durante el test):** corré con `INFLUX=true` para que k6 streamee a InfluxDB. Tu Grafana ya tiene el datasource `InfluxDB-k6` configurado (provisioning declarativo en `monitoring/grafana-deployment.yaml`) y un dashboard nuevo `k6 Load Tests — Chaos Operator` (`monitoring/k6-loadtest-dashboard.json`). Mientras el test corre, ves VUs, throughput, p95/p99 de reconciliación, y al lado las métricas internas del operator (Prometheus). Si querés que k6 NO streamee (corrida silenciosa), no pases `INFLUX=true`.

**Reportes HTML por corrida (post-test):** cada escenario emite seis archivos en `loadtest/results/` vía `lib/report.js`: `<scenario>-<timestamp>.{json,html,md}` + `<scenario>-latest.{json,html}`. Los HTML son self-contained con cards de KPIs, latencias y badge PASS/FAIL.

**Reports server agregador (archivo histórico):** `make load-reports-open` despliega un nginx en el cluster, genera un `index.html` con todas las corridas anteriores ordenadas por fecha (con badges PASS/FAIL), copia los HTML al pod y abre el navegador en `localhost:8088`. Útil para comparar corridas a lo largo del tiempo y compartir con el equipo durante una review.

### Limpieza

`make load-clean` borra solo CRs con label `test.chaos.engineering.io/scenario=load`. `make load-down` además elimina el deployment de carga y todos los port-forwards (Grafana, Prometheus, InfluxDB, reports). `make load-teardown` destruye el cluster kind por completo.

## Troubleshooting

### Pods en `ErrImagePull` o `ImagePullBackOff` durante `load-deploy`

Causas más frecuentes (en orden de probabilidad en kind local):

1. **Rate-limit de Docker Hub.** Pulls anónimos están limitados a 100 cada 6 horas por IP. Solución: `docker login` (aumenta a 200/6h gratis), o esperar a que se reinicie la ventana.
2. **Tag deprecado.** Las imágenes están pinneadas en `loadtest/load.mk` bajo `LOAD_EXTERNAL_IMAGES`. Si Docker Hub deja de publicar uno, hay que actualizar la versión ahí.
3. **Sin conectividad.** `docker pull <imagen>` desde el host falla → la red del Docker daemon no resuelve `registry-1.docker.io`.

### `kind load` falla con `ctr: content digest sha256:...: not found`

Causa raíz: kind v0.20+ corre `ctr images import --all-platforms --digests` que valida TODOS los digests del manifest list multi-arch. Cuando `docker pull` trae una imagen multi-arch (lo hace por default), el daemon guarda la metadata del manifest list referenciando digests de todas las arquitecturas (amd64, arm64, ppc64le, s390x…), pero solo descarga los layers de tu plataforma. `docker save` arrastra esa metadata al tarball, y ctr revienta porque no encuentra los digests de las otras arquitecturas.

**No es suficiente** con cambiar `kind load docker-image` por `kind load image-archive` — ambos fallan igual.

La solución que sí funciona está implementada en el target `load-images-prepull`:

```bash
docker rmi -f IMAGE                          # limpia el caché multi-arch
docker pull --platform=linux/amd64 IMAGE    # fuerza pull single-arch puro
docker save IMAGE -o tarball                # tarball ahora SIN manifest list
kind load image-archive tarball             # ctr lo importa sin error
```

El `--platform` detecta tu arquitectura desde `docker info` y mapea `x86_64`→`amd64`, `aarch64`→`arm64`. Si te aparece el error después de actualizar el Makefile, re-ejecutá:

```bash
make load-images-prepull
```

Issue de kind: [kubernetes-sigs/kind#3795](https://github.com/kubernetes-sigs/kind/issues/3795).

Recovery:

```bash
# Ver qué pod falló y por qué
kubectl describe pod -n monitoring -l app=influxdb | tail -20

# Pre-cargar manualmente las imágenes en kind (no requiere reconstruir el cluster)
make load-images-prepull

# Forzar nuevo intento del pod
kubectl delete pod -n monitoring -l app=influxdb
kubectl rollout status -n monitoring deployment/influxdb --timeout=120s
```

Si seguís sin pull desde el host, agregá tu cuenta Docker:

```bash
docker login
make load-images-prepull
```

### El binario k6 falla con "kubernetes is not a registered extension"

`load-bin` no terminó. Forzar reconstrucción:

```bash
rm -f loadtest/bin/k6
make load-bin
```

### Los CRs quedan en Phase=Pending

Probable causa: el deployment `load-target` no tiene pods Ready. Verificar:

```bash
kubectl get pods -n chaos-load -l app=load-target
kubectl describe deployment load-target -n chaos-load
```

### El operator entra en CrashLoopBackOff durante `load-spike`

Capturar logs y métricas antes de reproducir:

```bash
kubectl logs -n chaos-system deployment/chaos-operator --previous > /tmp/operator-prev.log
make load-prom-snap > /tmp/prom-snapshot.txt
```

Reportar como defecto con ambos archivos adjuntos.

### Prometheus no responde en `:9090`

```bash
ps aux | grep "kubectl port-forward.*prometheus" | grep -v grep
# Si no hay proceso:
make load-prom-pf
# Si hay zombie:
rm -f /tmp/.chaos-prom-pf.pid && make load-prom-pf
```

### Webhook rechaza los CRs con error de TLS

El webhook necesita su propio CA bundle. Verificar:

```bash
kubectl get validatingwebhookconfiguration | grep chaos
kubectl describe validatingwebhookconfiguration <nombre>
```

Si la fecha del certificado venció, redeployar:

```bash
kubectl delete validatingwebhookconfiguration --all -l app=chaos-operator
make load-deploy
```

## Consideraciones de seguridad

Todos los CRs de carga apuntan únicamente al namespace `chaos-load` con `labelSelectors.app=load-target` y `blastRadius.maxPods=1`, por lo que el impacto de cualquier escenario está acotado a ese deployment.

El binario k6 usa el kubeconfig del usuario (`$KUBECONFIG`, default `~/.kube/config`) — no se hardcodean tokens ni se desactiva la verificación TLS.

El stack de monitoring queda accesible solo vía port-forward; no se expone a la red externa.

Las queries a Prometheus están en una allowlist en `loadtest/scripts/lib/prom.js`; no se construyen expresiones PromQL a partir de input no validado.

## Referencias

- Plan formal de pruebas de carga: `Plan_Pruebas_Carga_ChaosOperator.docx` (en outputs).
- Plan general de pruebas: `Plan_Pruebas_ChaosOperator.docx`, `RTM_Plan_Pruebas_ChaosOperator.xlsx`.
- xk6-kubernetes: https://github.com/grafana/xk6-kubernetes
- k6 docs: https://grafana.com/docs/k6/latest/
