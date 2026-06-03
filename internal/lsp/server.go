package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Server is a small stdio Language Server Protocol implementation for
// OpenSpec Markdown documents.
type Server struct {
	in   *bufio.Reader
	out  io.Writer
	docs map[string]string
}

// NewServer returns a server that reads LSP messages from in and writes
// responses and notifications to out.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: bufio.NewReader(in), out: out, docs: make(map[string]string)}
}

// Run serves JSON-RPC messages until the input ends or an exit notification is
// received.
func (s *Server) Run() error {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req request
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Method == "exit" {
			return nil
		}
		if err := s.handle(req); err != nil {
			return err
		}
	}
}

func (s *Server) handle(req request) error {
	switch req.Method {
	case "initialize":
		return s.respond(req.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync":       map[string]any{"openClose": true, "change": 1},
				"documentSymbolProvider": true,
				"completionProvider": map[string]any{
					"triggerCharacters": []string{"#", "-", ":"},
				},
				"hoverProvider": true,
			},
			"serverInfo": map[string]string{"name": "openspec-lsp"},
		})
	case "shutdown":
		return s.respond(req.ID, nil)
	case "textDocument/didOpen":
		var p didOpenParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		s.docs[p.TextDocument.URI] = p.TextDocument.Text
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/didChange":
		var p didChangeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		if len(p.ContentChanges) > 0 {
			s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
		}
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/didClose":
		var p didCloseParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		delete(s.docs, p.TextDocument.URI)
		return s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []diagnostic{}})
	case "textDocument/documentSymbol":
		var p textDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []documentSymbol{})
		}
		return s.respond(req.ID, symbols(s.docs[p.TextDocument.URI]))
	case "textDocument/completion":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, completionList{})
		}
		return s.respond(req.ID, completionList{Items: completions(p.TextDocument.URI, s.docs[p.TextDocument.URI])})
	case "textDocument/hover":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, nil)
		}
		return s.respond(req.ID, hover{Contents: markupContent{Kind: "markdown", Value: hoverAt(p.TextDocument.URI, s.docs[p.TextDocument.URI], p.Position)}})
	default:
		if len(req.ID) == 0 {
			return nil
		}
		return s.respondError(req.ID, -32601, "method not found")
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	diags := analyze(uri, s.docs[uri])
	if diags == nil {
		diags = []diagnostic{}
	}
	return s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func (s *Server) respond(id json.RawMessage, result any) error {
	return writeMessage(s.out, response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) respondError(id json.RawMessage, code int, msg string) error {
	return writeMessage(s.out, response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: msg}})
}

func (s *Server) notify(method string, params any) error {
	return writeMessage(s.out, notification{JSONRPC: "2.0", Method: method, Params: params})
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	var length int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return nil, err
			}
			length = n
		}
	}
	if length == 0 {
		return nil, fmt.Errorf("missing content length")
	}
	msg := make([]byte, length)
	if _, err := io.ReadFull(r, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func writeMessage(w io.Writer, msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "Content-Length: %d\r\n\r\n", len(b))
	out.Write(b)
	_, err = w.Write(out.Bytes())
	return err
}
