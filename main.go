package main

import (
	"log"
	"os"

	"github.com/challamani/trace_gateway_failures/pkg/mcp"
)

func main() {
	// Create MCP server instance
	server := mcp.NewServer()

	// Setup stdio transport for MCP communication
	transport := &mcp.StdioTransport{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	// Log to stderr to avoid interfering with MCP protocol on stdout
	log.SetOutput(os.Stderr)

	// Try to initialize Kubernetes client (but continue even if it fails)
	if err := server.InitializeKubeClient(); err != nil {
		log.Printf("Warning: Failed to initialize Kubernetes client: %v", err)
		log.Println("Server will start, but pod log retrieval will not work without valid kubeconfig")
	} else {
		log.Println("Kubernetes client initialized successfully")
	}

	log.Println("MCP Server started - waiting for requests...")

	// Start server
	if err := server.Start(transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
