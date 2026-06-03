package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestServerPublishesDiagnostics(t *testing.T) {
	input := lspMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		lspMessage(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///repo/openspec/extensions/journey/login.md","languageId":"markdown","version":1,"text":"# Journey\n\n## Stages\n"}}}`) +
		lspMessage(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`) +
		lspMessage(`{"jsonrpc":"2.0","method":"exit"}`)
	var out bytes.Buffer
	if err := NewServer(strings.NewReader(input), &out).Run(); err != nil {
		t.Fatal(err)
	}
	messages := decodeOutput(t, out.String())
	if len(messages) < 3 {
		t.Fatalf("got %d messages, want at least 3: %s", len(messages), out.String())
	}
	found := false
	for _, msg := range messages {
		if msg["method"] == "textDocument/publishDiagnostics" {
			found = true
			params := msg["params"].(map[string]any)
			diags := params["diagnostics"].([]any)
			if len(diags) != 2 {
				t.Fatalf("diagnostics = %d, want 2: %+v", len(diags), diags)
			}
		}
	}
	if !found {
		t.Fatalf("publishDiagnostics not found: %+v", messages)
	}
}

func TestReadMessageRejectsLargeContentLength(t *testing.T) {
	_, err := readMessage(bufio.NewReader(strings.NewReader("Content-Length: 16777217\r\n\r\n")))
	if err == nil {
		t.Fatal("readMessage succeeded, want error")
	}
	if got, want := err.Error(), "content length too large"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func lspMessage(s string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(s), s)
}

func decodeOutput(t *testing.T, out string) []map[string]any {
	t.Helper()
	var messages []map[string]any
	r := strings.NewReader(out)
	for r.Len() > 0 {
		buf := make([]byte, r.Len())
		n, _ := r.Read(buf)
		chunks := strings.Split(string(buf[:n]), "Content-Length: ")
		for _, chunk := range chunks {
			if chunk == "" {
				continue
			}
			_, body, ok := strings.Cut(chunk, "\r\n\r\n")
			if !ok {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal([]byte(body), &msg); err != nil {
				t.Fatalf("unmarshal %q: %v", body, err)
			}
			messages = append(messages, msg)
		}
	}
	return messages
}
