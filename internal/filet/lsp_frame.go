package filet

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// lspErr is the JSON-RPC error object a server returns in a failed response.
type lspErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// readFrame parses one JSON-RPC message body, honouring Content-Length.
func readFrame(r *bufio.Reader) ([]byte, error) {
	length, err := frameLength(r)
	if err != nil {
		return nil, err
	}
	body := make([]byte, length)
	for wrote := 0; wrote < length; {
		n, err := r.Read(body[wrote:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("lsp: EOF inside message body")
		}
		wrote += n
	}
	return body, nil
}

// frameLength reads headers until the blank line and returns Content-Length.
func frameLength(r *bufio.Reader) (int, error) {
	length := 0
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return 0, err
		}
		if len(line) == 0 {
			break
		}
		if n, ok := parseLength(line); ok {
			length = n
		}
	}
	if length <= 0 {
		return 0, errors.New("lsp: no Content-Length header")
	}
	return length, nil
}

// parseLength reads Content-Length out of a header line, if present.
func parseLength(line []byte) (int, bool) {
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(line)), "Content-Length:%d", &n); err == nil {
		return n, true
	}
	return 0, false
}

// lspSend frames a raw payload as one JSON-RPC message and writes it.
func lspSend(s *lsp, payload []byte) error {
	head := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := s.conn.Write([]byte(head)); err != nil {
		return err
	}
	_, err := s.conn.Write(payload)
	return err
}

// lspRequest sends one request and waits for that response, skipping
// interleaved notifications. rpcErr is non-empty when the server answered with
// an error object; err is a transport failure.
func lspRequest(s *lsp, method string, params []byte) ([]byte, string, error) {
	s.next++
	msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"%s","params":%s}`, s.next, method, params)
	if err := lspSend(s, []byte(msg)); err != nil {
		return nil, "", err
	}
	for {
		body, err := readFrame(s.reader)
		if err != nil {
			return nil, "", err
		}
		id, rerr, ok := decodeHead(body)
		if !ok || id != s.next {
			continue
		}
		if rerr == nil {
			return body, "", nil
		}
		return body, rpcMessage(rerr), nil
	}
}

// lspNotify sends a notification that expects no reply.
func lspNotify(s *lsp, method string, params []byte) error {
	msg := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":%s}`, method, params)
	return lspSend(s, []byte(msg))
}

// decodeHead pulls the id and optional error object out of a response frame.
func decodeHead(body []byte) (int, *lspErr, bool) {
	var head struct {
		ID    int     `json:"id"`
		Error *lspErr `json:"error"`
	}
	if json.Unmarshal(body, &head) != nil {
		return 0, nil, false
	}
	return head.ID, head.Error, true
}

// rpcMessage returns the error object's message, with a default when a server
// omits it (a non-nil error object is a failure regardless of its message).
func rpcMessage(e *lspErr) string {
	if e.Message != "" {
		return e.Message
	}
	return "rpc error"
}
