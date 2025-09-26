package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charignon/umcp/internal/config"
	"github.com/charignon/umcp/internal/executor"
	"github.com/rs/zerolog/log"
)

// Server represents an MCP server instance
type Server struct {
	configs  []*config.Config
	protocol *Protocol
	executor *executor.CommandExecutor
	tools    map[string]*config.Tool
}

// NewServer creates a new MCP server
func NewServer(configs []*config.Config) *Server {
	server := &Server{
		configs:  configs,
		protocol: NewProtocol(os.Stdin, os.Stdout),
		executor: executor.NewCommandExecutor(),
		tools:    make(map[string]*config.Tool),
	}

	// Index all tools
	for _, cfg := range configs {
		for i := range cfg.Tools {
			tool := &cfg.Tools[i]
			fullName := fmt.Sprintf("%s_%s", cfg.Metadata.Name, tool.Name)
			server.tools[fullName] = tool
		}
	}

	return server
}

// Run starts the MCP server
func (s *Server) Run() error {
	log.Info().Msg("MCP server started")

	for {
		req, err := s.protocol.ReadRequest()
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Client disconnected")
				return nil
			}
			log.Error().Err(err).Msg("Failed to read request")
			continue
		}

		if err := s.handleRequest(req); err != nil {
			log.Error().Err(err).Msg("Failed to handle request")
			s.protocol.SendError(req.ID, InternalError, err.Error(), nil)
		}
	}
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req *Request) error {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return s.protocol.SendError(req.ID, MethodNotFound,
			fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) error {
	var params InitializeParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.protocol.SendError(req.ID, InvalidParams, "Invalid parameters", err.Error())
		}
	}

	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "umcp",
			Version: "1.0.0",
		},
	}

	return s.protocol.SendResult(req.ID, result)
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req *Request) error {
	tools := make([]ToolInfo, 0, len(s.tools))

	for _, cfg := range s.configs {
		for _, tool := range cfg.Tools {
			fullName := fmt.Sprintf("%s_%s", cfg.Metadata.Name, tool.Name)

			// Build input schema
			properties := make(map[string]Property)
			required := []string{}

			for _, arg := range tool.Arguments {
				prop := Property{
					Type:        s.mapArgTypeToJSONSchema(arg.Type),
					Description: arg.Description,
					Default:     arg.Default,
					Minimum:     arg.Min,
					Maximum:     arg.Max,
				}

				if arg.Type == "array" {
					prop.Items = &Property{Type: "string"}
				}

				properties[arg.Name] = prop

				if arg.Required {
					required = append(required, arg.Name)
				}
			}

			tools = append(tools, ToolInfo{
				Name:        fullName,
				Description: tool.Description,
				InputSchema: InputSchema{
					Type:       "object",
					Properties: properties,
					Required:   required,
				},
			})
		}
	}

	return s.protocol.SendResult(req.ID, ToolsListResult{Tools: tools})
}

// handleToolCall handles the tools/call request
func (s *Server) handleToolCall(req *Request) error {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.protocol.SendError(req.ID, InvalidParams, "Invalid parameters", err.Error())
	}

	tool, exists := s.tools[params.Name]
	if !exists {
		return s.protocol.SendError(req.ID, InvalidParams,
			fmt.Sprintf("Tool not found: %s", params.Name), nil)
	}

	// Find the config for this tool
	var toolConfig *config.Config
	for _, cfg := range s.configs {
		for _, t := range cfg.Tools {
			if fmt.Sprintf("%s_%s", cfg.Metadata.Name, t.Name) == params.Name {
				toolConfig = cfg
				break
			}
		}
		if toolConfig != nil {
			break
		}
	}

	if toolConfig == nil {
		return s.protocol.SendError(req.ID, InternalError, "Configuration not found", nil)
	}

	// Execute the command
	output, err := s.executor.Execute(toolConfig, tool, params.Arguments)
	if err != nil {
		return s.protocol.SendResult(req.ID, ToolCallResult{
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("Command failed: %v", err),
			}},
			IsError: true,
		})
	}

	return s.protocol.SendResult(req.ID, ToolCallResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	})
}

// mapArgTypeToJSONSchema maps argument types to JSON Schema types
func (s *Server) mapArgTypeToJSONSchema(argType string) string {
	switch argType {
	case "boolean":
		return "boolean"
	case "integer":
		return "integer"
	case "float":
		return "number"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string"
	}
}