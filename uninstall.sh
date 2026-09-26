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
echo "  Chaos Operator Uninstallation"
echo "=================================="
echo -e "${NC}"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

# Get cluster info
CLUSTER_CONTEXT=$(kubectl config current-context)
echo -e "${BLUE}Current cluster context: ${CLUSTER_CONTEXT}${NC}"

# Confirm uninstallation
echo -e "${YELLOW}This will remove the Chaos Operator and all CRDs from the cluster.${NC}"
echo -e "${RED}WARNING: All chaos experiments will be deleted!${NC}"
read -p "Are you sure you want to continue? (yes/no) " -r
echo
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Uninstallation cancelled"
    exit 0
fi

# Step 1: Delete all chaos experiments
echo -e "\n${YELLOW}Step 1/3: Deleting all chaos experiments...${NC}"
kubectl delete podchaos,networkchaos,stresschaos,httpchaos --all --all-namespaces --ignore-not-found=true
echo -e "${GREEN}✓ Chaos experiments deleted${NC}"

# Step 2: Undeploy operator
echo -e "\n${YELLOW}Step 2/3: Removing operator...${NC}"
make undeploy
echo -e "${GREEN}✓ Operator removed${NC}"

# Step 3: Uninstall CRDs
echo -e "\n${YELLOW}Step 3/3: Uninstalling CRDs...${NC}"
make uninstall-crds
echo -e "${GREEN}✓ CRDs uninstalled${NC}"

echo -e "\n${GREEN}=================================="
echo "  Uninstallation Complete!"
echo "==================================${NC}\n"
