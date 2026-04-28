#!/bin/bash
set -e

BASE_URL="http://localhost:6969"
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color
BOLD='\033[1m'

print_header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${YELLOW}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

pass() {
    echo -e "  ${GREEN}✓ PASS${NC}: $1"
}

fail() {
    echo -e "  ${RED}✗ FAIL${NC}: $1"
}

# ─────────────────────────────────────────────────────
print_header "1. Health Checks"
# ─────────────────────────────────────────────────────

echo -n "  Checking /healthz... "
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/healthz)
if [ "$HEALTH" == "200" ]; then
    pass "/healthz returned 200"
else
    fail "/healthz returned $HEALTH"
fi

echo -n "  Checking /readyz... "
READY=$(curl -s $BASE_URL/readyz)
echo -e "  Response: ${CYAN}$READY${NC}"
READY_STATUS=$(echo $READY | grep -o '"status":"ready"' || true)
if [ -n "$READY_STATUS" ]; then
    pass "/readyz — DB and Redis are UP"
else
    fail "/readyz — system not ready"
fi

# ─────────────────────────────────────────────────────
print_header "2. User Registration"
# ─────────────────────────────────────────────────────

TIMESTAMP=$(date +%s)
USERNAME="testuser_${TIMESTAMP}"

REGISTER_RESP=$(curl -s -X POST $BASE_URL/register \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"$USERNAME\", \"password\": \"password123\"}")

echo -e "  Response: ${CYAN}$REGISTER_RESP${NC}"
USER_ID=$(echo $REGISTER_RESP | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
if [ -n "$USER_ID" ]; then
    pass "User registered with ID: $USER_ID"
else
    fail "Registration failed"
fi

# ─────────────────────────────────────────────────────
print_header "3. User Login (JWT)"
# ─────────────────────────────────────────────────────

LOGIN_RESP=$(curl -s -X POST $BASE_URL/login \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"$USERNAME\", \"password\": \"password123\"}")

echo -e "  Response: ${CYAN}$LOGIN_RESP${NC}"
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"//')
if [ -n "$TOKEN" ]; then
    pass "JWT token received"
else
    fail "Login failed — no token"
    exit 1
fi

# ─────────────────────────────────────────────────────
print_header "4. Submit HIGH Priority Job"
# ─────────────────────────────────────────────────────

HIGH_JOB=$(curl -s -X POST $BASE_URL/jobs \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"type": "email_send", "payload": {"to": "user@example.com", "subject": "Welcome!"}, "priority": "high", "idempotency_key": "email-high-'$TIMESTAMP'"}')

echo -e "  Response: ${CYAN}$HIGH_JOB${NC}"
HIGH_JOB_ID=$(echo $HIGH_JOB | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
if [ -n "$HIGH_JOB_ID" ]; then
    pass "High-priority job created with ID: $HIGH_JOB_ID"
else
    fail "Failed to create high-priority job"
fi

# ─────────────────────────────────────────────────────
print_header "5. Submit DEFAULT Priority Job"
# ─────────────────────────────────────────────────────

DEFAULT_JOB=$(curl -s -X POST $BASE_URL/jobs \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"type": "report_generation", "payload": {"report_id": 42}, "idempotency_key": "report-default-'$TIMESTAMP'"}')

echo -e "  Response: ${CYAN}$DEFAULT_JOB${NC}"
DEFAULT_JOB_ID=$(echo $DEFAULT_JOB | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
if [ -n "$DEFAULT_JOB_ID" ]; then
    pass "Default-priority job created with ID: $DEFAULT_JOB_ID"
else
    fail "Failed to create default-priority job"
fi

# ─────────────────────────────────────────────────────
print_header "6. Submit LOW Priority Job"
# ─────────────────────────────────────────────────────

LOW_JOB=$(curl -s -X POST $BASE_URL/jobs \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"type": "cleanup", "payload": {"target": "old_files"}, "priority": "low", "idempotency_key": "cleanup-low-'$TIMESTAMP'"}')

echo -e "  Response: ${CYAN}$LOW_JOB${NC}"
LOW_JOB_ID=$(echo $LOW_JOB | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
if [ -n "$LOW_JOB_ID" ]; then
    pass "Low-priority job created with ID: $LOW_JOB_ID"
else
    fail "Failed to create low-priority job"
fi

# ─────────────────────────────────────────────────────
print_header "7. Idempotency Check (duplicate submission)"
# ─────────────────────────────────────────────────────

DUPE_JOB=$(curl -s -X POST $BASE_URL/jobs \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"type": "email_send", "payload": {"to": "user@example.com", "subject": "Welcome!"}, "priority": "high", "idempotency_key": "email-high-'$TIMESTAMP'"}')

echo -e "  Response: ${CYAN}$DUPE_JOB${NC}"
DUPE_MSG=$(echo $DUPE_JOB | grep -o '"duplicate job' || true)
if [ -n "$DUPE_MSG" ]; then
    pass "Idempotency working — duplicate detected!"
else
    fail "Idempotency check may not have triggered"
fi

# ─────────────────────────────────────────────────────
print_header "8. Wait for Worker to Process Jobs (5s)"
# ─────────────────────────────────────────────────────

echo "  Waiting 5 seconds for the worker to pick up jobs..."
sleep 5

# ─────────────────────────────────────────────────────
print_header "9. Check Job Statuses"
# ─────────────────────────────────────────────────────

for JOB_ID in $HIGH_JOB_ID $DEFAULT_JOB_ID $LOW_JOB_ID; do
    STATUS_RESP=$(curl -s $BASE_URL/jobs/$JOB_ID \
        -H "Authorization: Bearer $TOKEN")
    echo -e "  Job $JOB_ID: ${CYAN}$STATUS_RESP${NC}"
    STATUS=$(echo $STATUS_RESP | grep -o '"status":"[^"]*"' | sed 's/"status":"//;s/"//')
    if [ "$STATUS" == "completed" ]; then
        pass "Job $JOB_ID — completed"
    elif [ "$STATUS" == "running" ]; then
        echo -e "  ${YELLOW}⟳ IN PROGRESS${NC}: Job $JOB_ID still running"
    elif [ "$STATUS" == "pending" ]; then
        echo -e "  ${YELLOW}⏳ PENDING${NC}: Job $JOB_ID waiting in queue"
    else
        echo -e "  ${RED}Status: $STATUS${NC}"
    fi
done

# ─────────────────────────────────────────────────────
print_header "10. DLQ Replay Test"
# ─────────────────────────────────────────────────────

# Try replaying a completed job (should fail with "only dead jobs can be replayed")
REPLAY_RESP=$(curl -s -X POST $BASE_URL/admin/jobs/$HIGH_JOB_ID/replay \
    -H "Authorization: Bearer $TOKEN")
echo -e "  Replay completed job: ${CYAN}$REPLAY_RESP${NC}"
REPLAY_ERR=$(echo $REPLAY_RESP | grep -o '"only dead jobs' || true)
if [ -n "$REPLAY_ERR" ]; then
    pass "Replay correctly rejected non-dead job"
else
    echo -e "  ${YELLOW}ℹ NOTE${NC}: Job may be in dead state or endpoint behaved differently"
fi

# ─────────────────────────────────────────────────────
print_header "11. Prometheus Metrics"
# ─────────────────────────────────────────────────────

METRICS=$(curl -s $BASE_URL/metrics | grep -E "^jobs_|^worker_|^job_duration" | head -15)
if [ -n "$METRICS" ]; then
    pass "Prometheus metrics are being exposed:"
    echo -e "${CYAN}"
    echo "$METRICS"
    echo -e "${NC}"
else
    fail "No custom metrics found at /metrics"
fi

# ─────────────────────────────────────────────────────
print_header "✅ Test Summary"
# ─────────────────────────────────────────────────────

echo -e "  ${GREEN}All integration checks completed!${NC}"
echo -e "  API is live at: ${BOLD}$BASE_URL${NC}"
echo -e "  Metrics at:     ${BOLD}$BASE_URL/metrics${NC}"
echo ""
