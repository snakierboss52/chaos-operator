#!/bin/bash

set -e

echo "========================================="
echo "  Installing Monitoring Stack"
echo "========================================="

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Step 1/4: Creating monitoring namespace...${NC}"
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

echo -e "${YELLOW}Step 2/4: Deploying Prometheus...${NC}"
kubectl apply -f prometheus-config.yaml
kubectl apply -f prometheus-deployment.yaml

echo -e "${YELLOW}Step 3/4: Deploying Grafana...${NC}"
kubectl apply -f grafana-dashboards-configmap.yaml
kubectl apply -f grafana-deployment.yaml

echo -e "${YELLOW}Step 4/4: Waiting for pods to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=120s
kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=120s

echo ""
echo -e "${GREEN}========================================="
echo "  Monitoring Stack Installed!"
echo "=========================================${NC}"
echo ""
echo "Access URLs (for kind cluster):"
echo ""
echo "  Prometheus: http://localhost:30090"
echo "  Grafana:    http://localhost:30030"
echo ""
echo "Grafana Credentials:"
echo "  Username: admin"
echo "  Password: admin"
echo ""
echo "To access from your browser (if using kind):"
echo "  kubectl port-forward -n monitoring svc/prometheus 9090:9090"
echo "  kubectl port-forward -n monitoring svc/grafana 3000:3000"
echo ""
echo "Dashboard: 'Chaos Engineering Dashboard' should be auto-imported"
echo ""
