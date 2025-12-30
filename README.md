# Trace Gateway Failures

A comprehensive tool for tracing and diagnosing gateway failures in Kubernetes environments with support for Istio service mesh and advanced pod log collection.

## Features

- **Advanced Pod Log Collection**: Retrieve logs from multiple namespaces and deployments in a single request
- **Init Container Support**: Automatically detects and labels init container logs with `[INIT]` prefix
- **Istio Service Mesh Integration**: Captures istio-proxy sidecar logs for complete traffic visibility
- **Flexible Configuration**: Configure tail lines per target and historical log retrieval globally
- **Multi-Deployment Support**: Query logs from multiple deployments across different namespaces simultaneously

## API Documentation

### get_pod_logs Endpoint

The `get_pod_logs` API endpoint accepts a structured request to fetch logs from multiple Kubernetes pods across different namespaces and deployments.

#### API Structure

```json
{
  "targets": [
    {
      "namespace": "string",
      "deployments": ["string"],
      "tail_lines": integer
    }
  ],
  "previous": boolean
}
```

#### Parameters

- **targets** (array, required): An array of target objects defining which pods to query
  - **namespace** (string, required): The Kubernetes namespace to query
  - **deployments** (array of strings, required): List of deployment names within the namespace
  - **tail_lines** (integer, optional): Number of log lines to retrieve per container (default: 100)

- **previous** (boolean, optional): Global parameter to retrieve logs from previous/terminated containers (default: false)

#### Request Examples

##### Basic Single Namespace Query

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway", "auth-service"],
      "tail_lines": 200
    }
  ]
}
```

##### Multi-Namespace Query with Different Tail Lines

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway"],
      "tail_lines": 500
    },
    {
      "namespace": "staging",
      "deployments": ["api-gateway", "payment-service"],
      "tail_lines": 100
    }
  ],
  "previous": false
}
```

##### Query with Previous Container Logs

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway"]
    }
  ],
  "previous": true
}
```

#### Response Structure

The response includes logs from all containers in the specified pods, including:
- **Application containers**: Standard container logs
- **Init containers**: Labeled with `[INIT]` prefix
- **Istio sidecar containers**: istio-proxy logs when Istio is enabled

##### Example Response

```json
{
  "logs": {
    "production": {
      "api-gateway-7d8f9c5b6-x4k2m": {
        "[INIT] istio-init": [
          "2025-12-30T13:10:15Z Initializing iptables rules",
          "2025-12-30T13:10:16Z iptables configuration completed successfully"
        ],
        "api-gateway": [
          "2025-12-30T13:10:30Z Server started on port 8080",
          "2025-12-30T13:11:00Z Processing request GET /api/v1/status",
          "2025-12-30T13:11:45Z Error: Connection timeout to backend service"
        ],
        "istio-proxy": [
          "2025-12-30T13:10:20Z Envoy proxy initialized",
          "2025-12-30T13:11:00Z [outbound] upstream connect error: connection timeout",
          "2025-12-30T13:11:45Z [inbound] request failed with status 504"
        ]
      },
      "api-gateway-7d8f9c5b6-y9p3n": {
        "[INIT] istio-init": [
          "2025-12-30T13:09:45Z Initializing iptables rules"
        ],
        "api-gateway": [
          "2025-12-30T13:10:00Z Server started on port 8080",
          "2025-12-30T13:10:30Z Health check passed"
        ],
        "istio-proxy": [
          "2025-12-30T13:09:50Z Envoy proxy initialized",
          "2025-12-30T13:10:00Z Listener warming complete"
        ]
      }
    }
  },
  "errors": []
}
```

## Container Type Identification

### Init Containers

Init containers run before the main application container starts. Logs from init containers are automatically identified and prefixed with `[INIT]` for easy identification.

**Example:**
```
[INIT] istio-init: Initializing iptables rules
[INIT] config-loader: Loading configuration from ConfigMap
```

### Istio Proxy Sidecar

When running in an Istio service mesh, the `istio-proxy` container logs provide valuable insights into:
- Inbound and outbound traffic
- Service mesh configuration
- Connection errors and timeouts
- TLS/mTLS handshake issues
- Circuit breaker activations

**Example istio-proxy logs:**
```
2025-12-30T13:11:00Z [outbound] upstream connect error: connection timeout
2025-12-30T13:11:45Z [inbound] request failed with status 504
2025-12-30T13:12:00Z TLS handshake failed: certificate validation error
```

## Migration Guide

### Migrating from Previous API Version

If you're using an older version of this API, here's how to migrate to the new `targets` array structure:

#### Old API Format (Deprecated)

```json
{
  "namespace": "production",
  "deployments": ["api-gateway", "auth-service"],
  "tail_lines": 200,
  "previous": false
}
```

#### New API Format

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway", "auth-service"],
      "tail_lines": 200
    }
  ],
  "previous": false
}
```

### Key Changes

1. **targets array**: All namespace/deployment configurations are now wrapped in a `targets` array
2. **previous parameter**: Moved to the root level as a global parameter affecting all targets
3. **Multiple namespaces**: You can now query multiple namespaces in a single request

### Migration Benefits

- **Batch Operations**: Query multiple namespaces simultaneously
- **Flexible Configuration**: Set different tail_lines per namespace/deployment group
- **Better Organization**: Clearer structure for complex queries
- **Backward Compatibility**: Single-namespace queries are still simple with one target object

## Use Cases

### Debugging Gateway Timeouts

When experiencing gateway timeouts, query both the gateway and backend service logs along with Istio proxy logs:

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway", "backend-service"],
      "tail_lines": 500
    }
  ],
  "previous": false
}
```

Look for:
- Connection timeout errors in application logs
- Upstream connection errors in istio-proxy logs
- Init container failures that might prevent proper startup

### Cross-Environment Comparison

Compare behavior across different environments:

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["api-gateway"],
      "tail_lines": 300
    },
    {
      "namespace": "staging",
      "deployments": ["api-gateway"],
      "tail_lines": 300
    }
  ]
}
```

### Investigating Crashed Pods

When pods are crash-looping, retrieve logs from previous containers:

```json
{
  "targets": [
    {
      "namespace": "production",
      "deployments": ["problematic-service"],
      "tail_lines": 1000
    }
  ],
  "previous": true
}
```

## Installation

```bash
# Clone the repository
git clone https://github.com/challamani/trace_gateway_failures.git

# Navigate to the project directory
cd trace_gateway_failures

# Install dependencies
pip install -r requirements.txt
```

## Configuration

Configure your Kubernetes context to point to the cluster you want to monitor:

```bash
kubectl config use-context <your-cluster-context>
```

## Usage

```bash
# Start the service
python app.py

# Make API requests
curl -X POST http://localhost:8080/get_pod_logs \
  -H "Content-Type: application/json" \
  -d @request.json
```

## Best Practices

1. **Start with reasonable tail_lines**: Begin with 100-500 lines to avoid overwhelming output
2. **Use previous flag judiciously**: Only enable when investigating crashes or restarts
3. **Check istio-proxy logs**: Always review sidecar logs for network-related issues
4. **Watch for [INIT] logs**: Init container failures can prevent pods from starting
5. **Batch related services**: Query related services together for correlation analysis

## Troubleshooting

### No logs returned

- Verify the namespace and deployment names are correct
- Check that pods are running: `kubectl get pods -n <namespace>`
- Ensure your Kubernetes context has proper RBAC permissions

### Missing istio-proxy logs

- Verify Istio sidecar injection is enabled for the namespace
- Check pod annotations: `kubectl get pod <pod-name> -n <namespace> -o yaml`

### [INIT] logs not appearing

- Init containers may have already completed; check pod status
- Use `previous: true` if init containers failed and pod restarted

## Contributing

Contributions are welcome! Please submit pull requests or open issues for bugs and feature requests.

## License

MIT License - see LICENSE file for details

---

**Last Updated**: 2025-12-30  
**Version**: 2.0.0  
**Maintainer**: @challamani
