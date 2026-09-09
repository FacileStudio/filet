package filet

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Timeouts bound the language-server conversation so a dead process cannot
// block a check run forever. Vars, not consts, so tests can shorten them.
var (
	// lspTimeout bounds the whole conversation with one server (initialize,
	// shutdown and collect included).
	lspTimeout = 30 * time.Second
	// lspCollectTimeout is how long filet waits for a freshly-opened file's
	// diagnostics before treating the server as silent and moving on.
	lspCollectTimeout = 15 * time.Second
)

type lspPipes struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// lsp is one running language server, reached over JSON-RPC on stdio.
type lsp struct {
	conn    io.WriteCloser
	reader  *bufio.Reader
	cmd     *exec.Cmd
	rootAbs string
	next    int
}

// lspStart spawns the server, completes the initialize handshake and confirms
// the client is ready to receive push diagnostics. dir is rootAbs: cfg.root is
// always absolute. The child is put in its own process group so a hung server
// can be killed along with any helpers it spawned that inherit stdio.
func lspStart(bin string, args []string, dir string) (*lsp, error) {
	cc := exec.Command(bin, args...)
	cc.Dir = dir
	cc.Env = os.Environ()
	cc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	pipes, err := lspOpenPipes(cc)
	if err != nil {
		return nil, err
	}
	if err := cc.Start(); err != nil {
		_ = pipes.stdin.Close()
		_ = pipes.stdout.Close()
		_ = pipes.stderr.Close()
		return nil, err
	}
	go drain(pipes.stderr)
	s := &lsp{conn: pipes.stdin, reader: bufio.NewReader(pipes.stdout), cmd: cc, rootAbs: dir}
	rpcErr, serr := s.init(dir)
	if serr != nil || rpcErr != "" {
		s.close()
		return nil, errors.New("initialize: " + lspErrText(rpcErr, serr))
	}
	_ = lspNotify(s, "initialized", []byte("{}"))
	return s, nil
}

// init opens the LSP conversation, bounded so a server that accepts the
// connection but never answers initialize cannot hang the run.
func (s *lsp) init(rootAbs string) (string, error) {
	var rpcErr string
	serr := s.bounded(lspTimeout, func() error {
		params, inerr := jsonInitParams(rootAbs)
		if inerr != nil {
			return inerr
		}
		var rerr error
		_, rpcErr, rerr = lspRequest(s, "initialize", params)
		return rerr
	})
	return rpcErr, serr
}

// lspOpenPipes opens the child's three stdio pipes.
func lspOpenPipes(cc *exec.Cmd) (lspPipes, error) {
	stdin, err := cc.StdinPipe()
	if err != nil {
		return lspPipes{}, err
	}
	stderr, err := cc.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return lspPipes{}, err
	}
	stdout, err := cc.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stderr.Close()
		return lspPipes{}, err
	}
	return lspPipes{stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// drain copies a child's stderr to nothing so a chatty server cannot block on a
// full stderr pipe while filet drives the conversation.
func drain(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		if _, err := r.Read(buf); err != nil {
			return
		}
	}
}

// bounded runs do synchronously, killing the server if it has not returned by
// deadline. The timeout must kill, never just close stdin: the blocking reads
// are on a separate stdout pipe, so closing the child's write end cannot
// interrupt them — only reaping the process closes its pipes and unblocks the
// reader.
func (s *lsp) bounded(d time.Duration, do func() error) error {
	stop := make(chan struct{})
	go func() {
		select {
		case <-stop:
		case <-time.After(d):
			s.kill()
		}
	}()
	err := do()
	close(stop)
	return err
}

// kill terminates the server process group, so helpers the server spawned are
// reaped too. A helper holding the stdout pipe open would otherwise keep a
// blocking readFrame from ever seeing EOF. Wait is always called once, by close,
// so this only signals — it never reaps.
func (s *lsp) kill() {
	if p := s.cmd.Process; p != nil {
		_ = syscall.Kill(-p.Pid, syscall.SIGKILL)
	}
}

// lspErrText merges a JSON-RPC error object with a transport error into one
// readable string, keeping whichever is the more specific diagnosis.
func lspErrText(rpcErr string, err error) string {
	switch {
	case err != nil && rpcErr != "":
		return rpcErr + ": " + err.Error()
	case err != nil:
		return err.Error()
	default:
		return rpcErr
	}
}

// close shuts the server down gracefully and waits for it to exit, bounded by
// the watchdog so a hung process cannot block the run past lspTimeout.
func (s *lsp) close() {
	s.bounded(lspTimeout, func() error {
		_, _, _ = lspRequest(s, "shutdown", []byte("null"))
		_ = lspNotify(s, "exit", []byte("null"))
		_ = s.conn.Close()
		return s.cmd.Wait()
	})
}
