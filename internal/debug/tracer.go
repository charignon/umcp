package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// TraceEvent represents a single trace event
type TraceEvent struct {
	Timestamp  time.Time   `json:"timestamp"`
	Direction  string      `json:"direction"` // "in" or "out"
	Type       string      `json:"type"`      // "request", "response", "command", "output"
	Data       interface{} `json:"data"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Tracer handles debug tracing and replay
type Tracer struct {
	enabled    bool
	traceFile  *os.File
	events     []TraceEvent
	replayMode bool
	replayIdx  int
}

// NewTracer creates a new debug tracer
func NewTracer(enabled bool, traceFile string) (*Tracer, error) {
	tracer := &Tracer{
		enabled: enabled,
		events:  make([]TraceEvent, 0),
	}

	if enabled && traceFile != "" {
		file, err := os.Create(traceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create trace file: %w", err)
		}
		tracer.traceFile = file
		log.Info().Str("file", traceFile).Msg("Debug tracing enabled")
	}

	return tracer, nil
}

// NewReplayTracer creates a tracer for replay mode
func NewReplayTracer(replayFile string) (*Tracer, error) {
	data, err := os.ReadFile(replayFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read replay file: %w", err)
	}

	var events []TraceEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse replay file: %w", err)
	}

	log.Info().Str("file", replayFile).Int("events", len(events)).Msg("Replay mode enabled")

	return &Tracer{
		enabled:    true,
		replayMode: true,
		events:     events,
		replayIdx:  0,
	}, nil
}

// TraceIncoming logs an incoming message
func (t *Tracer) TraceIncoming(msgType string, data interface{}, metadata map[string]interface{}) {
	if !t.enabled {
		return
	}

	event := TraceEvent{
		Timestamp: time.Now(),
		Direction: "in",
		Type:      msgType,
		Data:      data,
		Metadata:  metadata,
	}

	t.addEvent(event)
	log.Debug().
		Str("direction", "in").
		Str("type", msgType).
		Interface("data", data).
		Msg("TRACE: Incoming")
}

// TraceOutgoing logs an outgoing message
func (t *Tracer) TraceOutgoing(msgType string, data interface{}, metadata map[string]interface{}) {
	if !t.enabled {
		return
	}

	event := TraceEvent{
		Timestamp: time.Now(),
		Direction: "out",
		Type:      msgType,
		Data:      data,
		Metadata:  metadata,
	}

	t.addEvent(event)
	log.Debug().
		Str("direction", "out").
		Str("type", msgType).
		Interface("data", data).
		Msg("TRACE: Outgoing")
}

// TraceCommand logs a command execution
func (t *Tracer) TraceCommand(command string, args []string, workingDir string, env []string) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"command":     command,
		"args":        args,
		"working_dir": workingDir,
		"env":         env,
	}

	event := TraceEvent{
		Timestamp: time.Now(),
		Direction: "internal",
		Type:      "command",
		Data:      fmt.Sprintf("%s %v", command, args),
		Metadata:  metadata,
	}

	t.addEvent(event)
	log.Debug().
		Str("command", command).
		Interface("args", args).
		Str("working_dir", workingDir).
		Msg("TRACE: Command execution")
}

// TraceCommandOutput logs command output
func (t *Tracer) TraceCommandOutput(output string, exitCode int, err error) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"exit_code": exitCode,
		"success":   err == nil,
	}

	if err != nil {
		metadata["error"] = err.Error()
	}

	event := TraceEvent{
		Timestamp: time.Now(),
		Direction: "internal",
		Type:      "output",
		Data:      output,
		Metadata:  metadata,
	}

	t.addEvent(event)
	log.Debug().
		Str("output", output).
		Int("exit_code", exitCode).
		Bool("success", err == nil).
		Msg("TRACE: Command output")
}

// GetNextReplayEvent returns the next event in replay mode
func (t *Tracer) GetNextReplayEvent() (*TraceEvent, bool) {
	if !t.replayMode || t.replayIdx >= len(t.events) {
		return nil, false
	}

	event := &t.events[t.replayIdx]
	t.replayIdx++
	return event, true
}

// IsReplayMode returns true if in replay mode
func (t *Tracer) IsReplayMode() bool {
	return t.replayMode
}

// addEvent adds an event to the trace
func (t *Tracer) addEvent(event TraceEvent) {
	if t.replayMode {
		return // Don't add events in replay mode
	}

	t.events = append(t.events, event)

	// Write to file if enabled
	if t.traceFile != nil {
		if data, err := json.Marshal(event); err == nil {
			fmt.Fprintf(t.traceFile, "%s\n", data)
			t.traceFile.Sync()
		}
	}
}

// Close closes the tracer and writes final trace file
func (t *Tracer) Close() error {
	if t.traceFile != nil {
		// Write all events as a JSON array
		if data, err := json.MarshalIndent(t.events, "", "  "); err == nil {
			t.traceFile.Seek(0, 0)
			t.traceFile.Truncate(0)
			t.traceFile.Write(data)
		}
		return t.traceFile.Close()
	}
	return nil
}

// PrintSummary prints a summary of the trace
func (t *Tracer) PrintSummary() {
	if !t.enabled {
		return
	}

	inCount := 0
	outCount := 0
	cmdCount := 0

	for _, event := range t.events {
		switch event.Direction {
		case "in":
			inCount++
		case "out":
			outCount++
		case "internal":
			if event.Type == "command" {
				cmdCount++
			}
		}
	}

	log.Info().
		Int("total_events", len(t.events)).
		Int("incoming", inCount).
		Int("outgoing", outCount).
		Int("commands", cmdCount).
		Msg("Debug trace summary")
}