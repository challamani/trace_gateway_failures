package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	MCPVersion = "2024-11-05"
)

// Server represents the MCP server
type Server struct {
	clientset *kubernetes.Clientset
}

// NewServer creates a new MCP server instance
func NewServer() *Server {
	return &Server{}
}

// InitializeKubeClient initializes the Kubernetes client using local kubeconfig
func (s *Server) InitializeKubeClient() error {
	// Use the default kubeconfig path
	kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	if home := homedir.HomeDir(); home != "" && kubeconfig == "" {
		kubeconfig = home + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	s.clientset = clientset
	log.Println("Kubernetes client initialized successfully")
	return nil
}

// StdioTransport handles stdio-based communication
type StdioTransport struct {
	Reader io.Reader
	Writer io.Writer
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in MCP protocol
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Start starts the MCP server with the given transport
func (s *Server) Start(transport *StdioTransport) error {
	scanner := bufio.NewScanner(transport.Reader)
	encoder := json.NewEncoder(transport.Writer)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			s.sendError(encoder, nil, -32700, "Parse error", nil)
			continue
		}

		response := s.handleRequest(&req)
		if err := encoder.Encode(response); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest processes an MCP request and returns a response
func (s *Server) handleRequest(req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": MCPVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "trace_gateway_failures",
				"version": "1.0.0",
			},
		},
	}
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req *MCPRequest) *MCPResponse {
	tools := []map[string]interface{}{
		{
			"name":        "get_pod_logs",
			"description": "Retrieve pod logs for given namespaces and deployment names to trace gateway failures",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespaces": map[string]interface{}{
						"type":        "array",
						"description": "List of Kubernetes namespaces to search for pods",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"deployments": map[string]interface{}{
						"type":        "array",
						"description": "List of deployment names to filter pods",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"tail_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Number of lines from the end of the logs to retrieve (default: 100)",
						"default":     100,
					},
					"previous": map[string]interface{}{
						"type":        "boolean",
						"description": "Retrieve logs from previous terminated container (default: false)",
						"default":     false,
					},
				},
				"required": []string{"namespaces"},
			},
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// ToolCallParams represents parameters for a tool call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(req *MCPRequest) *MCPResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	switch params.Name {
	case "get_pod_logs":
		return s.handleGetPodLogs(req, params.Arguments)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Tool not found",
			},
		}
	}
}

// handleGetPodLogs retrieves pod logs based on namespaces and deployments
func (s *Server) handleGetPodLogs(req *MCPRequest, args map[string]interface{}) *MCPResponse {
	// Parse arguments
	var namespaces []string
	if ns, ok := args["namespaces"].([]interface{}); ok {
		for _, n := range ns {
			if nsStr, ok := n.(string); ok {
				namespaces = append(namespaces, nsStr)
			}
		}
	}

	var deployments []string
	if deps, ok := args["deployments"].([]interface{}); ok {
		for _, d := range deps {
			if depStr, ok := d.(string); ok {
				deployments = append(deployments, depStr)
			}
		}
	}

	tailLines := int64(100)
	if tl, ok := args["tail_lines"].(float64); ok {
		tailLines = int64(tl)
	}

	previous := false
	if prev, ok := args["previous"].(bool); ok {
		previous = prev
	}

	if len(namespaces) == 0 {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "At least one namespace is required",
			},
		}
	}

	// Retrieve logs
	logsResult, err := s.getPodLogs(namespaces, deployments, tailLines, previous)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32000,
				Message: "Failed to retrieve pod logs",
				Data:    err.Error(),
			},
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": logsResult,
				},
			},
		},
	}
}

// getPodLogs retrieves logs from pods matching the criteria
func (s *Server) getPodLogs(namespaces []string, deployments []string, tailLines int64, previous bool) (string, error) {
	if s.clientset == nil {
		return "", fmt.Errorf("Kubernetes client not initialized - ensure valid kubeconfig is available")
	}

	ctx := context.Background()
	var result strings.Builder

	result.WriteString("=== Pod Logs Analysis for Gateway Failure Tracing ===\n\n")

	for _, namespace := range namespaces {
		result.WriteString(fmt.Sprintf("Namespace: %s\n", namespace))
		result.WriteString(strings.Repeat("=", 80) + "\n\n")

		// List all pods in the namespace
		pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			result.WriteString(fmt.Sprintf("Error listing pods in namespace %s: %v\n\n", namespace, err))
			continue
		}

		if len(pods.Items) == 0 {
			result.WriteString(fmt.Sprintf("No pods found in namespace %s\n\n", namespace))
			continue
		}

		// Filter pods by deployment if specified
		filteredPods := s.filterPodsByDeployment(pods.Items, deployments)

		if len(filteredPods) == 0 {
			if len(deployments) > 0 {
				result.WriteString(fmt.Sprintf("No pods found matching deployments %v in namespace %s\n\n", deployments, namespace))
			}
			continue
		}

		// Get logs from each pod
		for _, pod := range filteredPods {
			result.WriteString(fmt.Sprintf("Pod: %s (Status: %s)\n", pod.Name, pod.Status.Phase))

			// Get deployment/replicaset info from owner references
			ownerInfo := s.getOwnerInfo(&pod)
			if ownerInfo != "" {
				result.WriteString(fmt.Sprintf("Owner: %s\n", ownerInfo))
			}

			result.WriteString(strings.Repeat("-", 80) + "\n")

			// Get logs from each container in the pod
			for _, container := range pod.Spec.Containers {
				result.WriteString(fmt.Sprintf("\nContainer: %s\n", container.Name))

				logOptions := &corev1.PodLogOptions{
					Container: container.Name,
					TailLines: &tailLines,
					Previous:  previous,
				}

				logs, err := s.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions).Stream(ctx)
				if err != nil {
					result.WriteString(fmt.Sprintf("Error retrieving logs: %v\n", err))
					continue
				}

				logBytes, err := io.ReadAll(logs)
				logs.Close()
				if err != nil {
					result.WriteString(fmt.Sprintf("Error reading logs: %v\n", err))
					continue
				}

				if len(logBytes) == 0 {
					result.WriteString("(No logs available)\n")
				} else {
					result.WriteString(string(logBytes))
					result.WriteString("\n")
				}
			}

			result.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")
		}
	}

	return result.String(), nil
}

// filterPodsByDeployment filters pods by deployment names
func (s *Server) filterPodsByDeployment(pods []corev1.Pod, deployments []string) []corev1.Pod {
	if len(deployments) == 0 {
		return pods
	}

	var filtered []corev1.Pod
	for _, pod := range pods {
		// Check labels for deployment match
		for _, deployment := range deployments {
			// Check if pod has app label matching deployment
			if appLabel, ok := pod.Labels["app"]; ok && appLabel == deployment {
				filtered = append(filtered, pod)
				break
			}
			// Check if pod name contains deployment name
			if strings.Contains(pod.Name, deployment) {
				filtered = append(filtered, pod)
				break
			}
			// Check owner references
			for _, owner := range pod.OwnerReferences {
				if owner.Kind == "ReplicaSet" && strings.HasPrefix(owner.Name, deployment) {
					filtered = append(filtered, pod)
					break
				}
			}
		}
	}

	return filtered
}

// getOwnerInfo extracts owner information from pod
func (s *Server) getOwnerInfo(pod *corev1.Pod) string {
	if len(pod.OwnerReferences) == 0 {
		return ""
	}

	var owners []string
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
	}

	return strings.Join(owners, ", ")
}

// sendError sends an error response
func (s *Server) sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) {
	response := &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	encoder.Encode(response)
}
