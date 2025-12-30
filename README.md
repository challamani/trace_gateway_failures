# trace_gateway_failures

MCP Server to retrieve Gateway logs and metrics to identify traffic failures. This server implements the Model Context Protocol (MCP) and provides tools to retrieve Kubernetes pod logs for analyzing where incoming traffic might be failing in your gateway infrastructure.

## Features

- **MCP Protocol Support**: Implements the Model Context Protocol for seamless integration with MCP clients
- **Kubernetes Integration**: Uses the Kubernetes Go client library to interact with clusters
- **Local Kubeconfig**: Automatically uses your local `~/.kube/config` for cluster authentication
- **Flexible Filtering**: Filter pods by namespaces and deployment names
- **Comprehensive Logging**: Retrieve logs from all matching pods and containers
- **Gateway Failure Analysis**: Designed to help trace where incoming traffic is failing across your infrastructure

## Prerequisites

- Go 1.21 or later
- Access to a Kubernetes cluster
- Valid kubeconfig file at `~/.kube/config`
- Appropriate RBAC permissions to list pods and read logs in target namespaces

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/challamani/trace_gateway_failures.git
cd trace_gateway_failures

# Build the server
go build -o mcp-server .

# Run the server
./mcp-server
```

### Direct Run

```bash
go run main.go
```

## Usage

The MCP server communicates via stdin/stdout using the JSON-RPC 2.0 protocol. It can be integrated with any MCP client.

### Available Tools

#### `get_pod_logs`

Retrieves pod logs for given namespaces and deployment names to trace gateway failures.

**Parameters:**

- `namespaces` (array of strings, required): List of Kubernetes namespaces to search for pods
- `deployments` (array of strings, optional): List of deployment names to filter pods. If not provided, all pods in the namespace are included
- `tail_lines` (integer, optional, default: 100): Number of lines from the end of the logs to retrieve
- `previous` (boolean, optional, default: false): Retrieve logs from previous terminated container

**Example Request:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["default", "kube-system"],
      "deployments": ["nginx", "gateway"],
      "tail_lines": 200,
      "previous": false
    }
  }
}
```

**Example Response:**

The tool returns formatted logs showing:
- Namespace and pod information
- Pod status and owner references (Deployment/ReplicaSet)
- Logs from each container in the pod
- Clear separation between different pods and containers

## MCP Protocol Support

The server implements the following MCP methods:

- `initialize`: Initialize the MCP connection
- `tools/list`: List available tools
- `tools/call`: Execute a specific tool

## Architecture

```
main.go
└── pkg/mcp/
    └── server.go
        ├── Server: Main server structure
        ├── StdioTransport: Handles stdio communication
        ├── InitializeKubeClient(): Initializes Kubernetes client
        ├── Start(): Starts the server loop
        ├── handleRequest(): Routes MCP requests
        └── getPodLogs(): Retrieves and formats pod logs
```

## Use Case: Tracing Gateway Failures

This tool is designed to help identify where incoming traffic is failing in your Kubernetes infrastructure:

1. **Multi-Hop Analysis**: Check logs across multiple namespaces to trace the request path
2. **Deployment Filtering**: Focus on specific gateway deployments (e.g., ingress controllers, API gateways)
3. **Error Pattern Detection**: Review logs from multiple pods to identify common failure patterns
4. **Recent History**: Use `tail_lines` to focus on recent events
5. **Crash Analysis**: Use `previous: true` to examine logs from crashed containers

### Example Workflow

```bash
# Check gateway and upstream service logs
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": ["ingress-nginx", "api-gateway", "backend-services"],
      "deployments": ["nginx-ingress", "api-gateway", "user-service"],
      "tail_lines": 500
    }
  }
}' | ./mcp-server
```

## Configuration

The server uses the default kubeconfig location (`~/.kube/config`). No additional configuration is required as it runs on localhost and uses local kubeconfig for cluster authentication.

### RBAC Requirements

Ensure your kubeconfig user has the following permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-logs-reader
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list"]
```

## Development

### Project Structure

- `main.go`: Entry point for the MCP server
- `pkg/mcp/server.go`: Core MCP server implementation with Kubernetes integration
- `go.mod`: Go module dependencies

### Dependencies

- `k8s.io/client-go`: Kubernetes Go client library
- `k8s.io/api`: Kubernetes API types
- `k8s.io/apimachinery`: Kubernetes API machinery

### Testing

To test the server manually:

```bash
# Start the server
./mcp-server

# In another terminal, send requests via stdin
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-server
```

## Troubleshooting

### "Failed to initialize Kubernetes client"

- Ensure your kubeconfig file exists at `~/.kube/config`
- Verify the kubeconfig is valid and points to an accessible cluster
- Check that you have network connectivity to the cluster

### "Error listing pods in namespace X"

- Verify the namespace exists: `kubectl get namespace`
- Check RBAC permissions for your user
- Ensure the cluster is reachable

### "No pods found matching deployments"

- Verify deployment names are correct
- The tool matches pods by:
  - `app` label matching deployment name
  - Pod name containing deployment name
  - Owner references (ReplicaSet) starting with deployment name

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
