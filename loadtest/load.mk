# load.mk — targets de pruebas de carga sobre el Chaos Operator.
#
# Diseño:
#   - Idempotente: cada target se puede correr varias veces sin efectos
#     secundarios.
#   - Aislado: todos los recursos viven en el namespace 'chaos-load' y
#     usan la etiqueta scenario=load.
#   - Seguro: el binario k6 se construye con xk6-kubernetes y queda en
#     loadtest/bin/k6 (no se sobrescribe ningún binario del sistema).
#     No usamos --insecure-skip-tls-verify; la conexión al API server
#     respeta el kubeconfig del usuario.
#
# Variables exportables:
#   KIND_CLUSTER     — nombre del cluster kind (default: chaos-testing-v2)
#   IMG              — imagen del operator (default: goland-operator:latest)
#   PROM_URL         — endpoint de Prometheus (default: http://localhost:9090)
#   VUS, DURATION    — overrides para el escenario steady

KIND_CLUSTER ?= chaos-testing-v2
IMG ?= goland-operator:latest
PROM_URL ?= http://localhost:9090
VUS ?= 5
DURATION ?= 5m

LOAD_DIR := loadtest
LOAD_BIN := $(LOAD_DIR)/bin/k6
LOAD_SCRIPTS := $(LOAD_DIR)/scripts
LOAD_NAMESPACE := chaos-load
LOAD_PORT_FORWARD_PID := /tmp/.chaos-prom-pf.pid

# Bash strict mode dentro del Makefile.
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

.PHONY: load-help
load-help:
	@echo "Pruebas de carga — Chaos Engineering Operator"
	@echo ""
	@echo "Setup (idempotente):"
	@echo "  make load-up              - Crea cluster kind si no existe; espera nodes Ready."
	@echo "  make load-images-prepull  - Docker pull + kind load de imágenes externas."
	@echo "  make load-deploy          - Pre-pull + build + load image + deploy operator + monitoring + workloads."
	@echo "  make load-bin             - Construye k6 con xk6-kubernetes (loadtest/bin/k6)."
	@echo "  make load-prom-pf         - Inicia port-forward de Prometheus a localhost:9090."
	@echo ""
	@echo "Escenarios:"
	@echo "  make load-smoke       - 1 VU x 30s. Sanity check del setup."
	@echo "  make load-steady      - $(VUS) VUs x $(DURATION). Carga sostenida nominal."
	@echo "  make load-spike       - Ramp 0->100 VUs en 10s, hold 30s. Pico súbito."
	@echo "    → tip: agregá INFLUX=true para streamear métricas a Grafana en vivo."
	@echo "    → ej:  make load-steady INFLUX=true VUS=10"
	@echo ""
	@echo "Reportes (cada escenario produce JSON + HTML + MD en loadtest/results/):"
	@echo "  make load-report          - Lista los reportes generados (más recientes primero)."
	@echo "  make load-report-open     - Abre el último reporte HTML local en el navegador."
	@echo "  make load-report-clean    - Borra todos los reportes anteriores."
	@echo "  make load-reports-deploy  - Despliega nginx con los HTML reports (cluster)."
	@echo "  make load-reports-publish - Genera index agregador y copia HTML al pod."
	@echo "  make load-reports-open    - Publish + port-forward + abre en navegador."
	@echo "  make load-reports-stop    - Detiene port-forward del reports server."
	@echo ""
	@echo "Grafana / Prometheus / InfluxDB:"
	@echo "  make load-prom-snap      - Snapshot puntual de métricas clave del operator."
	@echo "  make load-grafana        - Port-forward + abre Grafana (one-shot)."
	@echo "  make load-grafana-pf     - Solo port-forward Grafana a :3000."
	@echo "  make load-grafana-reload - Recarga dashboards desde monitoring/*.json."
	@echo "  make load-grafana-stop   - Detiene el port-forward de Grafana."
	@echo "  make load-influx-pf      - Port-forward InfluxDB a :8086 (k6 streaming)."
	@echo "  make load-influx-stop    - Detiene el port-forward de InfluxDB."
	@echo "  make load-influx-truncate- Borra todas las series de la DB k6."
	@echo ""
	@echo "Limpieza:"
	@echo "  make load-clean       - Borra CRs de carga (label scenario=load) del namespace $(LOAD_NAMESPACE)."
	@echo "  make load-down        - Elimina workloads y namespace de carga; mantiene operator."
	@echo "  make load-teardown    - Destruye TODO (cluster kind incluido). Usar con cuidado."

# ---------------------------------------------------------------
# Setup
# ---------------------------------------------------------------

.PHONY: load-up
load-up:
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/kind/up.sh

.PHONY: load-images-prepull
load-images-prepull: load-up
	@# Por qué este target hace lo que hace:
	@# kind v0.20+ corre 'ctr images import --all-platforms --digests' que
	@# valida TODOS los digests del manifest list multi-arch. Cuando 'docker
	@# pull' trae una imagen multi-arch (default sin --platform), el daemon
	@# guarda la metadata del manifest list, y 'docker save' la incluye en
	@# el tarball — incluso cuando solo tenés los layers de tu plataforma.
	@# ctr falla porque los digests de las otras plataformas no existen.
	@#
	@# Fix: rmi + pull --platform=<host_arch>. Esto fuerza al daemon a
	@# guardar SOLO la imagen single-arch sin la referencia al manifest list.
	@# El tarball resultante es plano y kind lo importa sin error.
	@# Issue: kubernetes-sigs/kind#3795.
	@echo "→ Pre-pulling y cargando imágenes externas en kind..."
	@ARCH=$$(docker info --format '{{.Architecture}}' 2>/dev/null | sed 's/x86_64/amd64/;s/aarch64/arm64/'); \
	PLATFORM="linux/$${ARCH:-amd64}"; \
	echo "  Plataforma detectada: $$PLATFORM"; \
	TMPDIR_KIND=$$(mktemp -d -t kind-img-XXXXXX); \
	trap 'rm -rf "$$TMPDIR_KIND"' EXIT; \
	for img in $(LOAD_EXTERNAL_IMAGES); do \
		echo ""; \
		echo "  ──── $$img ────"; \
		echo "  → docker rmi (limpia manifest list multi-arch cacheado)"; \
		docker rmi -f "$$img" 2>/dev/null || true; \
		echo "  → docker pull --platform=$$PLATFORM (single-arch limpio)"; \
		docker pull --platform="$$PLATFORM" "$$img" || { \
			echo "  ❌ Error pulling $$img. Posibles causas:"; \
			echo "     - Sin red / rate-limit Docker Hub ('docker login' sube a 200/6h)"; \
			echo "     - Imagen movida o tag deprecado"; \
			exit 1; \
		}; \
		safe=$$(echo "$$img" | tr '/:' '__'); \
		tarball="$$TMPDIR_KIND/$$safe.tar"; \
		echo "  → docker save → tarball plano sin manifest list"; \
		docker save -o "$$tarball" "$$img"; \
		echo "  → kind load image-archive"; \
		kind load image-archive "$$tarball" --name $(KIND_CLUSTER); \
		echo "  ✓ $$img cargada en kind."; \
	done
	@echo ""
	@echo "✓ Imágenes externas cargadas en el nodo de kind."

.PHONY: load-deploy
load-deploy: load-up load-images-prepull
	@echo "→ Generando código y CRDs..."
	@$(MAKE) generate generate-crds
	@echo "→ Build de imagen $(IMG)..."
	@docker build -t $(IMG) .
	@echo "→ Cargando imagen en kind..."
	@kind load docker-image $(IMG) --name $(KIND_CLUSTER)
	@echo "→ Aplicando CRDs..."
	@kubectl apply -f config/crd/
	@echo "→ Desplegando operator..."
	@kubectl apply -f config/manager/namespace.yaml
	@kubectl apply -f config/rbac/
	@kubectl apply -f config/manager/deployment.yaml
	@kubectl apply -f config/manager/service.yaml
	@kubectl rollout restart deployment/chaos-operator -n chaos-system 2>/dev/null || true
	@kubectl wait --for=condition=available --timeout=180s deployment/chaos-operator -n chaos-system
	@echo "→ Desplegando addons de monitoreo con Helm..."
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/helms/install-local.sh
	@echo "→ Cargando dashboards individuales del repo en Grafana..."
	@$(MAKE) load-grafana-reload
	@echo "→ Desplegando workloads de carga (namespace $(LOAD_NAMESPACE))..."
	@kubectl apply -f $(LOAD_DIR)/workloads/load-target.yaml
	@kubectl wait --for=condition=available --timeout=180s deployment/load-target -n $(LOAD_NAMESPACE)
	@echo "✅ load-deploy completado. Operator + monitoring + workloads listos."

.PHONY: load-bin
load-bin: $(LOAD_BIN)

$(LOAD_BIN):
	@command -v go >/dev/null || { echo "❌ Go no está instalado"; exit 1; }
	@if ! command -v xk6 >/dev/null; then \
		echo "→ Instalando xk6..."; \
		go install go.k6.io/xk6/cmd/xk6@latest; \
	fi
	@mkdir -p $(LOAD_DIR)/bin
	@echo "→ Construyendo k6 con xk6-kubernetes..."
	@cd $(LOAD_DIR)/bin && \
		"$$(go env GOPATH)/bin/xk6" build \
			--with github.com/grafana/xk6-kubernetes \
			--output k6
	@echo "✓ k6 construido en $(LOAD_BIN)"

.PHONY: load-prom-pf
load-prom-pf:
	@if [ -f $(LOAD_PORT_FORWARD_PID) ] && kill -0 "$$(cat $(LOAD_PORT_FORWARD_PID))" 2>/dev/null; then \
		echo "✓ Port-forward de Prometheus ya activo (pid $$(cat $(LOAD_PORT_FORWARD_PID)))"; \
	else \
		echo "→ Iniciando port-forward Prometheus :9090..."; \
		kubectl port-forward -n monitoring svc/prometheus-server 9090:80 >/dev/null 2>&1 & \
		echo $$! > $(LOAD_PORT_FORWARD_PID); \
		sleep 2; \
		echo "✓ Prometheus accesible en $(PROM_URL) (pid $$(cat $(LOAD_PORT_FORWARD_PID)))"; \
	fi

# ---------------------------------------------------------------
# Escenarios
# ---------------------------------------------------------------

.PHONY: load-smoke
load-smoke: load-bin load-prom-pf
	@echo "→ Escenario SMOKE (30s, 1 VU)..."
	@KUBECONFIG="$${KUBECONFIG:-$$HOME/.kube/config}" \
		PROM_URL=$(PROM_URL) \
		$(LOAD_BIN) run $(LOAD_K6_OUT) $(LOAD_SCRIPTS)/00-smoke.js

.PHONY: load-steady
load-steady: load-bin load-prom-pf
	@echo "→ Escenario STEADY (VUs=$(VUS), duración=$(DURATION))..."
	@KUBECONFIG="$${KUBECONFIG:-$$HOME/.kube/config}" \
		PROM_URL=$(PROM_URL) \
		VUS=$(VUS) \
		DURATION=$(DURATION) \
		$(LOAD_BIN) run $(LOAD_K6_OUT) $(LOAD_SCRIPTS)/01-steady.js

.PHONY: load-spike
load-spike: load-bin load-prom-pf
	@echo "→ Escenario SPIKE (0→100 VUs en 10s, hold 30s)..."
	@KUBECONFIG="$${KUBECONFIG:-$$HOME/.kube/config}" \
		PROM_URL=$(PROM_URL) \
		$(LOAD_BIN) run $(LOAD_K6_OUT) $(LOAD_SCRIPTS)/02-spike.js

# ---------------------------------------------------------------
# Snapshot y limpieza
# ---------------------------------------------------------------

.PHONY: load-prom-snap
load-prom-snap: load-prom-pf
	@echo "=== Snapshot de métricas del operator ==="
	@for q in 'sum(chaos_experiments_active)' \
		'histogram_quantile(0.95, rate(chaos_reconciliation_duration_seconds_bucket[1m]))' \
		'sum(rate(chaos_execution_errors_total[1m]))' \
		'go_goroutines{job=~"chaos-operator.*"}' \
		'process_resident_memory_bytes{job=~"chaos-operator.*"}'; do \
		echo "--- $$q ---"; \
		curl -s --get --data-urlencode "query=$$q" "$(PROM_URL)/api/v1/query" | python3 -m json.tool 2>/dev/null || echo "(error)"; \
	done

# ---------------------------------------------------------------
# Grafana (dashboards desde monitoring/)
# ---------------------------------------------------------------
# Allowlist explícita de dashboards JSON. Cualquier dashboard que se quiera
# incluir debe agregarse aquí — defense-in-depth contra cargar JSON arbitrario
# o de procedencia no validada en el ConfigMap del cluster.
LOAD_GRAFANA_DASHBOARDS := \
	monitoring/chaos-dashboard.json \
	monitoring/chaos-kind-dashboard.json \
	monitoring/operator-overview-dashboard.json \
	monitoring/simple-dashboard.json \
	monitoring/k6-loadtest-dashboard.json
LOAD_GRAFANA_PF_PID := /tmp/.chaos-grafana-pf.pid
LOAD_INFLUX_PF_PID := /tmp/.chaos-influx-pf.pid

# k6 InfluxDB output: solo se activa si INFLUX=true en el ambiente.
# Por defecto usamos --out=null para no requerir port-forward.
ifeq ($(INFLUX),true)
LOAD_K6_OUT := --out influxdb=http://localhost:8086/k6
else
LOAD_K6_OUT :=
endif

# Imágenes externas que el cluster kind necesita.
# Pinneadas a patch inmutable. Pre-cargadas con `kind load docker-image` para:
#   - evitar rate-limit de Docker Hub (100 pulls anónimos / 6h)
#   - permitir trabajar offline tras la primera carga
#   - reducir tiempo de despliegue (la imagen ya está en el nodo)
LOAD_EXTERNAL_IMAGES := \
	influxdb:1.8.10 \
	nginxinc/nginx-unprivileged:1.27.1-alpine \
	nginx:1.27.1-alpine

.PHONY: load-grafana-reload
load-grafana-reload:
	@echo "→ Validando dashboards declarados..."
	@for f in $(LOAD_GRAFANA_DASHBOARDS); do \
		if [ ! -f "$$f" ]; then echo "❌ No existe: $$f"; exit 1; fi; \
		python3 -c "import json,sys; json.load(open('$$f'))" 2>/dev/null \
			|| { echo "❌ JSON inválido: $$f"; exit 1; }; \
	done
	@echo "✓ Todos los dashboards son JSON válido."
	@echo "→ Reconstruyendo ConfigMap 'grafana-dashboards' (montado en /var/lib/grafana/dashboards)..."
	@kubectl create configmap grafana-dashboards \
		--namespace=monitoring \
		--dry-run=client \
		--labels=grafana_dashboard=1 \
		$(foreach f,$(LOAD_GRAFANA_DASHBOARDS),--from-file=$(notdir $(f))=$(f)) \
		-o yaml | kubectl apply -f -
	@echo "✓ Sidecar de Grafana recargará los dashboards automáticamente."
	@echo "✅ Dashboards recargados:"
	@for f in $(LOAD_GRAFANA_DASHBOARDS); do echo "    • $$f"; done
	@echo "  Abrí Grafana con: make load-grafana-pf  →  http://localhost:3000 (admin/admin)"

.PHONY: load-grafana-pf
load-grafana-pf:
	@# Idempotente: si ya hay un port-forward activo, lo reutiliza.
	@# kubectl port-forward bindea en 127.0.0.1 por defecto (no expone fuera del host).
	@if [ -f $(LOAD_GRAFANA_PF_PID) ] && kill -0 "$$(cat $(LOAD_GRAFANA_PF_PID))" 2>/dev/null; then \
		echo "✓ Port-forward de Grafana ya activo (pid $$(cat $(LOAD_GRAFANA_PF_PID))). http://localhost:3000"; \
	else \
		echo "→ Verificando que el deployment de Grafana esté listo..."; \
		kubectl get deployment grafana -n monitoring >/dev/null 2>&1 \
			|| { echo "❌ Grafana no está desplegado. Corré primero 'make load-deploy'."; exit 1; }; \
		kubectl wait --for=condition=available --timeout=60s deployment/grafana -n monitoring >/dev/null \
			|| { echo "❌ Grafana no llegó a Available en 60s."; exit 1; }; \
		echo "→ Iniciando port-forward Grafana :3000..."; \
		kubectl port-forward -n monitoring svc/grafana 3000:80 >/dev/null 2>&1 & \
		echo $$! > $(LOAD_GRAFANA_PF_PID); \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if curl -fsS -o /dev/null --max-time 1 http://localhost:3000/api/health 2>/dev/null; then \
				echo "✓ Grafana respondiendo en http://localhost:3000  (admin/admin)"; \
				break; \
			fi; \
			sleep 1; \
		done; \
	fi

.PHONY: load-grafana
load-grafana: load-grafana-pf
	@# Mega-target: levanta el port-forward (si no existe) y abre el navegador.
	@echo "→ Abriendo Grafana en el navegador..."
	@if command -v open >/dev/null; then open http://localhost:3000; \
	elif command -v xdg-open >/dev/null; then xdg-open http://localhost:3000; \
	else echo "  (abrí manualmente http://localhost:3000 en tu navegador)"; fi

.PHONY: load-grafana-stop
load-grafana-stop:
	@if [ -f $(LOAD_GRAFANA_PF_PID) ]; then \
		kill "$$(cat $(LOAD_GRAFANA_PF_PID))" 2>/dev/null || true; \
		rm -f $(LOAD_GRAFANA_PF_PID); \
		echo "✓ Port-forward Grafana detenido."; \
	else \
		echo "(sin port-forward activo)"; \
	fi

# ---------------------------------------------------------------
# InfluxDB (backend de k6 streaming en vivo)
# ---------------------------------------------------------------
.PHONY: load-influx-pf
load-influx-pf:
	@if [ -f $(LOAD_INFLUX_PF_PID) ] && kill -0 "$$(cat $(LOAD_INFLUX_PF_PID))" 2>/dev/null; then \
		echo "✓ Port-forward de InfluxDB ya activo (pid $$(cat $(LOAD_INFLUX_PF_PID))). http://localhost:8086"; \
	else \
		kubectl get deployment influxdb -n monitoring >/dev/null 2>&1 \
			|| { echo "❌ InfluxDB no está desplegado. Corré 'make load-deploy'."; exit 1; }; \
		kubectl wait --for=condition=available --timeout=60s deployment/influxdb -n monitoring >/dev/null \
			|| { echo "❌ InfluxDB no llegó a Available."; exit 1; }; \
		echo "→ Iniciando port-forward InfluxDB :8086..."; \
		kubectl port-forward -n monitoring svc/influxdb 8086:8086 >/dev/null 2>&1 & \
		echo $$! > $(LOAD_INFLUX_PF_PID); \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if curl -fsS -o /dev/null --max-time 1 http://localhost:8086/ping 2>/dev/null; then \
				echo "✓ InfluxDB respondiendo en http://localhost:8086 (DB: k6)"; \
				break; \
			fi; \
			sleep 1; \
		done; \
	fi

.PHONY: load-influx-stop
load-influx-stop:
	@if [ -f $(LOAD_INFLUX_PF_PID) ]; then \
		kill "$$(cat $(LOAD_INFLUX_PF_PID))" 2>/dev/null || true; \
		rm -f $(LOAD_INFLUX_PF_PID); \
		echo "✓ Port-forward InfluxDB detenido."; \
	else \
		echo "(sin port-forward de InfluxDB activo)"; \
	fi

.PHONY: load-influx-truncate
load-influx-truncate: load-influx-pf
	@echo "→ Borrando todas las series de la DB k6..."
	@curl -fsS -G http://localhost:8086/query --data-urlencode 'q=DROP SERIES FROM /.*/' --data-urlencode "db=k6" \
		| python3 -m json.tool 2>/dev/null || echo "(error truncando — verificar pf)"
	@echo "✓ Series eliminadas."

# ---------------------------------------------------------------
# Reports server (nginx con HTML reports navegables)
# ---------------------------------------------------------------
LOAD_REPORTS_PF_PID := /tmp/.chaos-reports-pf.pid

.PHONY: load-reports-deploy
load-reports-deploy:
	@echo "→ Aplicando reports server (nginx en monitoring)..."
	@kubectl apply -f $(LOAD_DIR)/reports-server/deployment.yaml
	@kubectl wait --for=condition=available --timeout=120s deployment/loadtest-reports -n monitoring

.PHONY: load-reports-publish
load-reports-publish: load-reports-deploy
	@echo "→ Generando index.html agregador..."
	@python3 $(LOAD_DIR)/scripts/build-index.py \
		--results-dir $(LOAD_DIR)/results \
		--output $(LOAD_DIR)/results/index.html
	@pod=$$(kubectl get pod -n monitoring -l app=loadtest-reports -o jsonpath='{.items[0].metadata.name}'); \
		if [ -z "$$pod" ]; then echo "❌ No se encontró el pod del reports server"; exit 1; fi; \
		echo "→ Copiando reportes al pod $$pod..."; \
		for f in $(LOAD_DIR)/results/*.html $(LOAD_DIR)/results/*.json $(LOAD_DIR)/results/*.md; do \
			[ -f "$$f" ] || continue; \
			kubectl cp -n monitoring "$$f" "$$pod:/usr/share/nginx/html/$$(basename $$f)" >/dev/null 2>&1 || true; \
		done
	@echo "✅ Reportes publicados. Abrí con: make load-reports-open"

.PHONY: load-reports-pf
load-reports-pf: load-reports-deploy
	@if [ -f $(LOAD_REPORTS_PF_PID) ] && kill -0 "$$(cat $(LOAD_REPORTS_PF_PID))" 2>/dev/null; then \
		echo "✓ Port-forward de reports ya activo (pid $$(cat $(LOAD_REPORTS_PF_PID))). http://localhost:8088"; \
	else \
		echo "→ Iniciando port-forward reports :8088..."; \
		kubectl port-forward -n monitoring svc/loadtest-reports 8088:8088 >/dev/null 2>&1 & \
		echo $$! > $(LOAD_REPORTS_PF_PID); \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if curl -fsS -o /dev/null --max-time 1 http://localhost:8088/health 2>/dev/null; then \
				echo "✓ Reports server respondiendo en http://localhost:8088"; \
				break; \
			fi; \
			sleep 1; \
		done; \
	fi

.PHONY: load-reports-open
load-reports-open: load-reports-publish load-reports-pf
	@echo "→ Abriendo http://localhost:8088 en el navegador..."
	@if command -v open >/dev/null; then open http://localhost:8088; \
	elif command -v xdg-open >/dev/null; then xdg-open http://localhost:8088; \
	else echo "  (abrí manualmente http://localhost:8088 en tu navegador)"; fi

.PHONY: load-reports-stop
load-reports-stop:
	@if [ -f $(LOAD_REPORTS_PF_PID) ]; then \
		kill "$$(cat $(LOAD_REPORTS_PF_PID))" 2>/dev/null || true; \
		rm -f $(LOAD_REPORTS_PF_PID); \
		echo "✓ Port-forward reports detenido."; \
	else \
		echo "(sin port-forward de reports activo)"; \
	fi

.PHONY: load-report
load-report:
	@echo "=== Reportes en loadtest/results/ ==="
	@if [ ! -d $(LOAD_DIR)/results ] || [ -z "$$(ls -A $(LOAD_DIR)/results 2>/dev/null | grep -v .gitkeep)" ]; then \
		echo "  (sin resultados — corré primero make load-smoke / load-steady / load-spike)"; \
	else \
		ls -lt $(LOAD_DIR)/results/*.html $(LOAD_DIR)/results/*.json $(LOAD_DIR)/results/*.md 2>/dev/null | head -30; \
	fi

.PHONY: load-report-open
load-report-open:
	@latest=$$(ls -t $(LOAD_DIR)/results/*-latest.html 2>/dev/null | head -1); \
	if [ -z "$$latest" ]; then \
		echo "❌ No hay reportes HTML. Corré primero make load-smoke / load-steady / load-spike."; \
		exit 1; \
	fi; \
	echo "→ Abriendo $$latest"; \
	if command -v open >/dev/null; then open "$$latest"; \
	elif command -v xdg-open >/dev/null; then xdg-open "$$latest"; \
	else echo "  (abrí manualmente $$latest en tu navegador)"; fi

.PHONY: load-report-clean
load-report-clean:
	@echo "→ Borrando reportes anteriores en $(LOAD_DIR)/results/ (preserva .gitkeep)..."
	@find $(LOAD_DIR)/results -type f \( -name '*.html' -o -name '*.json' -o -name '*.md' \) -delete 2>/dev/null || true
	@echo "✓ Reportes eliminados."

.PHONY: load-clean
load-clean:
	@echo "→ Borrando CRs de carga del namespace $(LOAD_NAMESPACE)..."
	@kubectl delete podchaos -n $(LOAD_NAMESPACE) -l test.chaos.engineering.io/scenario=load --ignore-not-found
	@kubectl delete networkchaos -n $(LOAD_NAMESPACE) -l test.chaos.engineering.io/scenario=load --ignore-not-found
	@kubectl delete stresschaos -n $(LOAD_NAMESPACE) -l test.chaos.engineering.io/scenario=load --ignore-not-found
	@kubectl delete httpchaos -n $(LOAD_NAMESPACE) -l test.chaos.engineering.io/scenario=load --ignore-not-found
	@echo "✓ CRs de carga eliminados."

.PHONY: load-down
load-down: load-clean load-grafana-stop load-influx-stop load-reports-stop
	@if [ -f $(LOAD_PORT_FORWARD_PID) ]; then \
		kill "$$(cat $(LOAD_PORT_FORWARD_PID))" 2>/dev/null || true; \
		rm -f $(LOAD_PORT_FORWARD_PID); \
	fi
	@kubectl delete -f $(LOAD_DIR)/workloads/load-target.yaml --ignore-not-found
	@echo "✓ Workloads de carga eliminados; operator y monitoring permanecen."

.PHONY: load-teardown
load-teardown:
	@echo "⚠  Esto destruirá el cluster kind '$(KIND_CLUSTER)' completamente."
	@KIND_CLUSTER=$(KIND_CLUSTER) scripts/kind/down.sh
	@if [ -f $(LOAD_PORT_FORWARD_PID) ]; then rm -f $(LOAD_PORT_FORWARD_PID); fi
	@echo "✓ Cluster destruido."
