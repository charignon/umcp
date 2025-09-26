#!/bin/bash

# Setup script for integrating UMCP with Claude Desktop
# This script helps configure Claude Desktop to use UMCP for various CLI tools

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Claude Desktop config path (macOS)
CLAUDE_CONFIG_DIR="$HOME/Library/Application Support/Claude"
CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"

# UMCP paths
UMCP_DIR="$(cd "$(dirname "$0")" && pwd)"
UMCP_BINARY="$UMCP_DIR/umcp"

echo -e "${BLUE}UMCP Claude Desktop Integration Setup${NC}"
echo "======================================="
echo ""

# Check if UMCP is built
if [ ! -f "$UMCP_BINARY" ]; then
    echo -e "${YELLOW}UMCP binary not found. Building...${NC}"
    make -C "$UMCP_DIR" build
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to build UMCP. Please run 'make build' manually.${NC}"
        exit 1
    fi
fi

# Check if Claude Desktop config directory exists
if [ ! -d "$CLAUDE_CONFIG_DIR" ]; then
    echo -e "${YELLOW}Claude Desktop config directory not found.${NC}"
    echo "Creating directory: $CLAUDE_CONFIG_DIR"
    mkdir -p "$CLAUDE_CONFIG_DIR"
fi

# Function to add MCP server to config
add_mcp_server() {
    local name=$1
    local config_file=$2
    local description=$3

    echo -e "${GREEN}Adding MCP server: ${name}${NC}"
    echo "  Config: $config_file"
    echo "  Description: $description"

    # Generate the JSON entry
    cat <<EOF

To add the ${name} MCP server to Claude Desktop, add this to your
${CLAUDE_CONFIG_FILE}:

{
  "mcpServers": {
    "${name}": {
      "command": "${UMCP_BINARY}",
      "args": ["--config", "${UMCP_DIR}/configs/${config_file}"]
    }
  }
}
EOF
}

# Main menu
echo "Available UMCP configurations:"
echo ""

# Check for existing Claude config
if [ -f "$CLAUDE_CONFIG_FILE" ]; then
    echo -e "${GREEN}Found existing Claude Desktop config${NC}"
    echo ""

    # Backup existing config
    BACKUP_FILE="${CLAUDE_CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$CLAUDE_CONFIG_FILE" "$BACKUP_FILE"
    echo -e "${BLUE}Backed up existing config to:${NC}"
    echo "  $BACKUP_FILE"
    echo ""
fi

# List available configurations
echo "Select configurations to add to Claude Desktop:"
echo ""

configs=(
    "git:git.yaml:Git version control integration"
    "docker:docker.yaml:Docker container management"
    "ls:ls.yaml:File listing utility"
    "flashcards:flashcards.yaml:Spaced repetition flashcard system"
)

selected=()
for i in "${!configs[@]}"; do
    IFS=':' read -r name file desc <<< "${configs[$i]}"
    echo "  $((i+1)). ${name} - ${desc}"
done

echo ""
echo "Enter numbers separated by spaces (e.g., '1 2 4'), or 'all' for all configs:"
read -r selection

if [ "$selection" = "all" ]; then
    selected=("${configs[@]}")
else
    for num in $selection; do
        if [ "$num" -ge 1 ] && [ "$num" -le "${#configs[@]}" ]; then
            selected+=("${configs[$((num-1))]}")
        fi
    done
fi

# Generate the complete config
if [ ${#selected[@]} -gt 0 ]; then
    echo ""
    echo -e "${BLUE}Generating Claude Desktop configuration...${NC}"
    echo ""

    # Start JSON
    echo "{" > "$CLAUDE_CONFIG_FILE.new"
    echo '  "mcpServers": {' >> "$CLAUDE_CONFIG_FILE.new"

    # Add selected servers
    for i in "${!selected[@]}"; do
        IFS=':' read -r name file desc <<< "${selected[$i]}"

        # Add comma for all but last entry
        if [ $i -eq $((${#selected[@]} - 1)) ]; then
            comma=""
        else
            comma=","
        fi

        cat >> "$CLAUDE_CONFIG_FILE.new" <<EOF
    "${name}": {
      "command": "${UMCP_BINARY}",
      "args": ["--config", "${UMCP_DIR}/configs/${file}"]
    }${comma}
EOF
    done

    # Close JSON
    echo "  }" >> "$CLAUDE_CONFIG_FILE.new"
    echo "}" >> "$CLAUDE_CONFIG_FILE.new"

    echo -e "${GREEN}Configuration generated!${NC}"
    echo ""
    echo "Preview of new configuration:"
    echo "------------------------------"
    cat "$CLAUDE_CONFIG_FILE.new"
    echo "------------------------------"
    echo ""

    echo "Do you want to install this configuration? (y/n)"
    read -r confirm

    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        mv "$CLAUDE_CONFIG_FILE.new" "$CLAUDE_CONFIG_FILE"
        echo -e "${GREEN}✓ Configuration installed successfully!${NC}"
        echo ""
        echo "Next steps:"
        echo "1. Restart Claude Desktop"
        echo "2. The MCP servers will be available immediately"
        echo ""
        echo "You can now use commands like:"
        for item in "${selected[@]}"; do
            IFS=':' read -r name file desc <<< "$item"
            case $name in
                git)
                    echo "  - 'Show me the git status'"
                    ;;
                docker)
                    echo "  - 'List all running Docker containers'"
                    ;;
                flashcards)
                    echo "  - 'Quiz me on my Python flashcards'"
                    ;;
                ls)
                    echo "  - 'List files in the current directory'"
                    ;;
            esac
        done
    else
        rm "$CLAUDE_CONFIG_FILE.new"
        echo "Installation cancelled."
    fi
fi

echo ""
echo -e "${BLUE}Testing UMCP servers:${NC}"
echo ""
echo "You can test any configuration with:"
echo "  ${UMCP_BINARY} --config configs/<config>.yaml --test"
echo ""
echo "To validate all configurations:"
echo "  make -C ${UMCP_DIR} validate-configs"
echo ""

# Install UMCP to system if requested
echo "Would you like to install UMCP to /usr/local/bin for system-wide access? (y/n)"
read -r install_system

if [ "$install_system" = "y" ] || [ "$install_system" = "Y" ]; then
    echo "Installing UMCP to /usr/local/bin (may require password)..."
    sudo cp "$UMCP_BINARY" /usr/local/bin/umcp
    sudo chmod +x /usr/local/bin/umcp
    echo -e "${GREEN}✓ UMCP installed to /usr/local/bin/umcp${NC}"

    # Update config to use system binary
    if [ -f "$CLAUDE_CONFIG_FILE" ]; then
        echo "Updating config to use system UMCP binary..."
        sed -i '' "s|${UMCP_BINARY}|/usr/local/bin/umcp|g" "$CLAUDE_CONFIG_FILE"
        echo -e "${GREEN}✓ Config updated to use system binary${NC}"
    fi
fi

echo ""
echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo "For more information, see:"
echo "  - README.md for general usage"
echo "  - FLASHCARDS_EXAMPLE.md for the flashcards integration example"
echo ""
echo "Happy coding with Claude! 🎉"