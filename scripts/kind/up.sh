#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER:-chaos-testing-v2}"
CONFIG_FILE="${KIND_CONFIG:-kind-cluster.yaml}"
if [[ "$CONFIG_FILE" = /* ]]; then
  CONFIG_PATH="$CONFIG_FILE"
else
  CONFIG_PATH="$ROOT_DIR/$CONFIG_FILE"
fi

for dependency in kind kubectl docker; do
  command -v "$dependency" >/dev/null || { echo "Missing required command: $dependency" >&2; exit 1; }
done
[[ -f "$CONFIG_PATH" ]] || { echo "kind config file not found: $CONFIG_PATH" >&2; exit 1; }

if kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
  echo "kind cluster '$CLUSTER_NAME' already exists"
else
  kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG_PATH"
  kubectl get csr -o name | xargs -r kubectl certificate approve
fi

kubectl config use-context "kind-$CLUSTER_NAME" >/dev/null
kubectl wait --for=condition=Ready nodes --all --timeout=180s
echo "kind cluster '$CLUSTER_NAME' is ready (context kind-$CLUSTER_NAME)"
