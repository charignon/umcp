# UMCP - Universal MCP Bridge

Transform ANY command-line tool into an MCP (Model Context Protocol) server with just a YAML configuration file. No coding required!

## 🚀 Quick Start

```bash
# Build the project
make build

# Validate configurations
make validate-configs

# Run tests
make test

# Run smoke tests
make smoke-test
```

## 📋 Overview

UMCP (Universal MCP Bridge) is a Go-based adapter that automatically transforms command-line tools into MCP servers by reading simple YAML configuration files. This eliminates the need to write custom MCP servers for each CLI tool you want to expose to AI assistants.

### Key Features

- **Zero-Code MCP Servers**: Define your CLI tool's interface in YAML
- **Multiple Output Parsers**: JSON, CSV, XML, Regex, Lines, or Raw
- **Security Sandboxing**: Built-in command validation and path restrictions
- **Flexible Arguments**: Support for flags, positional args, arrays, and conditionals
- **Command Chaining**: Execute multiple commands in sequence
- **Type Safety**: Automatic type conversion and validation

## 🛠️ Installation

### From Source

```bash
git clone https://github.com/laurent/umcp.git
cd umcp
make install  # Installs to /usr/local/bin
```

### Manual Build

```bash
go build -o umcp main.go
```

## 📝 Configuration

### Basic Structure

Create a YAML file describing your CLI tool:

```yaml
version: "1.0"
metadata:
  name: mytool
  description: Description of your tool
  version: 1.0.0

settings:
  command: mytool           # The CLI command to run
  working_dir: "."          # Working directory
  timeout: 30s              # Command timeout
  environment:              # Environment variables
    - VAR_NAME=value

security:
  blocked_commands:         # Commands to block
    - rm
    - sudo
  allowed_paths:           # Restrict file access
    - /home/user/safe
  max_output_size: 10MB    # Limit output size

tools:
  - name: example
    description: Example command
    command: subcommand     # Optional subcommand
    arguments:
      - name: input
        description: Input file
        type: string
        required: true
        flag: "--input"
    output:
      type: json           # Output parser type
```

### Argument Types

```yaml
arguments:
  # String argument
  - name: message
    type: string
    flag: "--message"

  # Boolean flag
  - name: verbose
    type: boolean
    flag: "-v"

  # Integer with validation
  - name: port
    type: integer
    flag: "--port"
    min: 1
    max: 65535

  # Array (repeated flag)
  - name: tags
    type: array
    flag: "--tag"

  # Positional argument
  - name: filename
    type: string
    positional: true
    position: 0

  # Conditional argument
  - name: debug_level
    type: integer
    flag: "--debug-level"
    when: "${debug} == true"
```

### Output Parsers

```yaml
output:
  # Raw output (default)
  type: raw

  # JSON parsing
  type: json
  jq: ".items[]"  # Optional JQ filter

  # Line-by-line array
  type: lines

  # CSV to JSON
  type: csv

  # Regex extraction
  type: regex
  pattern: '^(\w+): (.+)$'
  groups:
    - name: key
      type: string
    - name: value
      type: string
```

## 🔧 Usage

### Command Line

```bash
# Run with single config
umcp --config git.yaml

# Multiple tools
umcp --config git.yaml --config docker.yaml

# Validate configuration
umcp --config myconfig.yaml --validate

# Generate Claude Desktop config
umcp --config git.yaml --generate-claude-config > claude_config.json

# Test mode (validates initialization)
umcp --config git.yaml --test
```

### Claude Desktop Integration

1. Generate the configuration:
```bash
umcp --config configs/git.yaml --generate-claude-config > git_mcp.json
```

2. Add to Claude Desktop config:
```json
{
  "mcpServers": {
    "git": {
      "command": "umcp",
      "args": ["--config", "/path/to/git.yaml"]
    }
  }
}
```

## 📦 Example Configurations

### Git

```yaml
version: "1.0"
metadata:
  name: git
  description: Git version control

settings:
  command: git
  environment:
    - GIT_PAGER=cat

tools:
  - name: status
    description: Show working tree status
    command: status
    arguments:
      - name: short
        type: boolean
        flag: "--short"
    output:
      type: lines

  - name: commit
    description: Commit changes
    command: commit
    arguments:
      - name: message
        type: string
        required: true
        flag: "-m"
      - name: all
        type: boolean
        flag: "--all"
```

### Docker

```yaml
version: "1.0"
metadata:
  name: docker
  description: Docker container management

settings:
  command: docker

tools:
  - name: ps
    description: List containers
    command: ps
    arguments:
      - name: all
        type: boolean
        flag: "--all"
      - name: format
        type: string
        flag: "--format"
    output:
      type: lines

  - name: run
    description: Run container
    command: run
    arguments:
      - name: image
        type: string
        required: true
        positional: true
      - name: detach
        type: boolean
        flag: "-d"
```

## 🏗️ Project Structure

```
umcp/
├── main.go                    # Entry point
├── internal/
│   ├── config/               # YAML config handling
│   │   ├── loader.go
│   │   ├── schema.go
│   │   └── validator.go
│   ├── mcp/                  # MCP protocol implementation
│   │   ├── server.go
│   │   ├── protocol.go
│   │   └── types.go
│   ├── executor/             # Command execution
│   │   ├── executor.go
│   │   ├── builder.go
│   │   └── sandbox.go
│   ├── parser/               # Output parsing
│   │   └── parser.go
│   └── logger/              # Logging utilities
│       └── logger.go
├── configs/                  # Example configurations
│   ├── git.yaml
│   ├── docker.yaml
│   └── ls.yaml
├── test/                     # Test files
│   └── smoke_test.sh
└── Makefile                  # Build automation
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run smoke tests
make smoke-test

# Validate all configs
make validate-configs

# Run specific test
go test ./internal/config -v
```

## 🔒 Security

UMCP includes built-in security features:

- **Command Sandboxing**: Block dangerous commands
- **Path Restrictions**: Limit file system access
- **Output Limits**: Prevent excessive memory usage
- **Input Sanitization**: Prevent command injection
- **Rate Limiting**: Control execution frequency

Configure security in your YAML:

```yaml
security:
  blocked_commands:
    - rm
    - sudo
    - shutdown
  allowed_paths:
    - /home/user/projects
    - /tmp
  max_output_size: 10485760  # 10MB
  rate_limit: "100/minute"
```

## 🎯 Development

```bash
# Development mode with auto-rebuild
make dev

# Format code
make fmt

# Run linter
make lint

# Security scan
make security

# Get dependencies
make deps
```

## 📄 License

MIT License - See LICENSE file for details

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 🐛 Troubleshooting

### Common Issues

**Config validation fails**
- Check YAML syntax
- Ensure required fields are present
- Validate regex patterns

**Command not found**
- Verify the command exists in PATH
- Check command spelling in config

**Permission denied**
- Check file permissions
- Verify allowed_paths configuration

**Output parsing fails**
- Ensure output format matches parser type
- Test regex patterns separately
- Check JSON validity

## 📚 Documentation

For more detailed documentation, see:
- [Configuration Schema](docs/schema.md)
- [Security Guide](docs/security.md)
- [Parser Reference](docs/parsers.md)
- [Examples](configs/)

## 🙏 Acknowledgments

- Inspired by the need to easily integrate CLI tools with Claude Desktop
- Built with the MCP (Model Context Protocol) specification
- Uses excellent Go libraries: zerolog, yaml.v3, testify

## 📞 Support

For issues and questions:
- Open an issue on GitHub
- Check existing issues for solutions
- Read the documentation thoroughly

---

**Made with ❤️ for the AI-assisted development community**