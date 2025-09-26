#!/bin/bash

# Smoke test script for umcp
set -e

BINARY="./umcp"
CONFIGS_DIR="configs"
TEST_DIR="test"
TEMP_DIR=$(mktemp -d)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${YELLOW}→${NC} $1"
}

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Test 1: Binary exists and is executable
info "Test 1: Checking binary exists"
if [ -x "$BINARY" ]; then
    pass "Binary exists and is executable"
else
    fail "Binary not found or not executable"
    exit 1
fi

# Test 2: Show version
info "Test 2: Checking version flag"
if $BINARY --version 2>&1 | grep -q "umcp version"; then
    pass "Version flag works"
else
    fail "Version flag failed"
fi

# Test 3: Validate configurations
info "Test 3: Validating configuration files"
for config in $CONFIGS_DIR/*.yaml; do
    if $BINARY --config "$config" --validate > /dev/null 2>&1; then
        pass "Config valid: $(basename $config)"
    else
        fail "Config invalid: $(basename $config)"
    fi
done

# Test 4: Test mode execution
info "Test 4: Testing test mode"
if $BINARY --config $CONFIGS_DIR/ls.yaml --test > /dev/null 2>&1; then
    pass "Test mode works"
else
    fail "Test mode failed"
fi

# Test 5: Generate Claude config
info "Test 5: Testing Claude config generation"
CLAUDE_CONFIG="$TEMP_DIR/claude_config.json"
if $BINARY --config $CONFIGS_DIR/git.yaml --generate-claude-config > "$CLAUDE_CONFIG" 2>&1; then
    if [ -s "$CLAUDE_CONFIG" ] && grep -q "mcpServers" "$CLAUDE_CONFIG"; then
        pass "Claude config generation works"
    else
        fail "Claude config generated but invalid"
    fi
else
    fail "Claude config generation failed"
fi

# Test 6: Multiple configs
info "Test 6: Testing multiple configs"
if $BINARY --config $CONFIGS_DIR/git.yaml --config $CONFIGS_DIR/ls.yaml --validate > /dev/null 2>&1; then
    pass "Multiple configs work"
else
    fail "Multiple configs failed"
fi

# Test 7: Invalid config handling
info "Test 7: Testing invalid config handling"
INVALID_CONFIG="$TEMP_DIR/invalid.yaml"
echo "invalid: yaml: content" > "$INVALID_CONFIG"
if ! $BINARY --config "$INVALID_CONFIG" --validate > /dev/null 2>&1; then
    pass "Invalid config rejected correctly"
else
    fail "Invalid config not rejected"
fi

# Test 8: Missing config file
info "Test 8: Testing missing config file"
if ! $BINARY --config "/nonexistent/config.yaml" --validate > /dev/null 2>&1; then
    pass "Missing config handled correctly"
else
    fail "Missing config not handled"
fi

# Test 9: MCP protocol test (basic)
info "Test 9: Testing MCP protocol initialization"
MCP_TEST_SCRIPT="$TEST_DIR/mcp_test.py"
if [ -f "$MCP_TEST_SCRIPT" ]; then
    if python3 "$MCP_TEST_SCRIPT" 2>&1; then
        pass "MCP protocol test passed"
    else
        fail "MCP protocol test failed"
    fi
else
    info "Skipping MCP protocol test (test script not found)"
fi

# Summary
echo ""
echo "========================================="
echo "Smoke Test Summary"
echo "========================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All smoke tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi