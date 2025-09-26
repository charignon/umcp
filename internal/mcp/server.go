package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charignon/umcp/internal/config"
	"github.com/charignon/umcp/internal/debug"
	"github.com/charignon/umcp/internal/executor"
	"github.com/rs/zerolog/log"
)

// ServerOptions contains options for server configuration
type ServerOptions struct {
	DebugMode   bool
	DebugTrace  string
	ReplayTrace string
}

// Server represents an MCP server instance
type Server struct {
	configs  []*config.Config
	protocol *Protocol
	executor *executor.CommandExecutor
	tools    map[string]*config.Tool
	tracer   *debug.Tracer
}

// NewServer creates a new MCP server
func NewServer(configs []*config.Config, opts ServerOptions) *Server {
	// Setup tracer
	var tracer *debug.Tracer
	var err error

	if opts.ReplayTrace != "" {
		tracer, err = debug.NewReplayTracer(opts.ReplayTrace)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to setup replay tracer")
		}
	} else if opts.DebugMode {
		tracer, err = debug.NewTracer(true, opts.DebugTrace)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to setup debug tracer")
		}
	} else {
		tracer, _ = debug.NewTracer(false, "")
	}

	exec := executor.NewCommandExecutor()
	exec.SetTracer(tracer)

	server := &Server{
		configs:  configs,
		protocol: NewProtocol(os.Stdin, os.Stdout),
		executor: exec,
		tools:    make(map[string]*config.Tool),
		tracer:   tracer,
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

	// Ensure tracer is closed on exit
	defer func() {
		if s.tracer != nil {
			s.tracer.PrintSummary()
			s.tracer.Close()
		}
	}()

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

		// Trace incoming request
		s.tracer.TraceIncoming("request", req, map[string]interface{}{
			"method": req.Method,
			"id":     req.ID,
		})

		if err := s.handleRequest(req); err != nil {
			log.Error().Err(err).Msg("Failed to handle request")

			// Trace error response
			errorResp := map[string]interface{}{
				"id":    req.ID,
				"error": err.Error(),
			}
			s.tracer.TraceOutgoing("error", errorResp, map[string]interface{}{
				"original_method": req.Method,
			})

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
	case "prompts/list":
		return s.handlePromptsList(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "notifications/initialized":
		return s.handleNotificationInitialized(req)
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

	// Trace outgoing response
	s.tracer.TraceOutgoing("response", result, map[string]interface{}{
		"method": "initialize",
		"id":     req.ID,
	})

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

	result := ToolsListResult{Tools: tools}

	// Trace outgoing response
	s.tracer.TraceOutgoing("response", result, map[string]interface{}{
		"method":     "tools/list",
		"id":         req.ID,
		"tool_count": len(tools),
	})

	return s.protocol.SendResult(req.ID, result)
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

	// Trace command execution details
	s.tracer.TraceIncoming("tool_call", params, map[string]interface{}{
		"tool_name": params.Name,
		"config":    toolConfig.Metadata.Name,
	})

	// Execute the command
	output, err := s.executor.Execute(toolConfig, tool, params.Arguments)

	if err != nil {
		result := ToolCallResult{
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("Command failed: %v", err),
			}},
			IsError: true,
		}

		// Trace error result
		s.tracer.TraceOutgoing("tool_error", result, map[string]interface{}{
			"method":    "tools/call",
			"id":        req.ID,
			"tool_name": params.Name,
			"error":     err.Error(),
		})

		return s.protocol.SendResult(req.ID, result)
	}

	result := ToolCallResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output,
		}},
	}

	// Trace successful result
	s.tracer.TraceOutgoing("tool_result", result, map[string]interface{}{
		"method":      "tools/call",
		"id":          req.ID,
		"tool_name":   params.Name,
		"output_size": len(output),
	})

	return s.protocol.SendResult(req.ID, result)
}

// handlePromptsList handles the prompts/list request
func (s *Server) handlePromptsList(req *Request) error {
	// UMCP currently doesn't support prompts, so return empty list
	result := PromptsListResult{
		Prompts: []PromptInfo{},
	}

	// Trace outgoing response
	s.tracer.TraceOutgoing("response", result, map[string]interface{}{
		"method": "prompts/list",
		"id":     req.ID,
		"count":  0,
	})

	return s.protocol.SendResult(req.ID, result)
}

// handleResourcesList handles the resources/list request
func (s *Server) handleResourcesList(req *Request) error {
	// UMCP currently doesn't support resources, so return empty list
	result := ResourcesListResult{
		Resources: []ResourceInfo{},
	}

	// Trace outgoing response
	s.tracer.TraceOutgoing("response", result, map[string]interface{}{
		"method": "resources/list",
		"id":     req.ID,
		"count":  0,
	})

	return s.protocol.SendResult(req.ID, result)
}

// handleNotificationInitialized handles the notifications/initialized notification
func (s *Server) handleNotificationInitialized(req *Request) error {
	// Trace the notification
	s.tracer.TraceIncoming("notification", req, map[string]interface{}{
		"method": "notifications/initialized",
	})

	// Notifications don't require a response - just return nil
	return nil
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