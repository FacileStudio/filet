package filet

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestLSPFindingSeverityAndLine pins the diagnostic→finding mapping, which is
// the contract an agent depends on: LSP severities 1-2-3 become error-warn-info
// and the 0-based range line becomes a 1-based finding line.
func TestLSPFindingSeverityAndLine(t *testing.T) {
	f := SourceFile{Display: "src/main.go", Ext: ".go"}
	cases := []struct {
		sev  int
		want string
		line int
	}{
		{1, "lsp.error", 1},
		{2, "lsp.warn", 1},
		{3, "lsp.info", 1},
		{4, "lsp.info", 1},
	}
	for _, c := range cases {
		found := lspFinding(f, lspDiagnostic{Range: lspRange{Start: lspPosition{Line: 0}}, Severity: c.sev, Code: "E1", Source: "compiler", Message: "boom"})
		if found.Rule != c.want {
			t.Errorf("severity %d: got rule %q want %q", c.sev, found.Rule, c.want)
		}
		if found.Line != c.line {
			t.Errorf("severity %d: got line %d want 1", c.sev, found.Line)
		}
		if !strings.Contains(found.Message, "E1 compiler: boom") {
			t.Errorf("message not decorated with code+source: %q", found.Message)
		}
		if err := lspFinding(f, lspDiagnostic{Range: lspRange{Start: lspPosition{Line: 7}}}).Line; err != 8 {
			t.Errorf("0-based line 7 should map to finding line 8, got %d", err)
		}
	}
}

// TestLSPFindingCarriesColumnDocsAndTags pins the extra context servers send
// that lspFinding folds in: the start character becomes the finding column, the
// codeDescription href becomes the Docs field (stylable muted by renderers),
// and tag 2 marks code deprecated. The message itself carries no brackets.
func TestLSPFindingCarriesColumnDocsAndTags(t *testing.T) {
	f := SourceFile{Display: "a.go", Ext: ".go"}
	fd := lspFinding(f, lspDiagnostic{
		Range:           lspRange{Start: lspPosition{Line: 2, Character: 7}},
		Severity:        1,
		Code:            "E9",
		Source:          "ts",
		Message:         "bad",
		Tags:            []int{2},
		CodeDescription: &lspCodeDescription{Href: "https://example.com/e9"},
	})
	if fd.Column != 8 {
		t.Errorf("character 7 should map to column 8, got %d", fd.Column)
	}
	if fd.Docs != "https://example.com/e9" {
		t.Errorf("codeDescription href not carried as Docs: %q", fd.Docs)
	}
	if fd.Message != "E9 ts: bad (deprecated)" {
		t.Errorf("message should be bracket-free code+source+tag, got %q", fd.Message)
	}
	if strings.Contains(fd.Message, "[") || strings.Contains(fd.Message, "]") {
		t.Errorf("message must not carry decorative brackets: %q", fd.Message)
	}
	if !strings.Contains(docsSuffix(fd), "https://example.com/e9") {
		t.Errorf("docsSuffix should surface the link: %q", docsSuffix(fd))
	}
	clean := lspFinding(f, lspDiagnostic{Range: lspRange{Start: lspPosition{Line: 4, Character: 1}}, Severity: 3, Message: "hi"})
	if clean.Column != 2 {
		t.Errorf("plain diagnostic should still map column, got %d", clean.Column)
	}
	if clean.Message != "hi" {
		t.Errorf("plain message must stay untouched, got %q", clean.Message)
	}
	if docsSuffix(clean) != "" {
		t.Errorf("a finding with no Docs must have an empty suffix, got %q", docsSuffix(clean))
	}
}

// TestReadFrameContentLength proves the framing reads exactly N bytes past the
// Content-Length header, the part a malformed server is likeliest to break.
func TestReadFrameContentLength(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":3,"result":null}`
	raw := "Content-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload
	br := bufio.NewReader(strings.NewReader(raw))
	body, err := readFrame(br)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if string(body) != payload {
		t.Errorf("frame body mismatch:\n got %s\nwant %s", body, payload)
	}
}

// TestCheckLSPDegradesGracefully proves the never-silent contract without any
// real server: a missing binary is one lsp.unavailable info finding, obeying
// both lsp.fail promotion and the disabled list.
func TestCheckLSPDegradesGracefully(t *testing.T) {
	mk := func(cfg *Config) *Config {
		if cfg.LSP.Servers == nil {
			cfg.LSP.Servers = map[string]Server{}
		}
		cfg.LSP.Servers[".go"] = Server{Server: "/nonexistent/filet-fake-lsp"}
		return cfg
	}
	f := SourceFile{Display: "main.go", Ext: ".go", Lines: []string{"package main"}}

	def := DefaultConfig()
	mk(def)
	out := CheckLSP(def, []SourceFile{f})
	if len(out) != 1 || out[0].Rule != "lsp.unavailable" || out[0].Severity != Info {
		t.Fatalf("expected one info lsp.unavailable, got %+v", out)
	}

	fail := DefaultConfig()
	mk(fail)
	fail.LSP.Fail = true
	if out := CheckLSP(fail, []SourceFile{f}); len(out) != 1 {
		t.Fatalf("fail promotion: want 1 finding, got %d", len(out))
	} else if out[0].Severity != Error {
		t.Fatalf("lsp.fail should promote to error, got %v", out[0].Severity)
	}

	dis := DefaultConfig()
	mk(dis)
	dis.Disabled = []string{"lsp.unavailable"}
	if out := CheckLSP(dis, []SourceFile{f}); len(out) != 0 {
		t.Fatalf("disabled lsp.unavailable should be hidden, got %+v", out)
	}
}

const fakeFullServer = `#!/usr/bin/env bash
send() { printf 'Content-Length: %d\r\n\r\n%s' "${#1}" "$1"; }
LEN=0
while IFS= read -r line; do
  line="${line%$'\r'}"
  if [[ -z "$line" ]]; then
    IFS= read -r -N "$LEN" body
    case "$body" in
      *initialize*) send '{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1}}}' ;;
      *shutdown*) send '{"jsonrpc":"2.0","id":2,"result":null}' ;;
      *didOpen*)
        uri=$(printf '%s' "$body" | sed -n 's/.*"uri":"\([^"]*\)".*/\1/p' | head -1)
        send "{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\",\"params\":{\"uri\":\"$uri\",\"diagnostics\":[{\"range\":{\"start\":{\"line\":2}},\"severity\":1,\"message\":\"boom\"}]}}"
        ;;
    esac
    LEN=0
  else
    [[ "$line" =~ ^Content-Length:[[:space:]]*([0-9]+)$ ]] && LEN="${BASH_REMATCH[1]}"
  fi
done
`

const fakeSilentServer = `#!/usr/bin/env bash
send() { printf 'Content-Length: %d\r\n\r\n%s' "${#1}" "$1"; }
LEN=0
while IFS= read -r line; do
  line="${line%$'\r'}"
  if [[ -z "$line" ]]; then
    IFS= read -r -N "$LEN" body
    case "$body" in
      *initialize*) send '{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}' ;;
      *shutdown*) send '{"jsonrpc":"2.0","id":2,"result":null}' ;;
    esac
    LEN=0
  else
    [[ "$line" =~ ^Content-Length:[[:space:]]*([0-9]+)$ ]] && LEN="${BASH_REMATCH[1]}"
  fi
done
`

func fakeServer(t *testing.T, script string) (bin, root string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, "proj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(dir, "server.sh")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, root
}

func lspFile(rel, body string) SourceFile {
	return SourceFile{Rel: rel, Ext: ".go", Src: []byte(body)}
}

// TestLSPLifecycleDrivesRealServer proves the happy path end to end: spawn,
// collect one pushed diagnostic, graceful close — against a scripted server.
func TestLSPLifecycleDrivesRealServer(t *testing.T) {
	bin, root := fakeServer(t, fakeFullServer)
	s, err := lspStart(bin, nil, root)
	if err != nil {
		t.Fatalf("lspStart failed: %v", err)
	}
	f := lspFile("main.go", "package main\n")
	got, missing := lspCollect(s, []SourceFile{f})
	s.close()
	if len(missing) != 0 {
		t.Fatalf("want no missing files, got %v", missing)
	}
	if diags := got["main.go"]; len(diags) != 1 || diags[0].Message != "boom" {
		t.Fatalf("want 1 'boom' diagnostic, got %+v", diags)
	}
}

// TestLSPStartFailsWhenServerExitsImmediately covers an instant-exit server:
// spawn must return an error, not hang or serve a dead process.
func TestLSPStartFailsWhenServerExitsImmediately(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "exit.sh")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if s, err := lspStart(bin, nil, dir); err == nil {
		s.close()
		t.Fatal("expected an error from a server that exits immediately")
	}
}

// TestLSPCollectBoundedByKill proves a server that accepts didOpen but never
// publishes diagnostics does not hang: after the collect timeout the process is
// killed, the partial (empty) set is returned and every file is listed missing.
func TestLSPCollectBoundedByKill(t *testing.T) {
	bin, root := fakeServer(t, fakeSilentServer)
	s, err := lspStart(bin, nil, root)
	if err != nil {
		t.Fatalf("lspStart failed: %v", err)
	}
	old := lspCollectTimeout
	lspCollectTimeout = 200 * time.Millisecond
	defer func() { lspCollectTimeout = old }()
	got, missing := lspCollect(s, []SourceFile{
		lspFile("a.go", "package p\n"),
		lspFile("b.go", "package p\n"),
	})
	s.close()
	if len(missing) != 2 {
		t.Fatalf("want both files missing, got %v", missing)
	}
	if got == nil {
		t.Fatal("partial diagnostics must be non-nil even when every file is missing")
	}
}

// TestLSPStartBoundedByKill proves a server that stalls on initialize is killed
// after the timeout instead of hanging the spawn.
func TestLSPStartBoundedByKill(t *testing.T) {
	bin, root := fakeServer(t, "#!/usr/bin/env bash\nsleep infinity\n")
	old := lspTimeout
	lspTimeout = 200 * time.Millisecond
	defer func() { lspTimeout = old }()
	start := time.Now()
	if _, err := lspStart(bin, nil, root); err == nil {
		t.Fatal("expected an error from a server that never answers initialize")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("spawn against a stalling server took far longer than the bounded timeout")
	}
}
