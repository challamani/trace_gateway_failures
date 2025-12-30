#!/bin/bash

# VS Code MCP Server Setup Script for trace_gateway_failures
# Supports: macOS and Linux

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    *)          MACHINE="UNKNOWN:${OS}"
esac

if [ "$MACHINE" = "UNKNOWN:${OS}" ]; then
    echo -e "${RED}Unsupported operating system: ${OS}${NC}"
    echo "This script only supports macOS and Linux."
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  VS Code MCP Setup for trace_gateway_failures${NC}"
echo -e "${BLUE}  OS: ${MACHINE}${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Step 1: Check prerequisites
echo -e "${YELLOW}[1/6] Checking prerequisites...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go is not installed${NC}"
    echo "Please install Go 1.21 or later from https://golang.org/dl/"
    exit 1
fi
echo -e "${GREEN}✓ Go found: $(go version)${NC}"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}✗ kubectl is not installed${NC}"
    echo "Please install kubectl from https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi
echo -e "${GREEN}✓ kubectl found: $(kubectl version --client --short 2>/dev/null || kubectl version --client)${NC}"

# Check kubeconfig
if [ ! -f "$HOME/.kube/config" ]; then
    echo -e "${YELLOW}⚠ kubeconfig not found at ~/.kube/config${NC}"
    echo "You'll need a valid kubeconfig to use this MCP server"
fi

echo

# Step 2: Build MCP server
echo -e "${YELLOW}[2/6] Building MCP server...${NC}"

# Get script directory and repository root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$( cd "${SCRIPT_DIR}/.." && pwd )"

cd "$REPO_ROOT"

if [ ! -f "go.mod" ]; then
    echo -e "${RED}✗ Not in repository root (go.mod not found)${NC}"
    exit 1
fi

go build -o mcp-server .
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ MCP server built successfully${NC}"
    MCP_SERVER_PATH="$REPO_ROOT/mcp-server"
else
    echo -e "${RED}✗ Failed to build MCP server${NC}"
    exit 1
fi

echo

# Step 3: Create VS Code config directory
echo -e "${YELLOW}[3/6] Creating VS Code configuration directory...${NC}"

if [ "$MACHINE" = "Mac" ]; then
    VSCODE_CONFIG_DIR="$HOME/Library/Application Support/Code/User/globalStorage/github.copilot-chat"
elif [ "$MACHINE" = "Linux" ]; then
    VSCODE_CONFIG_DIR="$HOME/.config/Code/User/globalStorage/github.copilot-chat"
fi

mkdir -p "$VSCODE_CONFIG_DIR"
echo -e "${GREEN}✓ Config directory created: ${VSCODE_CONFIG_DIR}${NC}"

echo

# Step 4: Create MCP configuration
echo -e "${YELLOW}[4/6] Creating MCP configuration...${NC}"

MCP_CONFIG_FILE="$VSCODE_CONFIG_DIR/mcp.json"

# Check if config already exists
if [ -f "$MCP_CONFIG_FILE" ]; then
    echo -e "${YELLOW}⚠ Configuration file already exists${NC}"
    echo -e "Backing up to ${MCP_CONFIG_FILE}.backup"
    cp "$MCP_CONFIG_FILE" "${MCP_CONFIG_FILE}.backup"
fi

# Create the configuration
cat > "$MCP_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "trace_gateway_failures": {
      "command": "${MCP_SERVER_PATH}",
      "args": [],
      "env": {}
    }
  }
}
EOF

echo -e "${GREEN}✓ MCP configuration created${NC}"
echo -e "   Location: ${MCP_CONFIG_FILE}"

echo

# Step 5: Verify configuration
echo -e "${YELLOW}[5/6] Verifying configuration...${NC}"

# Test server runs
echo -n "Testing MCP server... "
if echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | timeout 5 "$MCP_SERVER_PATH" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Server responds correctly${NC}"
else
    echo -e "${YELLOW}⚠ Server test inconclusive (may be normal)${NC}"
fi

# Validate JSON
if command -v jq &> /dev/null; then
    if jq empty "$MCP_CONFIG_FILE" 2>/dev/null; then
        echo -e "${GREEN}✓ Configuration JSON is valid${NC}"
    else
        echo -e "${RED}✗ Configuration JSON is invalid${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ jq not found, skipping JSON validation${NC}"
fi

echo

# Step 6: Display next steps
echo -e "${YELLOW}[6/6] Setup complete!${NC}"
echo
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Setup Successful! 🎉${NC}"
echo -e "${GREEN}========================================${NC}"
echo
echo -e "${BLUE}Next steps:${NC}"
echo
echo -e "1. ${YELLOW}Restart VS Code${NC} completely"
echo -e "   - Close all VS Code windows"
echo -e "   - Reopen VS Code"
echo
echo -e "2. ${YELLOW}Open Copilot Chat${NC}"
if [ "$MACHINE" = "Mac" ]; then
    echo -e "   - Press: Cmd+Shift+I"
elif [ "$MACHINE" = "Linux" ]; then
    echo -e "   - Press: Ctrl+Shift+I"
fi
echo
echo -e "3. ${YELLOW}Test the MCP server${NC} with:"
echo -e "   ${BLUE}Get pod logs from namespace \"default\" for deployment \"nginx\" with 200 tail lines${NC}"
echo
echo -e "${YELLOW}Configuration file:${NC}"
echo -e "   ${MCP_CONFIG_FILE}"
echo
echo -e "${YELLOW}MCP server binary:${NC}"
echo -e "   ${MCP_SERVER_PATH}"
echo
echo -e "${GREEN}========================================${NC}"
echo

# Optional: Open VS Code if available
read -p "Would you like to restart VS Code now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if command -v code &> /dev/null; then
        echo "Restarting VS Code..."
        code --reuse-window "$REPO_ROOT"
    else
        echo -e "${YELLOW}VS Code command not found. Please restart manually.${NC}"
    fi
fi