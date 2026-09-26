#!/bin/bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "=================================="
echo "  Chaos Operator Installation"
echo "=================================="
echo -e "${NC}"

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

# Check kubectl connectivity
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
    echo "Please ensure kubectl is configured correctly"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}\n"

# Get cluster info
CLUSTER_CONTEXT=$(kubectl config current-context)
echo -e "${BLUE}Current cluster context: ${CLUSTER_CONTEXT}${NC}"

# Confirm installation
read -p "Do you want to install the Chaos Operator to this cluster? (yes/no) " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Installation cancelled"
    exit 0
fi

# Step 1: Generate CRDs
echo -e "\n${YELLOW}Step 1/4: Generating CRDs...${NC}"
make generate-crds
echo -e "${GREEN}✓ CRDs generated${NC}"

# Step 2: Build Docker image
echo -e "\n${YELLOW}Step 2/4: Building Docker image...${NC}"

# Detect cluster type
if [[ $CLUSTER_CONTEXT == *"minikube"* ]]; then
    echo "Detected Minikube cluster"
    eval $(minikube docker-env)
    make docker-build IMG=goland-operator:latest
elif [[ $CLUSTER_CONTEXT == *"kind"* ]]; then
    echo "Detected Kind cluster"
    make docker-build IMG=goland-operator:latest
    kind load docker-image goland-operator:latest
else
    echo "Building for remote cluster"
    make docker-build IMG=goland-operator:latest
    echo -e "${YELLOW}Note: You may need to push the image to a registry${NC}"
    echo -e "${YELLOW}Run: make docker-push IMG=your-registry/goland-operator:latest${NC}"
    read -p "Have you pushed the image to a registry? (yes/no) " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo -e "${YELLOW}Please push the image and update config/manager/deployment.yaml${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}✓ Docker image ready${NC}"

# Step 3: Install CRDs
echo -e "\n${YELLOW}Step 3/4: Installing CRDs to cluster...${NC}"
make install-crds
echo -e "${GREEN}✓ CRDs installed${NC}"

# Step 4: Deploy operator
echo -e "\n${YELLOW}Step 4/4: Deploying operator...${NC}"
make deploy

echo -e "\n${GREEN}=================================="
echo "  Installation Complete! 🎉"
echo "==================================${NC}\n"

echo "Check operator status:"
echo "  kubectl get pods -n chaos-system"
echo ""
echo "View logs:"
echo "  kubectl logs -n chaos-system -l app=chaos-operator -f"
echo ""
echo "Try a sample experiment:"
echo "  kubectl apply -f samples/podchaos/basic-kill-one.yaml"
