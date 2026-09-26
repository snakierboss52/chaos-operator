#!/bin/bash
set -euo pipefail

echo "Generating CRD manifests..."

if ! command -v controller-gen &> /dev/null; then
    echo "Installing controller-gen..."
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
fi

CONTROLLER_GEN=$(which controller-gen 2>/dev/null || echo "$HOME/go/bin/controller-gen")

$CONTROLLER_GEN crd paths="./api/..." output:crd:artifacts:config=config/crd

echo "CRD manifests generated in config/crd/"
