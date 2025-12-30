package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/challamani/trace_gateway_failures/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	toolName        = "get_pod_logs"
	toolDescription = "Retrieve logs from pods in specified namespaces and deployments with support for init containers"
)

type MCPServer struct {
	config    *config.Config
	k8sClient *kubernetes.Clientset
}

type NamespaceConfig struct {
	Name        string   `json:"name"`
	Deployments []string `json:"deployments"`
	TailLines   int64    `json:"tail_lines,omitempty"`
}

func NewMCPServer(cfg *config.Config) (*MCPServer, error) {
	k8sClient, err := createK8sClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &MCPServer{
		config:    cfg,
		k8sClient: k8sClient,
	}, nil
}

func createK8sClient(cfg *config.Config) (*kubernetes.Clientset, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.Kubernetes.InCluster {
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		kubeconfigPath := cfg.Kubernetes.KubeconfigPath
		if kubeconfigPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			kubeconfigPath = homeDir + "/.kube/config"
		}

		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return clientset, nil
}

func (s *MCPServer) Start() error {
	mcpServer := server.NewMCPServer(
		"Kubernetes Pod Logs MCP Server",
		"1.0.0",
		server.WithToolHandler(s.handleToolsList, s.handleToolCall),
	)

	if err := server.ServeStdio(mcpServer); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	return nil
}

func (s *MCPServer) handleToolsList() ([]mcp.Tool, error) {
	tools := []mcp.Tool{
		{
			Name:        toolName,
			Description: toolDescription,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"namespaces": map[string]interface{}{
						"type":        "array",
						"description": "Array of namespace configurations with deployments and optional tail_lines",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Kubernetes namespace name",
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
									"description": "Number of lines to retrieve from the end of logs for this namespace (default: 100)",
									"default":     100,
								},
							},
							"required": []string{"name", "deployments"},
						},
					},
					"include_timestamps": map[string]interface{}{
						"type":        "boolean",
						"description": "Include timestamps in log output",
						"default":     false,
					},
					"since_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Only return logs newer than a relative duration in seconds",
					},
					"previous": map[string]interface{}{
						"type":        "boolean",
						"description": "Return previous terminated container logs",
						"default":     false,
					},
				},
				Required: []string{"namespaces"},
			},
		},
	}

	return tools, nil
}

func (s *MCPServer) handleToolCall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Name != toolName {
		return nil, fmt.Errorf("unknown tool: %s", request.Params.Name)
	}

	return s.handleGetPodLogs(ctx, request.Params.Arguments)
}

func (s *MCPServer) handleGetPodLogs(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse namespaces configuration
	namespacesRaw, ok := arguments["namespaces"]
	if !ok {
		return nil, fmt.Errorf("namespaces parameter is required")
	}

	namespacesArray, ok := namespacesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("namespaces must be an array")
	}

	var namespaceConfigs []NamespaceConfig
	for _, nsRaw := range namespacesArray {
		nsMap, ok := nsRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each namespace must be an object")
		}

		name, ok := nsMap["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("namespace name is required and must be a string")
		}

		deploymentsRaw, ok := nsMap["deployments"]
		if !ok {
			return nil, fmt.Errorf("deployments are required for namespace %s", name)
		}

		deploymentsArray, ok := deploymentsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("deployments must be an array for namespace %s", name)
		}

		var deployments []string
		for _, d := range deploymentsArray {
			if depStr, ok := d.(string); ok {
				deployments = append(deployments, depStr)
			}
		}

		if len(deployments) == 0 {
			return nil, fmt.Errorf("at least one deployment is required for namespace %s", name)
		}

		tailLines := int64(100) // default
		if tl, ok := nsMap["tail_lines"]; ok {
			if tlFloat, ok := tl.(float64); ok {
				tailLines = int64(tlFloat)
			}
		}

		namespaceConfigs = append(namespaceConfigs, NamespaceConfig{
			Name:        name,
			Deployments: deployments,
			TailLines:   tailLines,
		})
	}

	// Parse optional parameters
	includeTimestamps := false
	if ts, ok := arguments["include_timestamps"].(bool); ok {
		includeTimestamps = ts
	}

	var sinceSeconds *int64
	if ss, ok := arguments["since_seconds"].(float64); ok {
		seconds := int64(ss)
		sinceSeconds = &seconds
	}

	previous := false
	if prev, ok := arguments["previous"].(bool); ok {
		previous = prev
	}

	// Get pod logs
	logs, err := s.getPodLogs(ctx, namespaceConfigs, includeTimestamps, sinceSeconds, previous)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod logs: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []interface{}{
			mcp.TextContent{
				Type: "text",
				Text: logs,
			},
		},
	}, nil
}

func (s *MCPServer) getPodLogs(ctx context.Context, namespaceConfigs []NamespaceConfig, includeTimestamps bool, sinceSeconds *int64, previous bool) (string, error) {
	var allLogs strings.Builder

	for _, nsConfig := range namespaceConfigs {
		namespace := nsConfig.Name
		deployments := nsConfig.Deployments
		tailLines := nsConfig.TailLines

		allLogs.WriteString(fmt.Sprintf("\n=== Namespace: %s ===\n", namespace))

		pods, err := s.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing pods in namespace %s: %v", namespace, err)
			allLogs.WriteString(fmt.Sprintf("Error listing pods: %v\n", err))
			continue
		}

		filteredPods := filterPodsByDeployment(pods.Items, deployments)
		if len(filteredPods) == 0 {
			allLogs.WriteString(fmt.Sprintf("No pods found for deployments: %s\n", strings.Join(deployments, ", ")))
			continue
		}

		for _, pod := range filteredPods {
			ownerInfo := getOwnerInfo(&pod)
			allLogs.WriteString(fmt.Sprintf("\n--- Pod: %s (Owner: %s) ---\n", pod.Name, ownerInfo))

			// Get logs from init containers first
			for _, container := range pod.Spec.InitContainers {
				containerName := fmt.Sprintf("[INIT] %s", container.Name)
				allLogs.WriteString(fmt.Sprintf("\nContainer: %s\n", containerName))

				logOptions := &metav1.PodLogOptions{
					Container:  container.Name,
					Timestamps: includeTimestamps,
					TailLines:  &tailLines,
					Previous:   previous,
				}

				if sinceSeconds != nil {
					logOptions.SinceSeconds = sinceSeconds
				}

				req := s.k8sClient.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions)
				podLogs, err := req.Stream(ctx)
				if err != nil {
					log.Printf("Error getting logs for init container %s in pod %s: %v", container.Name, pod.Name, err)
					allLogs.WriteString(fmt.Sprintf("Error getting logs: %v\n", err))
					continue
				}

				scanner := bufio.NewScanner(podLogs)
				for scanner.Scan() {
					allLogs.WriteString(scanner.Text() + "\n")
				}
				podLogs.Close()

				if err := scanner.Err(); err != nil && err != io.EOF {
					log.Printf("Error reading logs for init container %s: %v", container.Name, err)
					allLogs.WriteString(fmt.Sprintf("Error reading logs: %v\n", err))
				}
			}

			// Get logs from regular containers
			for _, container := range pod.Spec.Containers {
				allLogs.WriteString(fmt.Sprintf("\nContainer: %s\n", container.Name))

				logOptions := &metav1.PodLogOptions{
					Container:  container.Name,
					Timestamps: includeTimestamps,
					TailLines:  &tailLines,
					Previous:   previous,
				}

				if sinceSeconds != nil {
					logOptions.SinceSeconds = sinceSeconds
				}

				req := s.k8sClient.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions)
				podLogs, err := req.Stream(ctx)
				if err != nil {
					log.Printf("Error getting logs for container %s in pod %s: %v", container.Name, pod.Name, err)
					allLogs.WriteString(fmt.Sprintf("Error getting logs: %v\n", err))
					continue
				}

				scanner := bufio.NewScanner(podLogs)
				for scanner.Scan() {
					allLogs.WriteString(scanner.Text() + "\n")
				}
				podLogs.Close()

				if err := scanner.Err(); err != nil && err != io.EOF {
					log.Printf("Error reading logs for container %s: %v", container.Name, err)
					allLogs.WriteString(fmt.Sprintf("Error reading logs: %v\n", err))
				}
			}
		}
	}

	return allLogs.String(), nil
}

func filterPodsByDeployment(pods []interface{}, deployments []string) []interface{} {
	if len(deployments) == 0 {
		return pods
	}

	deploymentSet := make(map[string]bool)
	for _, d := range deployments {
		deploymentSet[d] = true
	}

	var filtered []interface{}
	for _, pod := range pods {
		podObj, ok := pod.(map[string]interface{})
		if !ok {
			continue
		}

		metadata, ok := podObj["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		ownerReferences, ok := metadata["ownerReferences"].([]interface{})
		if !ok {
			continue
		}

		for _, owner := range ownerReferences {
			ownerObj, ok := owner.(map[string]interface{})
			if !ok {
				continue
			}

			kind, _ := ownerObj["kind"].(string)
			name, _ := ownerObj["name"].(string)

			if kind == "ReplicaSet" {
				// Extract deployment name from ReplicaSet name
				// ReplicaSet names are typically in format: deployment-name-xxxxx
				parts := strings.Split(name, "-")
				if len(parts) >= 2 {
					deploymentName := strings.Join(parts[:len(parts)-1], "-")
					if deploymentSet[deploymentName] {
						filtered = append(filtered, pod)
						break
					}
				}
			}
		}
	}

	return filtered
}

func getOwnerInfo(pod interface{}) string {
	podObj, ok := pod.(map[string]interface{})
	if !ok {
		return "Unknown"
	}

	metadata, ok := podObj["metadata"].(map[string]interface{})
	if !ok {
		return "Unknown"
	}

	ownerReferences, ok := metadata["ownerReferences"].([]interface{})
	if !ok || len(ownerReferences) == 0 {
		return "None"
	}

	var owners []string
	for _, owner := range ownerReferences {
		ownerObj, ok := owner.(map[string]interface{})
		if !ok {
			continue
		}

		kind, _ := ownerObj["kind"].(string)
		name, _ := ownerObj["name"].(string)
		owners = append(owners, fmt.Sprintf("%s/%s", kind, name))
	}

	return strings.Join(owners, ", ")
}

func sendError(w io.Writer, id interface{}, errMsg string) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    -32603,
			"message": errMsg,
		},
	}
	json.NewEncoder(w).Encode(response)
}
