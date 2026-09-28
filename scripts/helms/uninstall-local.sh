#!/usr/bin/env bash
set -Eeuo pipefail

KIND_CLUSTER="${KIND_CLUSTER:-chaos-testing-v2}"
CONTEXT="$(kubectl config current-context)"
if [[ "$CONTEXT" != "kind-$KIND_CLUSTER" ]]; then
  echo "Refusing to uninstall local addons: current context is '$CONTEXT', expected 'kind-$KIND_CLUSTER'." >&2
  exit 1
fi

for release in grafana prometheus kube-state-metrics; do
  helm uninstall "$release" --namespace monitoring --ignore-not-found
done
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
kubectl delete -f "$ROOT_DIR/scripts/helms/traefik/ingresses.yaml" --ignore-not-found
kubectl delete -f "$ROOT_DIR/scripts/helms/traefik/operator-metrics-ingress.yaml" --ignore-not-found
helm uninstall traefik --namespace traefik --ignore-not-found
kubectl delete configmap grafana-dashboards -n monitoring --ignore-not-found
kubectl delete -f "$ROOT_DIR/monitoring/influxdb-deployment.yaml" --ignore-not-found
echo "Helm monitoring releases and local InfluxDB removed; namespace monitoring was retained."
