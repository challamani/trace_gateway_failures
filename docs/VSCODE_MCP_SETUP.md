# VS Code MCP Setup Guide

This guide provides instructions for setting up Model Context Protocol (MCP) servers in Visual Studio Code on macOS and Linux systems.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Setup (Automated)](#quick-setup-automated)
- [Manual Setup](#manual-setup)
- [Configuration](#configuration)
- [Available MCP Servers](#available-mcp-servers)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before setting up MCP in VS Code, ensure you have:

- **Visual Studio Code** (version 1.85.0 or higher)
- **Node.js** (version 18.x or higher)
- **npm** (usually comes with Node.js)
- **Git** (for cloning repositories)
- Internet connection for downloading packages

### Installing Prerequisites

#### macOS

```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Node.js and npm
brew install node

# Install Git
brew install git
```

#### Linux (Ubuntu/Debian)

```bash
# Update package list
sudo apt update

# Install Node.js and npm
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Install Git
sudo apt install -y git
```

## Quick Setup (Automated)

We provide an automated setup script that configures MCP servers for VS Code on both macOS and Linux.

### Download and Run the Setup Script

```bash
# Download the setup script
curl -fsSL https://raw.githubusercontent.com/challamani/trace_gateway_failures/main/scripts/setup_vscode_mcp.sh -o setup_vscode_mcp.sh

# Make it executable
chmod +x setup_vscode_mcp.sh

# Run the setup script
./setup_vscode_mcp.sh
```

### What the Script Does

The automated setup script will:

1. ✅ Check for required prerequisites (Node.js, npm, VS Code)
2. ✅ Create the MCP configuration directory (`~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/`)
3. ✅ Install commonly used MCP servers:
   - `@modelcontextprotocol/server-filesystem` - File system operations
   - `@modelcontextprotocol/server-github` - GitHub integration
   - `@modelcontextprotocol/server-memory` - Persistent memory
   - `@modelcontextprotocol/server-brave-search` - Web search capabilities
4. ✅ Generate the `cline_mcp_settings.json` configuration file
5. ✅ Verify the installation
6. ✅ Provide next steps

### Script Options

```bash
# Run with custom configuration directory
./setup_vscode_mcp.sh --config-dir /custom/path

# Run in verbose mode
./setup_vscode_mcp.sh --verbose

# Skip prerequisite checks (not recommended)
./setup_vscode_mcp.sh --skip-checks

# Display help
./setup_vscode_mcp.sh --help
```

## Manual Setup

If you prefer to set up MCP servers manually, follow these steps:

### Step 1: Create Configuration Directory

```bash
# Create the MCP settings directory
mkdir -p ~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/
```

### Step 2: Install MCP Servers

```bash
# Install filesystem server
npm install -g @modelcontextprotocol/server-filesystem

# Install GitHub server
npm install -g @modelcontextprotocol/server-github

# Install memory server
npm install -g @modelcontextprotocol/server-memory

# Install brave-search server (optional)
npm install -g @modelcontextprotocol/server-brave-search
```

### Step 3: Create Configuration File

Create a file at `~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "node",
      "args": [
        "/usr/local/lib/node_modules/@modelcontextprotocol/server-filesystem/dist/index.js"
      ],
      "env": {
        "ALLOWED_DIRECTORIES": "/Users/YOUR_USERNAME/projects,/tmp"
      }
    },
    "github": {
      "command": "node",
      "args": [
        "/usr/local/lib/node_modules/@modelcontextprotocol/server-github/dist/index.js"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "your_github_token_here"
      }
    },
    "memory": {
      "command": "node",
      "args": [
        "/usr/local/lib/node_modules/@modelcontextprotocol/server-memory/dist/index.js"
      ]
    }
  }
}
```

**Note:** Replace `/usr/local/lib/node_modules` with the actual path where npm installs global packages. You can find this by running:

```bash
npm root -g
```

## Configuration

### Finding npm Global Directory

Different systems install npm packages in different locations:

```bash
# Find npm global directory
npm root -g

# Common locations:
# macOS: /usr/local/lib/node_modules
# Linux: /usr/lib/node_modules or /usr/local/lib/node_modules
```

### Configuring GitHub Access

To use the GitHub MCP server, you need a personal access token:

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select scopes: `repo`, `read:org`, `read:user`
4. Copy the token and add it to your configuration file

### Configuring File System Access

Update the `ALLOWED_DIRECTORIES` environment variable to include paths you want to allow:

```json
"env": {
  "ALLOWED_DIRECTORIES": "/home/username/projects,/home/username/documents,/tmp"
}
```

### Configuring Brave Search

To use the Brave Search MCP server, you need an API key:

1. Sign up at [Brave Search API](https://brave.com/search/api/)
2. Get your API key
3. Add to configuration:

```json
"brave-search": {
  "command": "node",
  "args": [
    "/usr/local/lib/node_modules/@modelcontextprotocol/server-brave-search/dist/index.js"
  ],
  "env": {
    "BRAVE_API_KEY": "your_brave_api_key_here"
  }
}
```

## Available MCP Servers

### Core Servers

| Server | Description | Installation |
|--------|-------------|--------------|
| `@modelcontextprotocol/server-filesystem` | File system operations | `npm install -g @modelcontextprotocol/server-filesystem` |
| `@modelcontextprotocol/server-github` | GitHub repository access | `npm install -g @modelcontextprotocol/server-github` |
| `@modelcontextprotocol/server-memory` | Persistent memory storage | `npm install -g @modelcontextprotocol/server-memory` |
| `@modelcontextprotocol/server-brave-search` | Web search via Brave | `npm install -g @modelcontextprotocol/server-brave-search` |

### Additional Servers

Explore more MCP servers at the [MCP Servers Registry](https://github.com/modelcontextprotocol/servers).

## Troubleshooting

### VS Code Can't Find MCP Servers

**Problem:** VS Code reports that it cannot find the MCP server executables.

**Solution:**
1. Verify the installation path using `npm root -g`
2. Update the paths in `cline_mcp_settings.json`
3. Ensure the server files are executable: `chmod +x /path/to/server/index.js`

### Permission Denied Errors

**Problem:** Getting permission denied errors when accessing files.

**Solution:**
1. Check the `ALLOWED_DIRECTORIES` in your configuration
2. Ensure the paths exist and are accessible
3. Verify file permissions: `ls -la /path/to/directory`

### GitHub Authentication Fails

**Problem:** GitHub MCP server reports authentication errors.

**Solution:**
1. Verify your token has the correct scopes (`repo`, `read:org`, `read:user`)
2. Check that the token hasn't expired
3. Ensure the token is correctly set in the `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable

### Node.js Version Issues

**Problem:** MCP servers fail to start due to Node.js version incompatibility.

**Solution:**
```bash
# Check your Node.js version
node --version

# Should be 18.x or higher. If not, update Node.js:
# macOS
brew upgrade node

# Linux
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs
```

### Configuration File Not Loaded

**Problem:** VS Code doesn't recognize the MCP configuration.

**Solution:**
1. Verify the configuration file is at the correct path:
   ```bash
   ls -la ~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json
   ```
2. Check JSON syntax: `cat cline_mcp_settings.json | jq .` (requires jq)
3. Restart VS Code completely

### Checking Logs

To debug issues, check the MCP server logs:

```bash
# VS Code logs location
~/.config/Code/logs/

# Look for MCP-related log entries
grep -r "MCP" ~/.config/Code/logs/
```

## Verifying Installation

After setup, verify your MCP configuration:

```bash
# Check if configuration file exists
ls -la ~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json

# Validate JSON syntax
cat ~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json | python3 -m json.tool

# Check installed MCP servers
npm list -g --depth=0 | grep modelcontextprotocol
```

## Next Steps

After successful setup:

1. 🚀 Restart Visual Studio Code
2. 📝 Open a project and test MCP server functionality
3. 🔧 Customize your MCP configuration as needed
4. 📚 Explore additional MCP servers from the registry

## Resources

- [MCP Official Documentation](https://modelcontextprotocol.io/)
- [MCP GitHub Repository](https://github.com/modelcontextprotocol)
- [VS Code Documentation](https://code.visualstudio.com/docs)

## Contributing

Found an issue or have suggestions for improving this guide? Please open an issue or submit a pull request to the repository.

---

**Last Updated:** 2025-12-30  
**Maintained by:** challamani  
**Repository:** trace_gateway_failures
