package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestScripts(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "openspec-lsp")
	if err := exec.Command("go", "build", "-o", bin, ".").Run(); err != nil {
		t.Fatalf("build openspec-lsp: %v", err)
	}

	cmds := scripttest.DefaultCmds()
	cmds["lsp"] = lspCmd(bin)
	engine := &script.Engine{Cmds: cmds, Conds: scripttest.DefaultConds()}
	scripttest.Test(t, context.Background(), engine, nil, "testdata/*.txt")
}

func lspCmd(bin string) script.Cmd {
	return script.Command(
		script.CmdUsage{Summary: "run openspec-lsp with newline-delimited JSON messages", Args: "file"},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) != 1 {
				return nil, script.ErrUsage
			}
			input, err := jsonLinesToFrames(s.Path(args[0]), map[string]string{"${WORK}": filepath.ToSlash(s.Path("."))})
			if err != nil {
				return nil, err
			}
			return func(*script.State) (string, string, error) {
				cmd := exec.CommandContext(s.Context(), bin)
				cmd.Dir = s.Path(".")
				cmd.Stdin = bytes.NewReader(input)
				out, err := cmd.CombinedOutput()
				if err != nil {
					return "", string(out), err
				}
				decoded, err := decodeFrames(out)
				if err != nil {
					return "", string(out), err
				}
				return decoded, "", nil
			}, nil
		},
	)
}

func jsonLinesToFrames(path string, repl map[string]string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out bytes.Buffer
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for old, new := range repl {
			line = strings.ReplaceAll(line, old, new)
		}
		if !json.Valid([]byte(line)) {
			return nil, fmt.Errorf("%s: invalid json line: %s", path, line)
		}
		fmt.Fprintf(&out, "Content-Length: %d\r\n\r\n%s", len(line), line)
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func decodeFrames(b []byte) (string, error) {
	r := bufio.NewReader(bytes.NewReader(b))
	var out strings.Builder
	for {
		msg, err := readFrame(r)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		out.Write(msg)
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func readFrame(r *bufio.Reader) ([]byte, error) {
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
