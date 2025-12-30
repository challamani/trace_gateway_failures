# Example MCP Requests

This file contains example requests that can be sent to the MCP server.

## Initialize Connection

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

## List Available Tools

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

## Get Pod Logs - Basic

Retrieve logs from all pods in the "default" namespace:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["default"]
    }
  }
}
```

## Get Pod Logs - Multiple Namespaces

Retrieve logs from multiple namespaces:

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["default", "kube-system", "ingress-nginx"]
    }
  }
}
```

## Get Pod Logs - Filtered by Deployment

Retrieve logs only from specific deployments:

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["default"],
      "deployments": ["nginx", "api-gateway"],
      "tail_lines": 200
    }
  }
}
```

## Get Pod Logs - Previous Container

Retrieve logs from previously terminated containers (useful for crashed pods):

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["default"],
      "deployments": ["my-app"],
      "tail_lines": 500,
      "previous": true
    }
  }
}
```

## Gateway Failure Tracing Example

Example for tracing traffic across multiple gateway hops:

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": [
        "ingress-nginx",
        "api-gateway",
        "backend-services",
        "database"
      ],
      "deployments": [
        "nginx-ingress-controller",
        "api-gateway",
        "user-service",
        "postgres"
      ],
      "tail_lines": 1000
    }
  }
}
```

## Usage with curl (HTTP wrapper needed)

Note: The MCP server uses stdio transport by default. To use with curl, you would need an HTTP wrapper. 
For direct testing, pipe JSON to the server:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-server
```

## Usage with a script

```bash
#!/bin/bash

# Send multiple requests in sequence
(
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  sleep 0.5
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  sleep 0.5
  echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_pod_logs","arguments":{"namespaces":["default"],"tail_lines":100}}}'
  sleep 2
) | ./mcp-server
```
