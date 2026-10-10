#!/usr/bin/env bash
# Deploys Ships 3D to the production server (jonbulserver) from this machine,
# in one command: it runs the server's own deploy scripts over SSH, checks
# the result and prints what is live.
#
#   scripts/deployProd.sh [all|back|front] [release|snapshot] [-y]
#
#   all (default)  backend then frontend      release (default)  latest GitHub release
#   back           ships-go-3d only           snapshot           latest-snapshot (open PRs)
#   front          ships-vue-3d only          -y                 don't ask, even with players online
#
# Needs the SSH key `ships-server` in the workspace root (or SHIPS_SERVER_KEY)
# and, on the server, ~/servers/ships/runShipsGo3d.sh and runShipsVue3d.sh
# (see the workspace CLAUDE.md, "Production server"). Restarting the backend
# drops whoever is playing, so it asks first when someone is.

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVER="${SHIPS_SERVER:-jonbul@192.168.1.10}"
KEY="${SHIPS_SERVER_KEY:-$ROOT_DIR/ships-server}"
REMOTE_DIR='~/servers/ships'
PUBLIC_HOST="${SHIPS_PUBLIC_HOST:-jonbul.ddns.net}"

if [[ -t 1 ]]; then
  BOLD=$'\e[1m' DIM=$'\e[2m' RED=$'\e[31m' GREEN=$'\e[32m' YELLOW=$'\e[33m' RESET=$'\e[0m'
else
  BOLD='' DIM='' RED='' GREEN='' YELLOW='' RESET=''
fi
info() { echo "${BOLD}[deploy]${RESET} $*"; }
warn() { echo "${YELLOW}${BOLD}[deploy] warning:${RESET} $*"; }
fail() { echo "${RED}${BOLD}[deploy] error:${RESET} $*" >&2; exit 1; }

usage() { sed -n '2,17p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit "${1:-0}"; }

# --- Arguments -----------------------------------------------------------------

TARGET=all
CHANNEL=release
ASSUME_YES=false
for arg in "$@"; do
  case "$arg" in
    all | back | front) TARGET=$arg ;;
    release | snapshot) CHANNEL=$arg ;;
    -y | --yes) ASSUME_YES=true ;;
    -h | --help) usage 0 ;;
    *) echo "unknown argument: $arg" >&2; usage 1 ;;
  esac
done

# --- Checks --------------------------------------------------------------------

[[ -f "$KEY" ]] || fail "SSH key not found: $KEY (set SHIPS_SERVER_KEY)"
if [[ "$(stat -c '%a' "$KEY")" != 600 ]]; then
  fail "SSH refuses keys others can read: run  chmod 600 $KEY"
fi
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 "$SERVER")
remote() { "${SSH[@]}" "$@"; }

remote true 2>/dev/null || fail "cannot SSH to $SERVER with $KEY"

# --- What is about to go live ----------------------------------------------------

tag=latest-snapshot
[[ "$CHANNEL" == release ]] && tag=''
repos=()
[[ "$TARGET" == all || "$TARGET" == back ]] && repos+=(ships-go-3d)
[[ "$TARGET" == all || "$TARGET" == front ]] && repos+=(ships-vue-3d)

if command -v gh >/dev/null; then
  for repo in "${repos[@]}"; do
    # Without a tag, gh shows the latest release.
    line=$(gh release view $tag -R "jonbul/$repo" \
      --json tagName,publishedAt,assets \
      -q '"\(.tagName), built \([.assets[].updatedAt] | max)"' 2>/dev/null) ||
      fail "no $CHANNEL found for jonbul/$repo"
    info "$repo: $line"
  done
else
  info "(install gh to see which build each channel points at)"
fi

# Who's playing: asked on the server itself, which reaches its own port
# reliably (from the LAN, the public name may not route back to it).
status=$(remote "curl -sk --max-time 5 https://localhost:3000/status" 2>/dev/null)
players=$(sed -n 's/.*"players":\([0-9]*\).*/\1/p' <<<"$status")
if [[ "$TARGET" != front && "${players:-0}" -gt 0 && "$ASSUME_YES" != true ]]; then
  warn "$players player(s) online: restarting the backend disconnects them."
  read -r -p "Deploy anyway? [y/N] " reply
  [[ "$reply" == [yY]* ]] || { info "cancelled"; exit 0; }
fi

# --- Deploy --------------------------------------------------------------------

deploy() {
  local script=$1 name=$2
  info "deploying $name ($CHANNEL)…"
  # The server scripts are verbose (set -x); show their outcome, and the full
  # log only when they fail.
  local log
  log=$(remote "cd $REMOTE_DIR && ./$script $CHANNEL" 2>&1)
  local code=$?
  if [[ $code -ne 0 ]]; then
    echo "$log" | tail -40
    fail "$name failed (exit $code); production may be half-deployed: see the log above"
  fi
  grep -E '^OK' <<<"$log" | sed "s/^/${GREEN}  /;s/$/${RESET}/"
}

# Backend first: if it fails, the frontend isn't touched.
[[ "$TARGET" == all || "$TARGET" == back ]] && deploy runShipsGo3d.sh ships-go-3d
[[ "$TARGET" == all || "$TARGET" == front ]] && deploy runShipsVue3d.sh ships-vue-3d

# --- Check from outside -----------------------------------------------------------

check() {
  local label=$1 url=$2
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 8 "$url")
  if [[ "$code" == 200 ]]; then
    echo "  ${GREEN}✓${RESET} $label  $url"
  else
    echo "  ${YELLOW}?${RESET} $label  $url  (HTTP $code from here; the server itself reported OK)"
  fi
}
info "checking from this machine (strict TLS):"
check "site   " "https://$PUBLIC_HOST:5173/"
check "backend" "https://$PUBLIC_HOST:3000/status"

echo
remote "docker ps --filter name=ships --format '  {{.Names}}\t{{.Status}}'" 2>/dev/null
echo
info "${GREEN}done.${RESET} ${DIM}Roll back to 2D: ssh in and run  docker stop ships-go-3d ships-vue-3d && docker start ships-go ships-vue ships-npc${RESET}"
