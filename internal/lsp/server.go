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

const maxMessageSize = 16 << 20

// Server is a small stdio Language Server Protocol implementation for
// OpenSpec Markdown documents.
type Server struct {
	in    *bufio.Reader
	out   io.Writer
	docs  map[string]string
	index workspaceIndex
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
		var p initializeParams
		_ = json.Unmarshal(req.Params, &p)
		s.setRoot(p)
		return s.respond(req.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync":        map[string]any{"openClose": true, "change": 1},
				"documentSymbolProvider":  true,
				"definitionProvider":      true,
				"referencesProvider":      true,
				"renameProvider":          map[string]any{"prepareProvider": true},
				"codeActionProvider":      true,
				"codeLensProvider":        map[string]any{},
				"documentLinkProvider":    map[string]any{},
				"workspaceSymbolProvider": true,
				"foldingRangeProvider":    true,
				"selectionRangeProvider":  true,
				"inlayHintProvider":       true,
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
		s.indexDirty(p.TextDocument.URI)
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/didChange":
		var p didChangeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		if len(p.ContentChanges) > 0 {
			s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
			s.indexDirty(p.TextDocument.URI)
		}
		return s.publishDiagnostics(p.TextDocument.URI)
	case "textDocument/didClose":
		var p didCloseParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		delete(s.docs, p.TextDocument.URI)
		s.indexDirty(p.TextDocument.URI)
		return s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []diagnostic{}})
	case "workspace/didChangeWatchedFiles":
		var p didChangeWatchedFilesParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil
		}
		s.index.dirty = true
		return nil
	case "textDocument/documentSymbol":
		var p textDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []documentSymbol{})
		}
		return s.respond(req.ID, symbols(s.text(p.TextDocument.URI)))
	case "textDocument/completion":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, completionList{})
		}
		return s.respond(req.ID, completionList{Items: s.completions(p.TextDocument.URI, s.text(p.TextDocument.URI), p.Position)})
	case "textDocument/hover":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, nil)
		}
		return s.respond(req.ID, hover{Contents: markupContent{Kind: "markdown", Value: s.hoverAt(p.TextDocument.URI, s.text(p.TextDocument.URI), p.Position)}})
	case "textDocument/definition":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []location{})
		}
		return s.respond(req.ID, s.definitions(p.TextDocument.URI, p.Position))
	case "textDocument/references":
		var p referenceParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []location{})
		}
		return s.respond(req.ID, s.references(p.TextDocument.URI, p.Position))
	case "textDocument/prepareRename":
		var p textDocumentPositionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, nil)
		}
		return s.respond(req.ID, s.prepareRename(p.TextDocument.URI, p.Position))
	case "textDocument/rename":
		var p renameParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, workspaceEdit{})
		}
		return s.respond(req.ID, s.rename(p.TextDocument.URI, p.Position, p.NewName))
	case "textDocument/codeAction":
		var p codeActionParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []codeAction{})
		}
		return s.respond(req.ID, s.codeActions(p.TextDocument.URI, s.text(p.TextDocument.URI), p.Context.Diagnostics))
	case "textDocument/codeLens":
		var p codeLensParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []codeLens{})
		}
		return s.respond(req.ID, s.codeLens(p.TextDocument.URI))
	case "textDocument/inlayHint":
		var p inlayHintParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []inlayHint{})
		}
		return s.respond(req.ID, s.inlayHints(p.TextDocument.URI, p.Range))
	case "textDocument/documentLink":
		var p textDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []documentLink{})
		}
		return s.respond(req.ID, s.documentLinks(p.TextDocument.URI))
	case "workspace/symbol":
		var p workspaceSymbolParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []workspaceSymbol{})
		}
		return s.respond(req.ID, s.workspaceSymbols(p.Query))
	case "textDocument/foldingRange":
		var p textDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []foldingRange{})
		}
		return s.respond(req.ID, foldingRanges(s.text(p.TextDocument.URI)))
	case "textDocument/selectionRange":
		var p selectionRangeParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return s.respond(req.ID, []selectionRange{})
		}
		return s.respond(req.ID, selectionRanges(s.text(p.TextDocument.URI), p.Positions))
	default:
		if len(req.ID) == 0 {
			return nil
		}
		return s.respondError(req.ID, -32601, "method not found")
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	diags := analyze(uri, s.text(uri))
	diags = append(diags, s.graphDiagnostics(uri)...)
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
	if length < 0 || length > maxMessageSize {
		return nil, fmt.Errorf("content length too large")
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
