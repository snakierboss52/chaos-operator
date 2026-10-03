#!/usr/bin/env bash
set -Eeuo pipefail

CLUSTER_NAME="${KIND_CLUSTER:-chaos-testing-v2}"
command -v kind >/dev/null || { echo "Missing required command: kind" >&2; exit 1; }
if ! kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
  echo "kind cluster '$CLUSTER_NAME' does not exist"
  exit 1
fi
kubectl --context "kind-$CLUSTER_NAME" get nodes -o wide
