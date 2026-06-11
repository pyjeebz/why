package mcpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestServeHandshakeListAndCall(t *testing.T) {
	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"why_dig","arguments":{"path":"a.go","start":4}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"why_dig","arguments":{"path":"missing.go"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"nope"}`,
	}, "\n"))

	dig := func(path string, start, end, depth int) (string, error) {
		if path == "missing.go" {
			return "", errors.New("cannot read missing.go")
		}
		return fmt.Sprintf("trail %s:%d-%d depth %d", path, start, end, depth), nil
	}

	var out bytes.Buffer
	if err := Serve(in, &out, dig); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d responses, want 5 (notification must be skipped):\n%s", len(lines), out.String())
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("response %d is not valid JSON: %s", i, line)
		}
	}

	checks := []struct{ line, want string }{
		{lines[0], `"protocolVersion":"2025-03-26"`},
		{lines[0], `"name":"why"`},
		{lines[1], `"why_dig"`},
		{lines[2], "trail a.go:4-4 depth 8"}, // end and depth defaulted
		{lines[3], `"isError":true`},
		{lines[3], "cannot read missing.go"},
		{lines[4], `-32601`},
	}
	for _, c := range checks {
		if !strings.Contains(c.line, c.want) {
			t.Errorf("response missing %q in: %s", c.want, c.line)
		}
	}
}
