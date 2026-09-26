#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# apply-experiments.sh
# Aplica experimentos de chaos engineering al cluster Kind.
# Uso:
#   ./scripts/apply-experiments.sh                    # aplica todos los tipos
#   ./scripts/apply-experiments.sh podchaos           # solo PodChaos
#   ./scripts/apply-experiments.sh networkchaos stresschaos  # dos tipos
#
# Tipos válidos: podchaos | networkchaos | stresschaos | httpchaos
# ─────────────────────────────────────────────────────────────────────────────

CONTEXT="${KUBE_CONTEXT:-kind-chaos-testing-v2}"
NAMESPACE="${CHAOS_NAMESPACE:-chaos-demo}"
SAMPLES_DIR="$(cd "$(dirname "$0")/../samples" && pwd)"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

ALL_TYPES=(podchaos networkchaos stresschaos httpchaos)

log()  { echo -e "${CYAN}[apply]${NC} $*"; }
ok()   { echo -e "${GREEN}  ✓${NC} $*"; }
warn() { echo -e "${YELLOW}  !${NC} $*"; }
fail() { echo -e "${RED}  ✗${NC} $*"; }

# ── Validaciones previas ──────────────────────────────────────────────────────

check_context() {
  if ! kubectl config get-contexts "$CONTEXT" &>/dev/null; then
    fail "Contexto '$CONTEXT' no encontrado."
    echo "  Contextos disponibles:"
    kubectl config get-contexts --no-headers -o name | sed 's/^/    /'
    exit 1
  fi
}

check_crds() {
  local missing=0
  for crd in podchaos networkchaos stresschaos httpchaos; do
    if ! kubectl get crd "${crd}.chaos.engineering.io" --context "$CONTEXT" &>/dev/null; then
      warn "CRD ${crd}.chaos.engineering.io no está instalado"
      missing=1
    fi
  done
  if [ "$missing" -eq 1 ]; then
    echo ""
    echo "  Instala los CRDs primero:"
    echo "    kubectl apply -f config/crd/ --context $CONTEXT"
    exit 1
  fi
}

check_namespace() {
  if ! kubectl get namespace "$NAMESPACE" --context "$CONTEXT" &>/dev/null; then
    warn "Namespace '$NAMESPACE' no existe — creándolo..."
    kubectl create namespace "$NAMESPACE" --context "$CONTEXT"
    ok "Namespace '$NAMESPACE' creado"
  fi
}

# ── Aplicar experimentos ──────────────────────────────────────────────────────

apply_type() {
  local type="$1"
  local dir="$SAMPLES_DIR/$type"

  if [ ! -d "$dir" ]; then
    warn "Directorio '$dir' no encontrado, saltando $type"
    return
  fi

  local files
  files=$(find "$dir" -maxdepth 1 -name "*.yaml" | sort)

  if [ -z "$files" ]; then
    warn "No hay archivos YAML en $dir"
    return
  fi

  echo ""
  log "── $type ──────────────────────────────"
  while IFS= read -r file; do
    name=$(basename "$file" .yaml)
    if kubectl apply -f "$file" --context "$CONTEXT" &>/dev/null; then
      ok "$name"
    else
      fail "$name  (revisa: kubectl apply -f $file --context $CONTEXT)"
    fi
  done <<< "$files"
}

# ── Estado final ──────────────────────────────────────────────────────────────

show_status() {
  echo ""
  log "── Estado de los experimentos en namespace '$NAMESPACE' ──"
  for type in podchaos networkchaos stresschaos httpchaos; do
    local count
    count=$(kubectl get "$type" -n "$NAMESPACE" --context "$CONTEXT" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -gt 0 ]; then
      echo ""
      echo -e "  ${YELLOW}${type}${NC}"
      kubectl get "$type" -n "$NAMESPACE" --context "$CONTEXT" 2>/dev/null | sed 's/^/    /'
    fi
  done
  echo ""
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║   Chaos Engineering — Apply Experiments  ║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
  echo ""
  log "Contexto : $CONTEXT"
  log "Namespace: $NAMESPACE"

  check_context
  check_crds
  check_namespace

  # Determinar qué tipos aplicar
  local types=()
  if [ $# -eq 0 ]; then
    types=("${ALL_TYPES[@]}")
  else
    for arg in "$@"; do
      # Normalizar: aceptar "PodChaos", "podchaos", "pod"
      local normalized
      normalized=$(echo "$arg" | tr '[:upper:]' '[:lower:]')
      case "$normalized" in
        pod*chaos|pod)         types+=(podchaos) ;;
        network*chaos|network) types+=(networkchaos) ;;
        stress*chaos|stress)   types+=(stresschaos) ;;
        http*chaos|http)       types+=(httpchaos) ;;
        *)
          warn "Tipo '$arg' no reconocido (podchaos|networkchaos|stresschaos|httpchaos)"
          ;;
      esac
    done
  fi

  if [ ${#types[@]} -eq 0 ]; then
    fail "No hay tipos válidos para aplicar."
    exit 1
  fi

  for type in "${types[@]}"; do
    apply_type "$type"
  done

  show_status

  echo -e "${GREEN}✓ Listo.${NC} Para monitorear:"
  echo "  kubectl get podchaos,networkchaos,stresschaos,httpchaos -n $NAMESPACE --context $CONTEXT -w"
  echo ""
}

main "$@"
