#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
KIND_CLUSTER="${KIND_CLUSTER:-chaos-testing-v2}"
CONTEXT="$(kubectl config current-context)"

for dependency in helm kubectl; do
  command -v "$dependency" >/dev/null || { echo "Missing required command: $dependency" >&2; exit 1; }
done
if [[ "$CONTEXT" != "kind-$KIND_CLUSTER" ]]; then
  echo "Refusing to install local addons: current context is '$CONTEXT', expected 'kind-$KIND_CLUSTER'." >&2
  echo "Set KIND_CLUSTER if using a differently named kind cluster." >&2
  exit 1
fi

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update
helm repo add grafana-community https://grafana-community.github.io/helm-charts --force-update
helm repo update prometheus-community grafana-community
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install kube-state-metrics prometheus-community/kube-state-metrics \
  --version "${KSM_CHART_VERSION:-8.3.0}" --namespace monitoring \
  --values "$ROOT_DIR/scripts/helms/kube-state-metrics/values.yaml" --wait --timeout 5m
helm upgrade --install prometheus prometheus-community/prometheus \
  --version "${PROMETHEUS_CHART_VERSION:-29.21.0}" --namespace monitoring \
  --values "$ROOT_DIR/scripts/helms/prometheus/values.yaml" --wait --timeout 5m
helm upgrade --install grafana grafana-community/grafana \
  --version "${GRAFANA_CHART_VERSION:-12.10.0}" --namespace monitoring \
  --values "$ROOT_DIR/scripts/helms/grafana/values.yaml" --wait --timeout 5m

# Grafana's sidecar watches labeled ConfigMaps and loads the repo dashboards.
DASHBOARDS=("$ROOT_DIR"/monitoring/*.json)
kubectl create configmap grafana-dashboards --namespace monitoring \
  "${DASHBOARDS[@]/#/--from-file=}" --dry-run=client -o yaml \
  | kubectl label --local -f - grafana_dashboard=1 -o yaml \
  | kubectl apply -f -

# k6's InfluxDB output remains a project-specific local addon, currently
# supplied as a hardened manifest rather than a Helm chart.
kubectl apply -f "$ROOT_DIR/monitoring/influxdb-deployment.yaml"
kubectl wait --for=condition=available deployment/influxdb -n monitoring --timeout=3m

echo "Local monitoring addons are ready in namespace monitoring."
echo "Run 'make load-prom-pf' and 'make load-grafana-pf' to open local port-forwards."
echo "Grafana credentials: admin/admin"
