#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# delete-experiments.sh
# Elimina experimentos de chaos engineering del cluster Kind.
# Uso:
#   ./scripts/delete-experiments.sh                    # elimina todos los tipos
#   ./scripts/delete-experiments.sh podchaos           # solo PodChaos
#   ./scripts/delete-experiments.sh networkchaos stresschaos
#   ./scripts/delete-experiments.sh --name basic-kill-one        # uno específico
#
# Flags:
#   --name <nombre>   Elimina solo el experimento con ese nombre
#   --all-types       Elimina todos los tipos (comportamiento por defecto sin args)
#   --force           No pide confirmación
# ─────────────────────────────────────────────────────────────────────────────

CONTEXT="${KUBE_CONTEXT:-kind-chaos-testing-v2}"
NAMESPACE="${CHAOS_NAMESPACE:-chaos-demo}"

RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
NC='\033[0m'

ALL_TYPES=(podchaos networkchaos stresschaos httpchaos)

log()  { echo -e "${CYAN}[delete]${NC} $*"; }
ok()   { echo -e "${GREEN}  ✓${NC} $*"; }
warn() { echo -e "${YELLOW}  !${NC} $*"; }
fail() { echo -e "${RED}  ✗${NC} $*"; }

# ── Helpers ───────────────────────────────────────────────────────────────────

count_experiments() {
  local total=0
  for type in "${ALL_TYPES[@]}"; do
    local n
    n=$(kubectl get "$type" -n "$NAMESPACE" --context "$CONTEXT" --no-headers 2>/dev/null | wc -l | tr -d ' ')
    total=$((total + n))
  done
  echo "$total"
}

show_existing() {
  local found=0
  for type in "${ALL_TYPES[@]}"; do
    local items
    items=$(kubectl get "$type" -n "$NAMESPACE" --context "$CONTEXT" --no-headers 2>/dev/null || true)
    if [ -n "$items" ]; then
      found=1
      echo -e "  ${YELLOW}${type}${NC}"
      echo "$items" | awk '{printf "    %-35s %-15s %s\n", $1, $NF, $(NF-1)}'
    fi
  done
  if [ "$found" -eq 0 ]; then
    warn "No hay experimentos activos en '$NAMESPACE'"
  fi
}

confirm() {
  local msg="$1"
  echo ""
  echo -e "${YELLOW}  ¿Confirmar?${NC} $msg [s/N] "
  read -r answer
  case "$answer" in
    [sS]|[yY]) return 0 ;;
    *) echo "  Cancelado."; exit 0 ;;
  esac
}

# ── Eliminar ──────────────────────────────────────────────────────────────────

delete_type() {
  local type="$1"
  local items
  items=$(kubectl get "$type" -n "$NAMESPACE" --context "$CONTEXT" --no-headers 2>/dev/null | awk '{print $1}' || true)

  if [ -z "$items" ]; then
    warn "No hay experimentos '$type' en '$NAMESPACE'"
    return
  fi

  echo ""
  log "── $type ──────────────────────────────"
  while IFS= read -r name; do
    if kubectl delete "$type" "$name" -n "$NAMESPACE" --context "$CONTEXT" &>/dev/null; then
      ok "Eliminado: $name"
    else
      fail "No se pudo eliminar: $name"
    fi
  done <<< "$items"
}

delete_by_name() {
  local name="$1"
  local deleted=0

  echo ""
  log "Buscando experimento con nombre '$name'..."

  for type in "${ALL_TYPES[@]}"; do
    if kubectl get "$type" "$name" -n "$NAMESPACE" --context "$CONTEXT" &>/dev/null; then
      if kubectl delete "$type" "$name" -n "$NAMESPACE" --context "$CONTEXT" &>/dev/null; then
        ok "Eliminado: $type/$name"
        deleted=1
      else
        fail "No se pudo eliminar: $type/$name"
      fi
    fi
  done

  if [ "$deleted" -eq 0 ]; then
    fail "No se encontró ningún experimento con nombre '$name' en '$NAMESPACE'"
    exit 1
  fi
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
  echo ""
  echo -e "${RED}╔══════════════════════════════════════════╗${NC}"
  echo -e "${RED}║   Chaos Engineering — Delete Experiments ║${NC}"
  echo -e "${RED}╚══════════════════════════════════════════╝${NC}"
  echo ""
  log "Contexto : $CONTEXT"
  log "Namespace: $NAMESPACE"

  # Verificar contexto
  if ! kubectl config get-contexts "$CONTEXT" &>/dev/null; then
    fail "Contexto '$CONTEXT' no encontrado."
    exit 1
  fi

  # Parsear argumentos
  local target_name=""
  local force=0
  local types=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --name)
        shift
        target_name="$1"
        ;;
      --force|-f)
        force=1
        ;;
      --all-types)
        types=("${ALL_TYPES[@]}")
        ;;
      -*)
        warn "Flag desconocido: $1"
        ;;
      *)
        local normalized
        normalized=$(echo "$1" | tr '[:upper:]' '[:lower:]')
        case "$normalized" in
          pod*chaos|pod)         types+=(podchaos) ;;
          network*chaos|network) types+=(networkchaos) ;;
          stress*chaos|stress)   types+=(stresschaos) ;;
          http*chaos|http)       types+=(httpchaos) ;;
          *)
            warn "Tipo '$1' no reconocido, ignorando"
            ;;
        esac
        ;;
    esac
    shift
  done

  # Eliminar por nombre específico
  if [ -n "$target_name" ]; then
    echo ""
    log "Experimentos existentes:"
    show_existing
    [ "$force" -eq 0 ] && confirm "Eliminar experimento '$target_name'"
    delete_by_name "$target_name"
    echo ""
    return
  fi

  # Si no se especificaron tipos, usar todos
  if [ ${#types[@]} -eq 0 ]; then
    types=("${ALL_TYPES[@]}")
  fi

  # Mostrar lo que existe
  echo ""
  log "Experimentos existentes en '$NAMESPACE':"
  show_existing

  local total
  total=$(count_experiments)
  if [ "$total" -eq 0 ]; then
    echo ""
    log "Nada que eliminar."
    exit 0
  fi

  # Confirmar
  if [ "$force" -eq 0 ]; then
    local type_list
    type_list=$(IFS=', '; echo "${types[*]}")
    confirm "Se eliminarán todos los experimentos de tipo: ${type_list} en namespace '${NAMESPACE}'"
  fi

  # Ejecutar borrado
  for type in "${types[@]}"; do
    delete_type "$type"
  done

  echo ""
  log "── Estado post-eliminación ──────────────────────"
  local remaining
  remaining=$(count_experiments)
  if [ "$remaining" -eq 0 ]; then
    echo -e "${GREEN}  ✓ Namespace '$NAMESPACE' sin experimentos activos.${NC}"
  else
    warn "$remaining experimento(s) aún presentes:"
    show_existing
  fi
  echo ""
}

main "$@"
