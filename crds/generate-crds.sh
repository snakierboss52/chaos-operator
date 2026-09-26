#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Generating CRDs...${NC}"

# Determine GOPATH
GOPATH=$(go env GOPATH)
CONTROLLER_GEN="${GOPATH}/bin/controller-gen"

# Check if controller-gen is installed
if [ ! -f "${CONTROLLER_GEN}" ]; then
    echo -e "${YELLOW}controller-gen not found. Installing...${NC}"
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
fi

# Generate CRDs
"${CONTROLLER_GEN}" crd:crdVersions=v1 paths="./api/..." output:crd:artifacts:config=config/crd

echo -e "${GREEN}CRDs generated successfully in config/crd/${NC}"
