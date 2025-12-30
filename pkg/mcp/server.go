package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/challamani/trace_gateway_failures/pkg/k8s"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToolNames
const (
	GetPodLogsToolName = "get_pod_logs"
)

// Server wraps the MCP server with Kubernetes client
type Server struct {
	mcpServer *server.MCPServer
	k8sClient *k8s.Client
}

// LogTarget represents a target for fetching logs with its own configuration
type LogTarget struct {
	Namespace   string   `json:"namespace"`
	Deployments []string `json:"deployments"`
	TailLines   int64    `json:"tail_lines"`
}

// NewServer creates a new MCP server with Kubernetes integration
func NewServer() (*Server, error) {
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	mcpServer := server.NewMCPServer(
		"Kubernetes Pod Logs Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s := &Server{
		mcpServer: mcpServer,
		k8sClient: k8sClient,
	}

	// Register tool handlers
	mcpServer.AddTool(mcp.Tool{
		Name:        GetPodLogsToolName,
		Description: "Get logs from pods in specified Kubernetes deployments across multiple namespaces. Supports fetching logs from multiple targets with individual configurations.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"targets": map[string]interface{}{
					"type":        "array",
					"description": "Array of log targets, each with namespace, deployments, and tail_lines",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"namespace": map[string]interface{}{
								"type":        "string",
								"description": "Kubernetes namespace to fetch logs from",
							},
							"deployments": map[string]interface{}{
								"type":        "array",
								"description": "Array of deployment names to fetch logs from",
								"items": map[string]interface{}{
									"type": "string",
								},
							},
							"tail_lines": map[string]interface{}{
								"type":        "number",
								"description": "Number of lines to fetch from the end of logs (default: 100)",
							},
						},
						"required": []string{"namespace", "deployments"},
					},
				},
				"previous": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to fetch logs from previous container instance (default: false)",
				},
			},
			Required: []string{"targets"},
		},
	}, s.handleGetPodLogs)

	return s, nil
}

// Start begins serving the MCP server
func (s *Server) Start() error {
	log.Println("Starting Kubernetes Pod Logs MCP Server...")
	return s.mcpServer.Serve()
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(request mcp.ToolsListRequest) (*mcp.ToolsListResult, error) {
	tools := []mcp.Tool{
		{
			Name:        GetPodLogsToolName,
			Description: "Get logs from pods in specified Kubernetes deployments across multiple namespaces. Supports fetching logs from multiple targets with individual configurations.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"targets": map[string]interface{}{
						"type":        "array",
						"description": "Array of log targets, each with namespace, deployments, and tail_lines",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type":        "string",
									"description": "Kubernetes namespace to fetch logs from",
								},
								"deployments": map[string]interface{}{
									"type":        "array",
									"description": "Array of deployment names to fetch logs from",
									"items": map[string]interface{}{
										"type": "string",
									},
								},
								"tail_lines": map[string]interface{}{
									"type":        "number",
									"description": "Number of lines to fetch from the end of logs (default: 100)",
								},
							},
							"required": []string{"namespace", "deployments"},
						},
					},
					"previous": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to fetch logs from previous container instance (default: false)",
					},
				},
				Required: []string{"targets"},
			},
		},
	}

	return &mcp.ToolsListResult{
		Tools: tools,
	}, nil
}

// handleGetPodLogs handles the get_pod_logs tool call
func (s *Server) handleGetPodLogs(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse targets array
	targetsRaw, ok := arguments["targets"]
	if !ok {
		return mcp.NewToolResultError("targets parameter is required"), nil
	}

	targetsArray, ok := targetsRaw.([]interface{})
	if !ok {
		return mcp.NewToolResultError("targets must be an array"), nil
	}

	var targets []LogTarget
	for i, targetRaw := range targetsArray {
		targetMap, ok := targetRaw.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("target at index %d must be an object", i)), nil
		}

		// Parse namespace
		namespace, ok := targetMap["namespace"].(string)
		if !ok || namespace == "" {
			return mcp.NewToolResultError(fmt.Sprintf("target at index %d: namespace is required and must be a string", i)), nil
		}

		// Parse deployments
		deploymentsRaw, ok := targetMap["deployments"]
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("target at index %d: deployments is required", i)), nil
		}

		deploymentsArray, ok := deploymentsRaw.([]interface{})
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("target at index %d: deployments must be an array", i)), nil
		}

		var deployments []string
		for _, d := range deploymentsArray {
			if depStr, ok := d.(string); ok {
				deployments = append(deployments, depStr)
			}
		}

		if len(deployments) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("target at index %d: at least one deployment is required", i)), nil
		}

		// Parse tail_lines (optional, default 100)
		tailLines := int64(100)
		if tailLinesRaw, ok := targetMap["tail_lines"]; ok {
			if tailLinesFloat, ok := tailLinesRaw.(float64); ok {
				tailLines = int64(tailLinesFloat)
			}
		}

		targets = append(targets, LogTarget{
			Namespace:   namespace,
			Deployments: deployments,
			TailLines:   tailLines,
		})
	}

	// Parse previous parameter
	previous := false
	if prevRaw, ok := arguments["previous"]; ok {
		if prevBool, ok := prevRaw.(bool); ok {
			previous = prevBool
		}
	}

	// Get pod logs using the new structure
	logs, err := s.getPodLogs(context.Background(), targets, previous)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
	}

	return mcp.NewToolResultText(logs), nil
}

// getPodLogs fetches logs from pods in the specified targets
func (s *Server) getPodLogs(ctx context.Context, targets []LogTarget, previous bool) (string, error) {
	var allLogs strings.Builder

	for _, target := range targets {
		allLogs.WriteString(fmt.Sprintf("\n=== Namespace: %s ===\n", target.Namespace))

		for _, deployment := range target.Deployments {
			// Get pods for this deployment
			pods, err := s.k8sClient.Clientset.CoreV1().Pods(target.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", deployment),
			})
			if err != nil {
				allLogs.WriteString(fmt.Sprintf("Error listing pods for deployment %s: %v\n", deployment, err))
				continue
			}

			if len(pods.Items) == 0 {
				allLogs.WriteString(fmt.Sprintf("No pods found for deployment: %s\n", deployment))
				continue
			}

			// Process each pod
			for _, pod := range pods.Items {
				allLogs.WriteString(fmt.Sprintf("\n--- Pod: %s (Deployment: %s) ---\n", pod.Name, deployment))

				// Get logs from all containers in the pod
				for _, container := range pod.Spec.Containers {
					containerLogs, err := s.getContainerLogs(ctx, target.Namespace, pod.Name, container.Name, target.TailLines, previous)
					if err != nil {
						allLogs.WriteString(fmt.Sprintf("Error getting logs for container %s: %v\n", container.Name, err))
						continue
					}
					allLogs.WriteString(fmt.Sprintf("\nContainer: %s\n", container.Name))
					allLogs.WriteString(containerLogs)
					allLogs.WriteString("\n")
				}

				// Get logs from init containers if they exist
				for _, initContainer := range pod.Spec.InitContainers {
					initContainerLogs, err := s.getContainerLogs(ctx, target.Namespace, pod.Name, initContainer.Name, target.TailLines, previous)
					if err != nil {
						allLogs.WriteString(fmt.Sprintf("Error getting logs for init container %s: %v\n", initContainer.Name, err))
						continue
					}
					allLogs.WriteString(fmt.Sprintf("\n[INIT] Container: %s\n", initContainer.Name))
					allLogs.WriteString(initContainerLogs)
					allLogs.WriteString("\n")
				}
			}
		}
	}

	return allLogs.String(), nil
}

// getContainerLogs fetches logs from a specific container
func (s *Server) getContainerLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64, previous bool) (string, error) {
	logOptions := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
		Previous:  previous,
	}

	req := s.k8sClient.Clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("error opening log stream: %w", err)
	}
	defer podLogs.Close()

	var logs strings.Builder
	reader := bufio.NewReader(podLogs)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if line != "" {
				logs.WriteString(line)
			}
			break
		}
		if err != nil {
			return "", fmt.Errorf("error reading logs: %w", err)
		}
		logs.WriteString(line)
	}

	return logs.String(), nil
}

// MarshalJSON implements custom JSON marshaling for the server
func (s *Server) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":    "Kubernetes Pod Logs Server",
		"version": "1.0.0",
	})
}
