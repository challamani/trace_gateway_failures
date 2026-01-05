# Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         MCP Client                               │
│                    (Claude Desktop, etc.)                        │
└───────────────┬─────────────────────────────────────────────────┘
                │ JSON-RPC 2.0 over stdin/stdout
                │
┌───────────────▼─────────────────────────────────────────────────┐
│                      MCP Server                                  │
│                  (trace-gateway-failures)                        │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              Request Handler                            │    │
│  │  - initialize: Setup protocol connection                │    │
│  │  - tools/list: List available tools                     │    │
│  │  - tools/call: Execute tool (get_pod_logs)              │    │
│  └────────────────────┬───────────────────────────────────┘    │
│                       │                                          │
│  ┌────────────────────▼───────────────────────────────────┐    │
│  │          Kubernetes Client (client-go)                  │    │
│  │  - List Pods in Namespaces                              │    │
│  │  - Filter by Deployment Names                           │    │
│  │  - Retrieve Pod Logs                                    │    │
│  └────────────────────┬───────────────────────────────────┘    │
│                       │                                          │
└───────────────────────┼──────────────────────────────────────────┘
                        │ kubeconfig authentication
                        │
┌───────────────────────▼──────────────────────────────────────────┐
│                    Kubernetes Cluster                            │
│                                                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Namespace  │  │  Namespace  │  │  Namespace  │             │
│  │   default   │  │   ingress   │  │  backend    │             │
│  │             │  │             │  │             │             │
│  │  ┌───────┐  │  │  ┌───────┐  │  │  ┌───────┐  │             │
│  │  │ Pod 1 │  │  │  │ Pod A │  │  │  │ Pod X │  │             │
│  │  └───────┘  │  │  └───────┘  │  │  └───────┘  │             │
│  │  ┌───────┐  │  │  ┌───────┐  │  │  ┌───────┐  │             │
│  │  │ Pod 2 │  │  │  │ Pod B │  │  │  │ Pod Y │  │             │
│  │  └───────┘  │  │  └───────┘  │  │  └───────┘  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

## Data Flow

1. **Client Request**: MCP client sends a JSON-RPC request to the server
2. **Request Parsing**: Server parses the request and routes to appropriate handler
3. **Tool Execution**: For `get_pod_logs`, the server:
   - Validates the namespace and deployment parameters
   - Connects to Kubernetes using local kubeconfig
   - Lists all pods in the specified namespaces
   - Filters pods by deployment names (if provided)
   - Retrieves logs from each container in matching pods
4. **Response Formation**: Server formats the logs and returns them as an MCP response
5. **Client Processing**: Client receives and displays the logs

## Gateway Failure Tracing Flow

```
User Request → Ingress Controller → API Gateway → Backend Service → Database
     ↓                ↓                  ↓              ↓              ↓
  Logs in         Logs in           Logs in        Logs in        Logs in
 Namespace:     Namespace:        Namespace:     Namespace:     Namespace:
  default      ingress-nginx     api-gateway      backend      database

                         ↓
              MCP Server collects logs from all hops
                         ↓
                Analyzes where traffic fails
```

## Tool: get_pod_logs

### Input Schema

```json
{
  "namespaces": ["namespace1", "namespace2"],     // Required
  "deployments": ["deployment1", "deployment2"],  // Optional
  "tail_lines": 100,                              // Optional (default: 100)
  "previous": false                               // Optional (default: false)
}
```

### Output Format

```
=== Pod Logs Analysis for Gateway Failure Tracing ===

Namespace: default
================================================================================

Pod: nginx-deployment-7d8c4c4d9f-abc12 (Status: Running)
Owner: ReplicaSet/nginx-deployment-7d8c4c4d9f
--------------------------------------------------------------------------------

Container: nginx
[Container logs here...]

================================================================================
```

## Security Considerations

- **No RBAC Implementation**: The server relies on local kubeconfig for authentication
- **Local Execution**: Server runs on localhost only
- **Read-Only Access**: Only reads pod information and logs (no write operations)
- **No Secrets Storage**: No credentials or sensitive data stored by the server

## Error Handling

- **Missing Kubeconfig**: Server starts with a warning but continues to run
- **Invalid Namespace**: Returns error message instead of crashing
- **Missing Pods**: Gracefully reports no pods found
- **Log Retrieval Errors**: Reports error per container/pod but continues with others
