#!/bin/bash

echo "========================================="
echo "  Accessing Monitoring Stack"
echo "========================================="
echo ""
echo "Setting up port-forwards..."
echo ""

# Kill any existing port-forwards
pkill -f "port-forward.*prometheus" 2>/dev/null || true
pkill -f "port-forward.*grafana" 2>/dev/null || true

# Wait a moment
sleep 2

# Start port-forwards in background
echo "Starting Prometheus port-forward (9090)..."
kubectl port-forward -n monitoring svc/prometheus 9090:9090 > /dev/null 2>&1 &
PROM_PID=$!

echo "Starting Grafana port-forward (3000)..."
kubectl port-forward -n monitoring svc/grafana 3000:3000 > /dev/null 2>&1 &
GRAFANA_PID=$!

# Wait for port-forwards to establish
sleep 3

echo ""
echo "✅ Port-forwards established!"
echo ""
echo "========================================="
echo "  Access URLs"
echo "========================================="
echo ""
echo "  Prometheus: http://localhost:9090"
echo "  Grafana:    http://localhost:3000"
echo ""
echo "Grafana Credentials:"
echo "  Username: admin"
echo "  Password: admin"
echo ""
echo "Dashboard: Go to Dashboards → Browse → 'Chaos Engineering Dashboard'"
echo ""
echo "========================================="
echo "  Quick Test"
echo "========================================="
echo ""
echo "Testing Prometheus..."
curl -s http://localhost:9090/-/healthy > /dev/null && echo "✅ Prometheus is healthy" || echo "❌ Prometheus is not responding"

echo ""
echo "Testing Grafana..."
curl -s http://localhost:3000/api/health > /dev/null && echo "✅ Grafana is healthy" || echo "❌ Grafana is not responding"

echo ""
echo "========================================="
echo "  Viewing Chaos Metrics"
echo "========================================="
echo ""
echo "Query chaos metrics in Prometheus:"
echo "  http://localhost:9090/graph?g0.expr=chaos_experiments_total"
echo ""
echo "Press Ctrl+C to stop port-forwards"
echo ""

# Wait for user interrupt
trap "echo ''; echo 'Stopping port-forwards...'; kill $PROM_PID $GRAFANA_PID 2>/dev/null; exit 0" INT TERM

# Keep script running
wait
