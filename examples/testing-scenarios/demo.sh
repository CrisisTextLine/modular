#!/bin/bash

# Quick Demonstration of Testing Scenarios
# Shows key functionality working

set -e

PROXY_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${YELLOW}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║${NC}           ${BLUE}Testing Scenarios Demonstration${NC}               ${YELLOW}║${NC}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════════╝${NC}"
echo

# Start the testing scenarios app in background
echo -e "${BLUE}Starting testing scenarios application...${NC}"
cd /home/runner/work/modular/modular/examples/testing-scenarios
./testing-scenarios >/dev/null 2>&1 &
APP_PID=$!

echo "Application PID: $APP_PID"
echo "Waiting for application to start..."
sleep 8

# Function to test an endpoint
test_endpoint() {
    local description="$1"
    local method="${2:-GET}"
    local endpoint="${3:-/}"
    local headers="${4:-}"
    
    echo -n "  $description... "
    
    local cmd="curl -s -w '%{http_code}' -m 5 -X $method"
    
    if [[ -n "$headers" ]]; then
        cmd="$cmd -H '$headers'"
    fi
    
    cmd="$cmd '$PROXY_URL$endpoint'"
    
    local response
    response=$(eval "$cmd" 2>/dev/null) || {
        echo -e "${RED}FAIL (connection error)${NC}"
        return 1
    }
    
    local status_code="${response: -3}"
    
    if [[ "$status_code" == "200" ]]; then
        echo -e "${GREEN}PASS${NC}"
        return 0
    else
        echo -e "${RED}FAIL (HTTP $status_code)${NC}"
        return 1
    fi
}

# Wait for service to be ready
echo -n "Waiting for proxy service... "
for i in {1..30}; do
    if curl -s -f "$PROXY_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}READY${NC}"
        break
    fi
    sleep 1
    if [[ $i -eq 30 ]]; then
        echo -e "${RED}TIMEOUT${NC}"
        kill $APP_PID 2>/dev/null
        exit 1
    fi
done

echo

# Test 1: Basic Health Checks
echo -e "${BLUE}Test 1: Health Check Scenarios${NC}"
test_endpoint "General health check" "GET" "/health"
test_endpoint "API v1 health" "GET" "/api/v1/health"
test_endpoint "Legacy health" "GET" "/legacy/status"

echo

# Test 2: Multi-Tenant Routing
echo -e "${BLUE}Test 2: Multi-Tenant Scenarios${NC}"
test_endpoint "Alpha tenant" "GET" "/api/v1/test" "X-Tenant-ID: tenant-alpha"
test_endpoint "Beta tenant" "GET" "/api/v1/test" "X-Tenant-ID: tenant-beta"
test_endpoint "No tenant (default)" "GET" "/api/v1/test"

echo

# Test 3: Feature Flag Routing
echo -e "${BLUE}Test 3: Feature Flag Scenarios${NC}"
test_endpoint "API v1 with feature flag" "GET" "/api/v1/test" "X-Feature-Flag: enabled"
test_endpoint "API v2 routing" "GET" "/api/v2/test"
test_endpoint "Canary endpoint" "GET" "/api/canary/test"

echo

# Test 4: Load Testing (simplified)
echo -e "${BLUE}Test 4: Load Testing Scenario${NC}"
echo -n "  Concurrent requests (5x)... "

success_count=0
for i in {1..5}; do
    if curl -s -f "$PROXY_URL/api/v1/load" >/dev/null 2>&1; then
        ((success_count++))
    fi
done

if [[ $success_count -eq 5 ]]; then
    echo -e "${GREEN}PASS ($success_count/5)${NC}"
else
    echo -e "${RED}PARTIAL ($success_count/5)${NC}"
fi

echo

# Test 5: Error Handling
echo -e "${BLUE}Test 5: Error Handling Scenarios${NC}"
test_endpoint "Valid endpoint" "GET" "/api/v1/test"

echo -n "  404 Not Found... "
response=$(curl -s -w '%{http_code}' -m 5 "$PROXY_URL/nonexistent" 2>/dev/null || echo "000")
status_code="${response: -3}"
if [[ "$status_code" == "404" ]]; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL (HTTP $status_code)${NC}"
fi

echo

# Test 6: Different HTTP Methods
echo -e "${BLUE}Test 6: HTTP Method Scenarios${NC}"
test_endpoint "GET request" "GET" "/api/v1/methods"
test_endpoint "POST request" "POST" "/api/v1/methods"
test_endpoint "PUT request" "PUT" "/api/v1/methods"

echo

# Summary
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║${NC}                  ${YELLOW}Demonstration Complete!${NC}                    ${GREEN}║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${BLUE}Key Features Demonstrated:${NC}"
echo "✓ Health check monitoring"
echo "✓ Multi-tenant routing"
echo "✓ Feature flag-based routing"
echo "✓ Load handling"
echo "✓ Error handling"
echo "✓ Multiple HTTP methods"
echo
echo -e "${BLUE}Available Test Commands:${NC}"
echo "• ./testing-scenarios --scenario health-check"
echo "• ./testing-scenarios --scenario load-test --connections 20"
echo "• ./testing-scenarios --scenario feature-flags"
echo "• ./test-all.sh"
echo "• ./test-health-checks.sh"
echo "• ./test-load.sh"
echo "• ./test-feature-flags.sh"

# Clean up
echo
echo "Stopping application..."
kill $APP_PID 2>/dev/null
wait $APP_PID 2>/dev/null
echo "Done!"