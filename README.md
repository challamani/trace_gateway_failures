# Kubernetes Gateway Failure Tracing MCP Server

An MCP (Model Context Protocol) server that helps trace and analyze Kubernetes gateway failures by providing intelligent log retrieval and analysis capabilities.

## Overview

This server exposes tools to:
- Retrieve pod logs from multiple Kubernetes namespaces with deployment-specific filtering
- Support for init containers (labeled with `[INIT]` prefix)
- Analyze gateway-related failures across your Kubernetes infrastructure
- Filter logs by deployment names within each namespace
- Configure different tail line limits per namespace

## Features

- 🔍 **Multi-namespace log retrieval** with namespace-scoped deployment filtering
- 🎯 **Deployment-specific filtering** - specify which deployments to monitor per namespace
- 🔧 **Init container support** - automatically detects and labels init containers
- 📊 **Structured log output** with clear pod, container, and namespace separation
- ⚙️ **Flexible configuration** - different tail line limits per namespace
- 🔄 **Previous container logs** support for troubleshooting crashed containers
- 🚀 **Built with MCP SDK** for seamless AI assistant integration

## Installation

### Prerequisites

- Python 3.10 or higher
- Access to a Kubernetes cluster with valid `kubectl` configuration
- `kubectl` command-line tool installed and configured

### Install from Source

```bash
# Clone the repository
git clone https://github.com/challamani/trace_gateway_failures.git
cd trace_gateway_failures

# Install the package
pip install -e .
```

## Usage

### Running the Server

You can run the server using either `stdio` or `sse` transport:

```bash
# Using stdio transport (default)
python -m trace_gateway_failures

# Using SSE transport
python -m trace_gateway_failures --transport sse
```

### Available Tools

#### `get_pod_logs`

Retrieves pod logs for given namespaces and deployment names to trace gateway failures. **Now supports init containers** (labeled with `[INIT]` prefix) and **namespace-scoped deployment filtering**.

**Parameters:**

- `namespaces` (array of objects, required): Array of namespace configurations
  - `name` (string, required): Kubernetes namespace name
  - `deployments` (array of strings, required): List of deployment names to filter pods in this namespace
  - `tail_lines` (integer, optional, default: 100): Number of lines from the end of logs for this namespace
- `previous` (boolean, optional, default: false): Retrieve logs from previous terminated container

**Example Request - Single Namespace:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": [
        {
          "name": "production",
          "deployments": ["api-gateway", "auth-service"],
          "tail_lines": 200
        }
      ],
      "previous": false
    }
  }
}
```

**Example Request - Multiple Namespaces with Different Configurations:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_pod_logs",
    "arguments": {
      "namespaces": [
        {
          "name": "production",
          "deployments": ["api-gateway"],
          "tail_lines": 500
        },
        {
          "name": "staging",
          "deployments": ["api-gateway", "payment-service"],
          "tail_lines": 100
        }
      ],
      "previous": false
    }
  }
}
```

**Example Response:**

The tool returns formatted logs showing:
- Namespace and pod information
- Pod status and owner references (Deployment/ReplicaSet)
- **Init container logs with `[INIT]` prefix** (e.g., `[INIT] istio-init`)
- Logs from all regular containers (including sidecars like `istio-proxy`)
- Clear separation between different pods and containers

**Sample Output:**
```
=== Pod Logs Analysis for Gateway Failure Tracing ===

Namespace: production
================================================================================

Pod: api-gateway-7d8f9c5b6-x4k2m (Status: Running)
Owner: ReplicaSet/api-gateway-7d8f9c5b6
--------------------------------------------------------------------------------

Container: [INIT] istio-init
2025-12-30T15:10:15Z Initializing iptables rules
2025-12-30T15:10:16Z iptables configuration completed

Container: api-gateway
2025-12-30T15:10:30Z Server started on port 8080
2025-12-30T15:11:00Z Processing request GET /api/v1/status

Container: istio-proxy
2025-12-30T15:10:20Z Envoy proxy initialized
2025-12-30T15:11:00Z [outbound] upstream connect to backend-service:8080
```

## Integration with AI Assistants

### Claude Desktop Configuration

Add to your Claude Desktop configuration file:

**MacOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "trace_gateway_failures": {
      "command": "python",
      "args": ["-m", "trace_gateway_failures"]
    }
  }
}
```

### Example Workflow

1. **Ask Claude to investigate gateway failures:**
   ```
   "Check the logs for api-gateway and auth-service deployments in the production 
   namespace for any errors in the last 200 lines"
   ```

2. **Claude will use the tool with the new format:**
   ```json
   {
     "namespaces": [
       {
         "name": "production",
         "deployments": ["api-gateway", "auth-service"],
         "tail_lines": 200
       }
     ]
   }
   ```

3. **Analyze results across multiple namespaces:**
   ```
   "Compare the gateway logs between production and staging environments, 
   focusing on the api-gateway deployment"
   ```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    AI Assistant (Claude)                 │
└────────────────────┬────────────────────────────────────┘
                     │ MCP Protocol
                     │
┌────────────────────▼────────────────────────────────────┐
│         Trace Gateway Failures MCP Server               │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Tools:                                          │  │
│  │  - get_pod_logs (namespace + deployment filter) │  │
│  │  - init container detection                     │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │ kubectl commands
                     │
┌────────────────────▼────────────────────────────────────┐
│              Kubernetes Cluster                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │ Namespace 1 │  │ Namespace 2 │  │ Namespace N │    │
│  │ - Pods      │  │ - Pods      │  │ - Pods      │    │
│  │ - Init Ctrs │  │ - Init Ctrs │  │ - Init Ctrs │    │
│  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## Development

### Project Structure

```
trace_gateway_failures/
├── src/
│   └── trace_gateway_failures/
│       ├── __init__.py
│       ├── __main__.py
│       └── server.py
├── pyproject.toml
└── README.md
```

### Running Tests

```bash
# Run the server in debug mode
python -m trace_gateway_failures --transport stdio
```

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Troubleshooting

### kubectl Not Found
Ensure `kubectl` is installed and available in your system PATH.

### Permission Denied
Verify your Kubernetes configuration has appropriate RBAC permissions to read pods and logs.

### No Logs Retrieved
- Check if pods exist in the specified namespaces
- Verify deployment names are correct
- Ensure pods are in Running state or have terminated containers for previous logs

## License

[Add your license here]

## Author

Challamani - [GitHub Profile](https://github.com/challamani)

## Acknowledgments

- Built with [Model Context Protocol SDK](https://github.com/modelcontextprotocol)
- Kubernetes log retrieval powered by `kubectl`