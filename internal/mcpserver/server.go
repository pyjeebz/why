// Package mcpserver is a minimal MCP stdio server exposing dig to
// agents: temporal context for code they are about to change. One
// JSON-RPC 2.0 message per line, no external dependencies.
package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Version is reported to clients during the MCP handshake.
const Version = "0.1.0"

// DigFunc runs a dig and renders the trail; wired up by the cli.
type DigFunc func(path string, start, end, depth int) (string, error)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var toolDef = map[string]any{
	"name": "why_dig",
	"description": "Recover why a region of code is the way it is: the commits, " +
		"pull requests, issues, and human notes that shaped it, newest first. " +
		"Use before changing a constant, threshold, or guard you don't fully " +
		"understand — the code shows what it says; why_dig shows how it got that way.",
	"inputSchema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "file path, relative to the server's working directory"},
			"start": map[string]any{"type": "integer", "description": "first line of the region (1-based; omit for the whole file)"},
			"end":   map[string]any{"type": "integer", "description": "last line of the region (defaults to start)"},
			"depth": map[string]any{"type": "integer", "description": "maximum hops back through history (default 8)"},
		},
		"required": []any{"path"},
	},
}

// Serve answers MCP requests from r on w until EOF. Notifications and
// unparseable lines are skipped; only requests carrying an id get a
// response, per JSON-RPC 2.0.
func Serve(r io.Reader, w io.Writer, dig DigFunc) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	enc := json.NewEncoder(w)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil || req.ID == nil {
			continue
		}
		if err := enc.Encode(handle(req, dig)); err != nil {
			return err
		}
	}
	return sc.Err()
}

func handle(req request, dig DigFunc) response {
	resp := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = "2025-03-26"
		}
		resp.Result = map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "why", "version": Version},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": []any{toolDef}}
	case "tools/call":
		resp.Result = call(req.Params, dig)
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

// call runs the why_dig tool. Tool failures are reported inside the
// result with isError, not as protocol errors, per MCP convention.
func call(params json.RawMessage, dig DigFunc) map[string]any {
	var p struct {
		Name      string `json:"name"`
		Arguments struct {
			Path  string `json:"path"`
			Start int    `json:"start"`
			End   int    `json:"end"`
			Depth int    `json:"depth"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return errResult(fmt.Sprintf("bad tool call params: %v", err))
	}
	if p.Name != "why_dig" {
		return errResult("unknown tool: " + p.Name)
	}

	a := p.Arguments
	if a.End == 0 {
		a.End = a.Start
	}
	if a.Depth <= 0 {
		a.Depth = 8
	}

	text, err := dig(a.Path, a.Start, a.End, a.Depth)
	if err != nil {
		return errResult(err.Error())
	}
	return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
}

func errResult(msg string) map[string]any {
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": msg}},
		"isError": true,
	}
}
