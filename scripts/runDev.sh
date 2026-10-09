#!/usr/bin/env bash
# Runs Ships 3D for development: ships-go-3d (backend) and ships-vue-3d
# (frontend) together, with their logs interleaved in this console, and prints
# the URLs to open once both are up. Ctrl+C stops both.
#
# Expects the workspace layout: ships-go-3d/, ships-vue-3d/ and the shared
# files/ (with .env and ssl/) side by side. MongoDB must already be running
# (ships-go/scripts/runMongoContainer.sh starts a local one).
#
# It uses the 2D game's ports (backend PORT from files/.env, default 3000;
# frontend 5173), so the origins ships-go already allows cover it - and the
# 2D game can't be running at the same time.

set -uo pipefail

BACK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(dirname "$BACK_DIR")"
FRONT_DIR="$ROOT_DIR/ships-vue-3d"
FILES_DIR="$ROOT_DIR/files"
FRONT_PORT=5173
STARTUP_TIMEOUT=90

if [[ -t 1 ]]; then
  BOLD=$'\e[1m' DIM=$'\e[2m' RED=$'\e[31m' GREEN=$'\e[32m' YELLOW=$'\e[33m' BLUE=$'\e[34m' MAGENTA=$'\e[35m' RESET=$'\e[0m'
else
  BOLD='' DIM='' RED='' GREEN='' YELLOW='' BLUE='' MAGENTA='' RESET=''
fi

info() { echo "${BOLD}[run]${RESET} $*"; }
warn() { echo "${YELLOW}${BOLD}[run] warning:${RESET} $*"; }
fail() { echo "${RED}${BOLD}[run] error:${RESET} $*" >&2; exit 1; }

# --- Checks --------------------------------------------------------------------

command -v go >/dev/null || fail "go is not installed"
command -v npm >/dev/null || fail "npm is not installed"
command -v curl >/dev/null || fail "curl is not installed"
[[ -d "$FRONT_DIR" ]] || fail "ships-vue-3d not found at $FRONT_DIR"
[[ -f "$FILES_DIR/.env" ]] || fail "$FILES_DIR/.env is missing (it holds MONGODB_URI and the SSL paths)"

# The .env/ssl symlinks are gitignored, so a fresh clone lacks them.
link() {
  local target="$1" name="$2"
  if [[ ! -e "$name" && -e "$(dirname "$name")/$target" ]]; then
    ln -sfn "$target" "$name" && info "linked $name -> $target"
  fi
}
link ../files/.env "$BACK_DIR/.env"
link ../files/ssl "$BACK_DIR/ssl"
link ../files/ssl "$FRONT_DIR/ssl"

# Reads one variable from the shared .env without sourcing the whole file
# (it holds secrets, and isn't guaranteed to be valid bash).
env_value() {
  grep -E "^[[:space:]]*$1=" "$FILES_DIR/.env" | tail -n 1 | cut -d= -f2- | sed -e 's/^["'\'']//' -e 's/["'\'']$//'
}

# Same precedence as ships-go-3d's config: SHIPS3D_PORT, then PORT, then 3000.
BACK_PORT="${SHIPS3D_PORT:-$(env_value SHIPS3D_PORT)}"
BACK_PORT="${BACK_PORT:-${PORT:-$(env_value PORT)}}"
BACK_PORT="${BACK_PORT:-3000}"

if [[ -f "$FRONT_DIR/ssl/cert.pem" && -f "$FRONT_DIR/ssl/key.pem" ]]; then
  PROTOCOL=https
else
  PROTOCOL=http
  warn "no certificate in files/ssl: running over plain HTTP, where the Secure session cookie won't work (login will fail)"
fi

# The backend accepts ALLOWED_ORIGINS plus SHIPS3D_ALLOWED_ORIGINS.
ORIGINS="$(env_value ALLOWED_ORIGINS)|$(env_value SHIPS3D_ALLOWED_ORIGINS)"
if [[ "|$ORIGINS|" != *"|$PROTOCOL://localhost:$FRONT_PORT|"* ]]; then
  warn "ALLOWED_ORIGINS in files/.env doesn't include $PROTOCOL://localhost:$FRONT_PORT;"
  warn "the browser will be refused by the backend."
fi

port_busy() { ss -ltnH "sport = :$1" 2>/dev/null | grep -q .; }
port_busy "$BACK_PORT" && fail "port $BACK_PORT is already in use (the 2D ships-go, or another ships-go-3d?)"
port_busy "$FRONT_PORT" && fail "port $FRONT_PORT is already in use (the 2D ships-vue, or another ships-vue-3d?)"

if [[ ! -d "$FRONT_DIR/node_modules" ]]; then
  info "installing frontend dependencies…"
  (cd "$FRONT_DIR" && npm install) || fail "npm install failed"
fi

# --- Start ---------------------------------------------------------------------

BACK_PID=''
FRONT_PID=''

# Each service runs in its own process group (setsid), and is stopped by
# killing the whole group: `go run` and `npm run` start the real server as a
# child, so killing only the pid we know would leave it running.
stop() {
  trap - INT TERM EXIT
  echo
  info "stopping…"
  for pid in "$BACK_PID" "$FRONT_PID"; do
    [[ -n "$pid" ]] && kill -TERM -- "-$pid" 2>/dev/null
  done
  wait 2>/dev/null
  info "stopped"
}
trap 'stop; exit 0' INT TERM
trap stop EXIT

# Prefixes every line of a service's output with its name.
prefix() { sed -u "s/^/$1 /"; }

info "starting backend (ships-go-3d) on port $BACK_PORT…"
(cd "$BACK_DIR" && SHIPS3D_PORT="$BACK_PORT" exec setsid go run .) \
  > >(prefix "${BLUE}[back]${RESET} ") 2>&1 &
BACK_PID=$!

info "starting frontend (ships-vue-3d) on port $FRONT_PORT…"
(cd "$FRONT_DIR" && VITE_API_PORT="$BACK_PORT" exec setsid npm run dev -- --port "$FRONT_PORT" --strictPort) \
  > >(prefix "${MAGENTA}[front]${RESET}") 2>&1 &
FRONT_PID=$!

# Waits until a URL answers, or the service dies, or time runs out.
wait_for() {
  local name="$1" url="$2" pid="$3"
  for ((i = 0; i < STARTUP_TIMEOUT; i++)); do
    curl -sk -o /dev/null --max-time 2 "$url" && return 0
    kill -0 "$pid" 2>/dev/null || fail "$name exited during startup (see its log above)"
    sleep 1
  done
  fail "$name didn't answer on $url within ${STARTUP_TIMEOUT}s"
}

wait_for "backend" "$PROTOCOL://localhost:$BACK_PORT/status" "$BACK_PID"
wait_for "frontend" "$PROTOCOL://localhost:$FRONT_PORT/" "$FRONT_PID"

LAN_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
WS_PROTOCOL=ws
[[ "$PROTOCOL" == https ]] && WS_PROTOCOL=wss

cat <<EOF

${GREEN}${BOLD}  Ships 3D is running${RESET}

  ${BOLD}Open the game:${RESET}  ${GREEN}$PROTOCOL://localhost:$FRONT_PORT${RESET}
EOF
[[ -n "$LAN_IP" ]] && echo "  On your network: $PROTOCOL://$LAN_IP:$FRONT_PORT"
cat <<EOF

  Backend API:     $PROTOCOL://localhost:$BACK_PORT
  Game websocket:  $WS_PROTOCOL://localhost:$BACK_PORT/ws
EOF
[[ "$PROTOCOL" == https ]] && echo "  ${DIM}Self-signed certificate: open the backend URL once and accept it, or the API calls fail.${RESET}"
cat <<EOF

  ${DIM}Ctrl+C stops both.${RESET}

EOF

# Run until either service stops; then stop the other too.
wait -n "$BACK_PID" "$FRONT_PID"
warn "a service exited; stopping the other one"
exit 1
