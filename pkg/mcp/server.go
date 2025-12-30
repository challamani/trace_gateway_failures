package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Server represents the MCP server
type Server struct {
	clientset *kubernetes.Clientset
}

// NewServer creates a new MCP server instance
func NewServer() (*Server, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Server{
		clientset: clientset,
	}, nil
}

// Target represents a single log collection target
type Target struct {
	Namespace   string   `json:"namespace"`
	Deployments []string `json:"deployments"`
	TailLines   int64    `json:"tail_lines,omitempty"`
}

// LogsRequest represents the request parameters for fetching logs
type LogsRequest struct {
	Targets  []Target `json:"targets"`
	Previous bool     `json:"previous,omitempty"`
}

// GetPodLogs fetches logs from pods based on the targets array
func (s *Server) GetPodLogs(ctx context.Context, req LogsRequest) (string, error) {
	var allLogs strings.Builder

	for _, target := range req.Targets {
		namespace := target.Namespace
		if namespace == "" {
			namespace = "default"
		}

		tailLines := target.TailLines
		if tailLines == 0 {
			tailLines = 100 // Default tail lines
		}

		for _, deploymentName := range target.Deployments {
			// Get deployment to find pods
			deployment, err := s.clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				allLogs.WriteString(fmt.Sprintf("Error getting deployment %s in namespace %s: %v\n\n", deploymentName, namespace, err))
				continue
			}

			// Get pods for this deployment
			labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
			pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				allLogs.WriteString(fmt.Sprintf("Error listing pods for deployment %s: %v\n\n", deploymentName, err))
				continue
			}

			if len(pods.Items) == 0 {
				allLogs.WriteString(fmt.Sprintf("No pods found for deployment %s in namespace %s\n\n", deploymentName, namespace))
				continue
			}

			// Fetch logs from each pod
			for _, pod := range pods.Items {
				allLogs.WriteString(fmt.Sprintf("=== Logs from Pod: %s (Namespace: %s, Deployment: %s) ===\n", pod.Name, namespace, deploymentName))

				// Get logs from init containers
				for _, initContainer := range pod.Spec.InitContainers {
					containerLogs, err := s.fetchContainerLogs(ctx, namespace, pod.Name, initContainer.Name, tailLines, req.Previous, true)
					if err != nil {
						allLogs.WriteString(fmt.Sprintf("[INIT] Container %s - Error: %v\n", initContainer.Name, err))
					} else {
						allLogs.WriteString(fmt.Sprintf("[INIT] Container: %s\n%s\n", initContainer.Name, containerLogs))
					}
				}

				// Get logs from regular containers
				for _, container := range pod.Spec.Containers {
					containerLogs, err := s.fetchContainerLogs(ctx, namespace, pod.Name, container.Name, tailLines, req.Previous, false)
					if err != nil {
						allLogs.WriteString(fmt.Sprintf("Container %s - Error: %v\n", container.Name, err))
					} else {
						allLogs.WriteString(fmt.Sprintf("Container: %s\n%s\n", container.Name, containerLogs))
					}
				}

				allLogs.WriteString("\n")
			}
		}
	}

	if allLogs.Len() == 0 {
		return "No logs found for the specified targets", nil
	}

	return allLogs.String(), nil
}

// fetchContainerLogs retrieves logs from a specific container in a pod
func (s *Server) fetchContainerLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64, previous, isInit bool) (string, error) {
	podLogOpts := &metav1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
		Previous:  previous,
	}

	req := s.clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get log stream: %w", err)
	}
	defer podLogs.Close()

	var logs strings.Builder
	scanner := bufio.NewScanner(podLogs)
	for scanner.Scan() {
		logs.WriteString(scanner.Text())
		logs.WriteString("\n")
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading logs: %w", err)
	}

	return logs.String(), nil
}

// HandleToolCall processes MCP tool calls
func (s *Server) HandleToolCall(toolName string, arguments map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "get_pod_logs":
		return s.handleGetPodLogs(arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// handleGetPodLogs processes the get_pod_logs tool call
func (s *Server) handleGetPodLogs(arguments map[string]interface{}) (interface{}, error) {
	// Parse arguments
	var req LogsRequest

	// Extract targets
	targetsData, ok := arguments["targets"]
	if !ok || targetsData == nil {
		return nil, fmt.Errorf("targets parameter is required")
	}

	// Convert to JSON and back to parse properly
	targetsJSON, err := json.Marshal(targetsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal targets: %w", err)
	}

	if err := json.Unmarshal(targetsJSON, &req.Targets); err != nil {
		return nil, fmt.Errorf("failed to parse targets: %w", err)
	}

	// Extract previous flag
	if previousVal, ok := arguments["previous"]; ok {
		if previousBool, ok := previousVal.(bool); ok {
			req.Previous = previousBool
		}
	}

	// Validate that we have at least one target
	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Fetch logs
	logs, err := s.GetPodLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod logs: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": logs,
			},
		},
	}, nil
}

// GetToolDefinitions returns the MCP tool definitions
func (s *Server) GetToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "get_pod_logs",
			"description": "Fetch logs from Kubernetes pods based on deployment names across multiple namespaces with init container support",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"targets": map[string]interface{}{
						"type":        "array",
						"description": "Array of log collection targets, each specifying namespace, deployments, and tail lines",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"namespace": map[string]interface{}{
									"type":        "string",
									"description": "Kubernetes namespace (defaults to 'default' if not specified)",
								},
								"deployments": map[string]interface{}{
									"type":        "array",
									"description": "Array of deployment names to fetch logs from",
									"items": map[string]interface{}{
										"type": "string",
									},
								},
								"tail_lines": map[string]interface{}{
									"type":        "integer",
									"description": "Number of lines to tail from the end of logs (defaults to 100 if not specified)",
								},
							},
							"required": []string{"namespace", "deployments"},
						},
					},
					"previous": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, fetch logs from previous terminated container instances (globalCurrent Date: 2025-12-30 13:23:05)",
					},
				},
				"required": []string{"targets"},
			},
		},
	}
}
