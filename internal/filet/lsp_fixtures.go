package filet

// fakeFullServer responds to the initialize and shutdown handshake and pushes
// one publishDiagnostics payload for a didOpen file. It proves the happy path
// end to end against a scripted, self-contained LSP server.
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

// fakeSilentServer answers initialize and shutdown but never publishes
// diagnostics, so filet's collect timeout has to produce the missing set.
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
