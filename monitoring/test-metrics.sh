#!/bin/bash

set -e

echo "========================================="
echo "  Testing Chaos Metrics"
echo "========================================="
echo ""

# Check if operator is running
echo "1. Checking operator status..."
OPERATOR_POD=$(kubectl get pods -n chaos-system -l app=chaos-operator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$OPERATOR_POD" ]; then
    echo "❌ Operator not found. Please start the operator first."
    echo "   Run: make run"
    exit 1
fi

echo "✅ Operator is running: $OPERATOR_POD"
echo ""

# Check if prometheus is running
echo "2. Checking Prometheus status..."
if kubectl get deployment prometheus-server -n monitoring >/dev/null 2>&1; then
	kubectl wait --for=condition=available deployment/prometheus-server -n monitoring --timeout=60s >/dev/null
    echo "✅ Prometheus is running"
else
    echo "❌ Prometheus not running. Run: ./monitoring/install-monitoring.sh"
    exit 1
fi
echo ""

# Create test experiment
echo "3. Creating test chaos experiment..."
cat <<EOF | kubectl apply -f -
apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: metrics-test
  namespace: chaos-demo
spec:
  mode: one
  selector:
    namespaces:
      - chaos-demo
    labelSelectors:
      app: nginx
  action: pod-kill
  duration: "10s"
EOF

echo "✅ Experiment created"
echo ""

# Wait for experiment to execute
echo "4. Waiting for experiment to execute (15 seconds)..."
sleep 15
echo ""

# Check metrics from operator directly
echo "5. Checking metrics from operator..."
echo ""
kubectl port-forward -n default $OPERATOR_POD 8080:8080 > /dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 2

echo "Fetching metrics..."
METRICS=$(curl -s http://localhost:8080/metrics 2>/dev/null)

kill $PORT_FORWARD_PID 2>/dev/null

if [ -z "$METRICS" ]; then
    echo "❌ Could not fetch metrics"
    exit 1
fi

echo ""
echo "✅ Chaos Metrics Found:"
echo ""
echo "$METRICS" | grep "chaos_" | head -20
echo ""
echo "..."
echo ""

# Query Prometheus
echo "6. Querying Prometheus..."
kubectl port-forward -n monitoring svc/prometheus-server 9090:80 > /dev/null 2>&1 &
PROM_PID=$!
sleep 3

echo ""
echo "Total experiments:"
curl -s 'http://localhost:9090/api/v1/query?query=sum(chaos_experiments_total)' | grep -o '"result":\[[^]]*\]' || echo "No data yet"

echo ""
echo "Active experiments:"
curl -s 'http://localhost:9090/api/v1/query?query=sum(chaos_experiments_active)' | grep -o '"result":\[[^]]*\]' || echo "No data yet"

echo ""
echo "Targets affected:"
curl -s 'http://localhost:9090/api/v1/query?query=sum(chaos_targets_affected_total)' | grep -o '"result":\[[^]]*\]' || echo "No data yet"

kill $PROM_PID 2>/dev/null
echo ""

# Cleanup
echo ""
echo "7. Cleaning up test experiment..."
kubectl delete podchaos metrics-test -n chaos-demo 2>/dev/null || true
echo "✅ Cleanup complete"
echo ""

echo "========================================="
echo "  ✅ Test Complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "  1. Run: ./access-monitoring.sh"
echo "  2. Open Grafana: http://localhost:3000"
echo "  3. View the 'Chaos Engineering Dashboard'"
echo "  4. Create more experiments to see metrics populate"
echo ""
