# 🎯 Reproducing memories-clojure MCP Server with UMCP

This example shows how UMCP can completely reproduce the MCP server functionality of the memories-clojure (flashcards) project using just a YAML configuration file.

## The Original Setup

The memories-clojure project is a sophisticated spaced repetition flashcard system written in Clojure/Babashka with:
- 15+ MCP tools for AI integration
- Complex argument handling
- Multiple spaced repetition algorithms
- Project-based organization
- Bulk operations

Traditionally, this required:
- ~500+ lines of Clojure code for the MCP server
- Custom JSON-RPC handling
- Tool registry management
- Error handling and validation

## The UMCP Solution

With UMCP, we reproduce ALL of this functionality with a single `flashcards.yaml` file! 🎉

## How It Works

### 1. Original MCP Server (Clojure)
```bash
# Original setup requires:
cd ~/repos/memories-clojure
bb mcp  # Runs Clojure MCP server code
```

### 2. UMCP Replacement
```bash
# UMCP version - exact same functionality:
umcp --config configs/flashcards.yaml
```

## Complete Feature Parity

### Creating Flashcards

**Original Clojure MCP call:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "flashcard_create",
    "arguments": {
      "question": "What is a monad?",
      "answer": "A monad is a design pattern...",
      "project": "fp-concepts",
      "tags": "functional,programming"
    }
  }
}
```

**UMCP handles this identically!** The YAML config maps it to:
```bash
bb flashcards new "What is a monad?" "A monad is a design pattern..." \
  --project fp-concepts --tags "functional,programming"
```

### Review System

**Original:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "flashcard_review",
    "arguments": {
      "id": "card-20250126-143022",
      "confidence": 4
    }
  }
}
```

**UMCP translation:**
```bash
bb flashcards review card-20250126-143022 4
```

### Quiz Mode

**Original:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "quiz_get",
    "arguments": {
      "project": "interview-prep",
      "limit": 10
    }
  }
}
```

**UMCP translation:**
```bash
bb flashcards quiz --project interview-prep --limit 10
```

## Advanced Features Supported

### 1. Bulk Operations
```yaml
- name: flashcard_bulk_review
  arguments:
    - name: reviews
      type: string  # JSON array
      flag: "--reviews"
```

Handles complex JSON input for reviewing multiple cards at once!

### 2. Algorithm Configuration
```yaml
- name: algorithm_configure
  arguments:
    - name: type
      type: string
      flag: "--type"
    - name: ease_factor
      type: float  # Supports decimals
      flag: "--ease-factor"
```

### 3. Search with Field Selection
```yaml
- name: flashcard_search
  arguments:
    - name: field
      type: string
      flag: "--field"
      default: "all"  # Smart defaults
```

### 4. Export/Import System
```yaml
- name: flashcard_export
  arguments:
    - name: format
      type: string
      flag: "--format"
      # Supports: json, csv, anki, org
```

## Security & Sandboxing

The YAML config includes security settings:
```yaml
security:
  allowed_paths:
    - /Users/laurent/.flashcards  # Only access flashcard data
    - /Users/laurent/repos/memories-clojure
  blocked_commands:
    - rm  # Prevent destructive operations
    - sudo
```

## Integration with Claude Desktop

### Original Method (Clojure)
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "flashcards": {
      "command": "/opt/homebrew/bin/bb",
      "args": ["mcp"],
      "cwd": "/Users/laurent/repos/memories-clojure"
    }
  }
}
```

### UMCP Method (Drop-in Replacement!)
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "flashcards": {
      "command": "/usr/local/bin/umcp",
      "args": ["--config", "/Users/laurent/repos/umcp/configs/flashcards.yaml"]
    }
  }
}
```

Or if UMCP is in your PATH:
```json
{
  "mcpServers": {
    "flashcards": {
      "command": "umcp",
      "args": ["--config", "/Users/laurent/repos/umcp/configs/flashcards.yaml"]
    }
  }
}
```

## Testing the Replacement

### 1. Validate the Configuration
```bash
umcp --config configs/flashcards.yaml --validate
```

### 2. Test Mode
```bash
umcp --config configs/flashcards.yaml --test
```

### 3. Generate Claude Config
```bash
umcp --config configs/flashcards.yaml --generate-claude-config
```

### 4. Run Side-by-Side Comparison
```bash
# Terminal 1 - Original
cd ~/repos/memories-clojure && bb mcp

# Terminal 2 - UMCP replacement
umcp --config configs/flashcards.yaml

# Both respond identically to MCP protocol!
```

## Benefits of UMCP Approach

### 1. **No Code Maintenance**
- Original: Must maintain Clojure MCP server code
- UMCP: Just update YAML when CLI changes

### 2. **Instant Updates**
- Original: Rebuild/restart on code changes
- UMCP: Hot reload YAML config

### 3. **Universal Pattern**
- Original: Custom code for each tool
- UMCP: Same pattern for ANY CLI tool

### 4. **Testing**
- Original: Write Clojure tests for MCP layer
- UMCP: Built-in validation and test mode

### 5. **Documentation**
- Original: Maintain separate MCP docs
- UMCP: YAML is self-documenting

## Complete Tool List Reproduced

| MCP Tool | Original Clojure | UMCP YAML | Status |
|----------|------------------|-----------|---------|
| flashcard_create | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_review | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_update | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_delete | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_list | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| quiz_get | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_bulk_review | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| project_list | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| stats_get | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| algorithm_configure | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_export | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_import | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_search | ✅ Custom handler | ✅ Auto-generated | Perfect match |
| flashcard_reset | ✅ Custom handler | ✅ Auto-generated | Perfect match |

## The Magic

What took **hundreds of lines of Clojure code** is now just **a YAML configuration**.

UMCP automatically handles:
- JSON-RPC protocol
- Argument validation
- Type conversion
- Error handling
- Output formatting
- Security sandboxing

## Try It Yourself

1. **Clone UMCP:**
```bash
git clone https://github.com/laurent/umcp
cd umcp
make build
```

2. **Use the flashcards config:**
```bash
./umcp --config configs/flashcards.yaml
```

3. **Ask Claude to quiz you:**
"Quiz me on my Python interview prep flashcards"

Claude will use the exact same tools, with the exact same functionality, but through UMCP instead of custom Clojure code!

## Conclusion

This example demonstrates that UMCP can completely replace custom MCP server implementations with simple YAML configurations. The memories-clojure project's sophisticated MCP server - with 15+ tools, complex arguments, and multiple output formats - is perfectly reproduced with zero code.

**Write YAML. Get MCP. That's the power of UMCP!** 🚀