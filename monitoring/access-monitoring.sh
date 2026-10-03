#!/usr/bin/env bash
set -Eeuo pipefail

PROM_PID=""
GRAFANA_PID=""
cleanup() {
  [[ -z "$PROM_PID" ]] || kill "$PROM_PID" 2>/dev/null || true
  [[ -z "$GRAFANA_PID" ]] || kill "$GRAFANA_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

kubectl wait --for=condition=available deployment/prometheus-server -n monitoring --timeout=120s
kubectl wait --for=condition=available deployment/grafana -n monitoring --timeout=120s
kubectl port-forward -n monitoring svc/prometheus-server 9090:80 >/dev/null 2>&1 &
PROM_PID=$!
kubectl port-forward -n monitoring svc/grafana 3000:80 >/dev/null 2>&1 &
GRAFANA_PID=$!

echo "Prometheus: http://localhost:9090"
echo "Grafana:    http://localhost:3000 (admin/admin)"
echo "Press Ctrl+C to stop both port-forwards."
wait "$PROM_PID" "$GRAFANA_PID"
