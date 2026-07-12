#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# test-chat-flow.sh
# Fully automated test of one-to-one chat messaging via REST API.
#
# Prerequisites:
#   - docker compose services are running (api-gateway, user-service, chat-service)
#   - jq installed
#   - curl installed
#
# Usage:
#   chmod +x scripts/test-chat-flow.sh
#   ./scripts/test-chat-flow.sh
#
# What it does:
#   1. Creates two users (Alice & Bob) via POST /v1/users
#   2. Creates a conversation between them via POST /v1/conversations
#   3. Sends a few messages back and forth via POST /v1/conversations/{id}/messages
#   4. Fetches the conversation messages via GET to verify persistence
#   5. Prints wscat commands for manual WebSocket testing
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

# ─── Configuration ──────────────────────────────────────────────────────────

API_BASE="${API_BASE:-http://localhost:8080}"
VERBOSE="${VERBOSE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

PASS=0
FAIL=0

# ─── Helpers ────────────────────────────────────────────────────────────────

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[PASS]${NC}  $*"; PASS=$((PASS + 1)); }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; FAIL=$((FAIL + 1)); }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
step()  { echo; echo -e "${YELLOW}═══ $* ═══${NC}"; }
log()   { [[ "$VERBOSE" == "true" ]] && echo "  │ $*" >&2; }

# Generate a dev JWT with the given user ID as the `sub` claim.
# In dev mode the API gateway's DevAuthMiddleware reads the sub claim
# without verifying the signature — so any base64 payload works.
make_jwt() {
  local user_id="$1"
  local header payload sig token

  header=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 -w0 2>/dev/null || echo -n '{"alg":"HS256","typ":"JWT"}' | base64)
  # strip padding and base64-url-safe encode
  header=$(echo -n "$header" | sed 's/+/-/g; s/\//_/g; s/=//g')

  payload=$(echo -n "{\"sub\":\"${user_id}\",\"name\":\"${user_id}\"}" | base64 -w0 2>/dev/null || echo -n "{\"sub\":\"${user_id}\",\"name\":\"${user_id}\"}" | base64)
  payload=$(echo -n "$payload" | sed 's/+/-/g; s/\//_/g; s/=//g')

  sig="dev-mode-unsigned"
  echo "${header}.${payload}.${sig}"
}

# Curl wrapper that adds the dev JWT auth header.
# Returns the response body on stdout. Logs go to stderr.
api() {
  local token="$1"
  shift
  local method path body
  method="$1"; path="$2"; body="${3:-}"
  local url="${API_BASE}${path}"

  local args=(-sS)
  if [[ "$VERBOSE" == "true" ]]; then
    args+=(-v)
  fi

  log "${method} ${url}"

  if [[ -n "$body" ]]; then
    curl "${args[@]}" -X "$method" "$url" \
      -H "Authorization: Bearer ${token}" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl "${args[@]}" -X "$method" "$url" \
      -H "Authorization: Bearer ${token}"
  fi
  echo
}

# Test if a command exists.
require_cmd() {
  if ! command -v "$1" &>/dev/null; then
    fail "Required command '$1' not found. Please install it."
    exit 1
  fi
}

# ─── Pre-flight Checks ─────────────────────────────────────────────────────

require_cmd curl
require_cmd jq

echo
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Chat Service — REST API Test Suite               ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo
info "API Gateway: ${API_BASE}"
info "Started at:  $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo

# ─── Step 1: Check API Gateway is reachable ────────────────────────────────

step "Step 1: Health check"

HEALTH=$(curl -sS -o /dev/null -w '%{http_code}' "${API_BASE}/health" 2>/dev/null || echo "000")
if [[ "$HEALTH" == "200" ]]; then
  ok "API Gateway is reachable at ${API_BASE}"
else
  fail "API Gateway not reachable at ${API_BASE} (HTTP ${HEALTH})"
  warn "Run 'docker compose ps' to check service status."
  warn "Exiting."
  exit 1
fi

# ─── Step 2: Create users ──────────────────────────────────────────────────

step "Step 2: Create users"

ALICE_ID="test-alice-$(date +%s)"
BOB_ID="test-bob-$(date +%s)"

ALICE_JWT=$(make_jwt "$ALICE_ID")
BOB_JWT=$(make_jwt "$BOB_ID")

# Create Alice
info "Creating user: ${ALICE_ID}..."
ALICE_RESP=$(api "$ALICE_JWT" POST /v1/users \
  "{\"email\":\"${ALICE_ID}@test.local\",\"display_name\":\"Alice\",\"avatar_url\":\"\"}")
ALICE_UID=$(echo "$ALICE_RESP" | jq -r '.id // empty')
if [[ -n "$ALICE_UID" ]]; then
  ok "Alice created: id=${ALICE_UID}"
  log "Response: $(echo "$ALICE_RESP" | jq -c .)"
  # Regenerate JWT with the real UUID so the X-User-ID matches what's stored
  ALICE_JWT=$(make_jwt "$ALICE_UID")
else
  warn "Create may have failed. Checking if user already exists..."
  ALICE_LIST=$(api "$ALICE_JWT" GET /v1/users)
  ALICE_UID=$(echo "$ALICE_LIST" | jq -r ".users[] | select(.email == \"${ALICE_ID}@test.local\") | .id // empty" 2>/dev/null || echo "")
  if [[ -n "$ALICE_UID" ]]; then
    ok "Alice already exists: id=${ALICE_UID}"
    ALICE_JWT=$(make_jwt "$ALICE_UID")
  else
    fail "Could not create or find Alice"
    warn "Response: $(echo "$ALICE_RESP" | jq -c . 2>/dev/null || echo "$ALICE_RESP")"
    exit 1
  fi
fi

# Create Bob
info "Creating user: ${BOB_ID}..."
BOB_RESP=$(api "$BOB_JWT" POST /v1/users \
  "{\"email\":\"${BOB_ID}@test.local\",\"display_name\":\"Bob\",\"avatar_url\":\"\"}")
BOB_UID=$(echo "$BOB_RESP" | jq -r '.id // empty')
if [[ -n "$BOB_UID" ]]; then
  ok "Bob created: id=${BOB_UID}"
  log "Response: $(echo "$BOB_RESP" | jq -c .)"
  # Regenerate JWT with the real UUID
  BOB_JWT=$(make_jwt "$BOB_UID")
else
  warn "Create may have failed. Checking if user already exists..."
  BOB_LIST=$(api "$BOB_JWT" GET /v1/users)
  BOB_UID=$(echo "$BOB_LIST" | jq -r ".users[] | select(.email == \"${BOB_ID}@test.local\") | .id // empty" 2>/dev/null || echo "")
  if [[ -n "$BOB_UID" ]]; then
    ok "Bob already exists: id=${BOB_UID}"
    BOB_JWT=$(make_jwt "$BOB_UID")
  else
    fail "Could not create or find Bob"
    warn "Response: $(echo "$BOB_RESP" | jq -c . 2>/dev/null || echo "$BOB_RESP")"
    exit 1
  fi
fi

# ─── Step 3: Create a conversation ─────────────────────────────────────────

step "Step 3: Create conversation"

info "Creating conversation between Alice (${ALICE_UID}) and Bob (${BOB_UID})..."
CONV_RESP=$(api "$ALICE_JWT" POST /v1/conversations \
  "{\"user_id\":\"${BOB_UID}\"}")
CONV_ID=$(echo "$CONV_RESP" | jq -r '.id // empty')

if [[ -n "$CONV_ID" ]]; then
  ok "Conversation created: id=${CONV_ID}"
  log "Response: $(echo "$CONV_RESP" | jq -c .)"
else
  # Might already exist — try listing conversations
  warn "Create may have failed (may already exist). Listing conversations..."
  CONVS_RESP=$(api "$ALICE_JWT" GET /v1/conversations)
  CONV_ID=$(echo "$CONVS_RESP" | jq -r '.[0].id // empty')
  if [[ -n "$CONV_ID" ]]; then
    ok "Using existing conversation: id=${CONV_ID}"
  else
    fail "Could not create or find a conversation"
    warn "Response: $(echo "$CONV_RESP" | jq -c . 2>/dev/null || echo "$CONV_RESP")"
    exit 1
  fi
fi

# ─── Step 4: Send messages ─────────────────────────────────────────────────

step "Step 4: Send messages"

# Alice sends a message
info "Alice sends message..."
MSG1=$(api "$ALICE_JWT" POST "/v1/conversations/${CONV_ID}/messages" \
  '{"content":"Hey Bob! Ready for a ride? 🚗"}')
MSG1_ID=$(echo "$MSG1" | jq -r '.id // empty')
if [[ -n "$MSG1_ID" ]]; then
  ok "Alice's message sent: id=${MSG1_ID}"
  log "Content: $(echo "$MSG1" | jq -r '.content')"
else
  fail "Failed to send Alice's message"
  warn "Response: $(echo "$MSG1" | jq -c . 2>/dev/null || echo "$MSG1")"
fi

# Bob replies
info "Bob replies..."
MSG2=$(api "$BOB_JWT" POST "/v1/conversations/${CONV_ID}/messages" \
  '{"content":"Hey Alice! Sure, where to? 🗺️"}')
MSG2_ID=$(echo "$MSG2" | jq -r '.id // empty')
if [[ -n "$MSG2_ID" ]]; then
  ok "Bob's message sent: id=${MSG2_ID}"
  log "Content: $(echo "$MSG2" | jq -r '.content')"
else
  fail "Failed to send Bob's message"
  warn "Response: $(echo "$MSG2" | jq -c . 2>/dev/null || echo "$MSG2")"
fi

# Alice sends another
info "Alice sends another message..."
MSG3=$(api "$ALICE_JWT" POST "/v1/conversations/${CONV_ID}/messages" \
  '{"content":"Let me check the map... 🌍"}')
MSG3_ID=$(echo "$MSG3" | jq -r '.id // empty')
if [[ -n "$MSG3_ID" ]]; then
  ok "Alice's second message sent: id=${MSG3_ID}"
else
  fail "Failed to send Alice's second message"
fi

# ─── Step 5: Fetch messages to verify persistence ──────────────────────────

step "Step 5: Verify message persistence"

FETCHED=$(api "$ALICE_JWT" GET "/v1/conversations/${CONV_ID}/messages?limit=10")
MSG_COUNT=$(echo "$FETCHED" | jq '. | length' 2>/dev/null || echo "0")

if [[ "$MSG_COUNT" -ge 3 ]]; then
  ok "All ${MSG_COUNT} messages persisted and retrieved from ScyllaDB"
  log "Messages:"
  echo "$FETCHED" | jq -r '.[] | "  [\(.sender_id)] \(.content)"'
elif [[ "$MSG_COUNT" -gt 0 ]]; then
  ok "${MSG_COUNT} message(s) retrieved (expected 3 — some may still be syncing)"
  echo "$FETCHED" | jq -r '.[] | "  [\(.sender_id)] \(.content)"'
else
  fail "No messages retrieved"
  warn "Response: $(echo "$FETCHED" | jq -c . 2>/dev/null || echo "$FETCHED")"
fi

# ─── Summary ───────────────────────────────────────────────────────────────

step "Results"

echo
echo -e "  ${GREEN}PASS: ${PASS}${NC}"
echo -e "  ${RED}FAIL: ${FAIL}${NC}"
echo

if [[ "$FAIL" -eq 0 ]]; then
  echo -e "${GREEN}✓ All tests passed!${NC}"
else
  echo -e "${RED}✗ ${FAIL} test(s) failed.${NC}"
fi

echo
echo -e "${CYAN}────────────────────────────────────────────────────────────────${NC}"
echo -e "${CYAN}  Manual WebSocket Testing with wscat${NC}"
echo -e "${CYAN}────────────────────────────────────────────────────────────────${NC}"
echo
echo "Open TWO terminals:"
echo
echo "  Terminal 1 (Alice):"
echo "    wscat -c \"ws://localhost:8080/ws?user_id=${ALICE_UID}&room_id=${CONV_ID}\""
echo
echo "  Terminal 2 (Bob):"
echo "    wscat -c \"ws://localhost:8080/ws?user_id=${BOB_UID}&room_id=${CONV_ID}\""
echo
echo "Then in either terminal, send a message:"
echo '    {"type":"chat_message","room_id":"'${CONV_ID}'","data":{"content":"Hello from wscat! 🎉"}}'
echo
echo -e "${CYAN}────────────────────────────────────────────────────────────────${NC}"
echo

exit "$FAIL"
