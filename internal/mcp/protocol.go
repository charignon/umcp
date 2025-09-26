package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

// Protocol handles JSON-RPC 2.0 communication
type Protocol struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewProtocol creates a new protocol handler
func NewProtocol(reader io.Reader, writer io.Writer) *Protocol {
	return &Protocol{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
}

// ReadRequest reads a JSON-RPC request from stdin
func (p *Protocol) ReadRequest() (*Request, error) {
	line, err := p.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read request: %w", err)
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		log.Error().Bytes("data", line).Msg("Failed to parse request")
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	log.Debug().
		Interface("id", req.ID).
		Str("method", req.Method).
		Msg("Received request")

	return &req, nil
}

// SendResponse sends a JSON-RPC response to stdout
func (p *Protocol) SendResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if _, err := p.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	if _, err := p.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	log.Debug().
		Interface("id", resp.ID).
		Bool("hasError", resp.Error != nil).
		Msg("Sent response")

	return nil
}

// SendError sends an error response
func (p *Protocol) SendError(id interface{}, code int, message string, data interface{}) error {
	return p.SendResponse(&Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

// SendResult sends a successful response
func (p *Protocol) SendResult(id interface{}, result interface{}) error {
	return p.SendResponse(&Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// Error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)