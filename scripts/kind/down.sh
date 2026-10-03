#!/usr/bin/env bash
set -Eeuo pipefail

CLUSTER_NAME="${KIND_CLUSTER:-chaos-testing-v2}"
command -v kind >/dev/null || { echo "Missing required command: kind" >&2; exit 1; }
kind delete cluster --name "$CLUSTER_NAME"
