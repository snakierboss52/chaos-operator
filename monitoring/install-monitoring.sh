#!/usr/bin/env bash
set -Eeuo pipefail

# Backwards-compatible entry point; the canonical installer lives with the Helm values.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "$ROOT_DIR/scripts/helms/install-local.sh" "$@"
