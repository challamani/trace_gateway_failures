package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sServer struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewK8sServer(kubeconfig string) (*K8sServer, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &K8sServer{
		clientset: clientset,
		config:    config,
	}, nil
}

func (s *K8sServer) Start() error {
	mcpServer := server.NewMCPServer(
		"kubernetes-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	listPodsToolName := "list_pods"
	listPodsTool := mcp.NewTool(listPodsToolName,
		mcp.WithDescription("List all pods in a specific namespace or all namespaces"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace to list pods from, use 'all' for all namespaces"),
		),
	)

	getPodLogsTool := mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Get logs from a specific pod and container"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the pod"),
		),
		mcp.WithString("pod",
			mcp.Required(),
			mcp.Description("Name of the pod"),
		),
		mcp.WithString("container",
			mcp.Description("Name of the container (optional, uses first container if not specified)"),
		),
		mcp.WithNumber("tail_lines",
			mcp.Description("Number of lines to tail (optional, defaults to 100)"),
		),
	)

	describePodTool := mcp.NewTool("describe_pod",
		mcp.WithDescription("Get detailed information about a specific pod"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the pod"),
		),
		mcp.WithString("pod",
			mcp.Required(),
			mcp.Description("Name of the pod"),
		),
	)

	mcpServer.AddTool(listPodsTool, s.handleListPods)
	mcpServer.AddTool(getPodLogsTool, s.handleGetPodLogs)
	mcpServer.AddTool(describePodTool, s.handleDescribePod)

	if err := mcpServer.Serve(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *K8sServer) handleListPods(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := request.Params.Arguments["namespace"].(string)

	if namespace == "all" {
		namespace = ""
	}

	pods, err := s.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
	}

	result := make([]map[string]interface{}, 0, len(pods.Items))
	for _, pod := range pods.Items {
		result = append(result, map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"status":    string(pod.Status.Phase),
			"created":   pod.CreationTimestamp.Format(time.RFC3339),
		})
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *K8sServer) handleGetPodLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := request.Params.Arguments["namespace"].(string)
	podName := request.Params.Arguments["pod"].(string)
	container, _ := request.Params.Arguments["container"].(string)
	tailLines, _ := request.Params.Arguments["tail_lines"].(float64)

	if tailLines == 0 {
		tailLines = 100
	}

	tail := int64(tailLines)
	logOptions := &corev1.PodLogOptions{
		Container: container,
		TailLines: &tail,
	}

	req := s.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	logs, err := req.Stream(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
	}
	defer logs.Close()

	logData, err := io.ReadAll(logs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read pod logs: %v", err)), nil
	}

	return mcp.NewToolResultText(string(logData)), nil
}

func (s *K8sServer) handleDescribePod(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := request.Params.Arguments["namespace"].(string)
	podName := request.Params.Arguments["pod"].(string)

	pod, err := s.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get pod: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Name: %s\n", pod.Name))
	output.WriteString(fmt.Sprintf("Namespace: %s\n", pod.Namespace))
	output.WriteString(fmt.Sprintf("Status: %s\n", pod.Status.Phase))
	output.WriteString(fmt.Sprintf("Created: %s\n", pod.CreationTimestamp.Format(time.RFC3339)))

	if pod.Status.Message != "" {
		output.WriteString(fmt.Sprintf("Message: %s\n", pod.Status.Message))
	}

	if pod.Status.Reason != "" {
		output.WriteString(fmt.Sprintf("Reason: %s\n", pod.Status.Reason))
	}

	output.WriteString("\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		output.WriteString(fmt.Sprintf("  - Name: %s\n", container.Name))
		output.WriteString(fmt.Sprintf("    Image: %s\n", container.Image))
	}

	output.WriteString("\nContainer Statuses:\n")
	for _, status := range pod.Status.ContainerStatuses {
		output.WriteString(fmt.Sprintf("  - Name: %s\n", status.Name))
		output.WriteString(fmt.Sprintf("    Ready: %v\n", status.Ready))
		output.WriteString(fmt.Sprintf("    Restart Count: %d\n", status.RestartCount))

		if status.State.Running != nil {
			output.WriteString(fmt.Sprintf("    State: Running (started: %s)\n", status.State.Running.StartedAt.Format(time.RFC3339)))
		} else if status.State.Waiting != nil {
			output.WriteString(fmt.Sprintf("    State: Waiting (reason: %s)\n", status.State.Waiting.Reason))
		} else if status.State.Terminated != nil {
			output.WriteString(fmt.Sprintf("    State: Terminated (reason: %s, exit code: %d)\n",
				status.State.Terminated.Reason, status.State.Terminated.ExitCode))
		}
	}

	output.WriteString("\nEvents:\n")
	events, err := s.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		log.Printf("failed to get events: %v", err)
	} else {
		for _, event := range events.Items {
			output.WriteString(fmt.Sprintf("  - [%s] %s: %s\n",
				event.LastTimestamp.Format(time.RFC3339),
				event.Reason,
				event.Message))
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}
