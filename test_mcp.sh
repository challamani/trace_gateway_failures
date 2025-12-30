#!/bin/bash

# Test script for MCP server
# This script sends various MCP requests to test the server functionality

echo "=== Testing MCP Server ==="
echo

# Test 1: Initialize
echo "Test 1: Initialize Request"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-server &
SERVER_PID=$!
sleep 2

# Kill the server process
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
echo

# Test 2: List Tools
echo "Test 2: List Tools Request"
(
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  sleep 0.5
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  sleep 0.5
) | timeout 5 ./mcp-server 2>/dev/null
echo

# Test 3: Call Tool (Example - will fail if no cluster access)
echo "Test 3: Get Pod Logs Request (Example)"
(
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  sleep 0.5
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_pod_logs","arguments":{"namespaces":["default"],"tail_lines":50}}}'
  sleep 2
) | timeout 10 ./mcp-server 2>&1 | head -50
echo

echo "=== Test Complete ==="
echo "Note: The get_pod_logs test requires a valid Kubernetes cluster connection."
echo "If you see connection errors, ensure your kubeconfig is properly configured."
