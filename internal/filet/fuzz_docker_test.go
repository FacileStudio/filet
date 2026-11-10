package filet

import (
	"path/filepath"
	"testing"
)

func FuzzParseDockerfile(f *testing.F) {
	f.Add([]byte("FROM golang:latest\nENV DB_PASSWORD=hunter2\nCOPY . .\nRUN apt-get update && apt-get install -y curl\nRUN go build -o app .\nCMD ./app\n"))
	f.Add([]byte("FROM golang:1.26 AS build\nRUN go build \\\n  -o /app .\nFROM scratch\nCOPY --from=build /app /app\n"))
	f.Add([]byte("\r\n# comment\n\\\n\tRUN\\"))
	f.Fuzz(func(t *testing.T, src []byte) {
		prev := 0
		for _, in := range ParseDockerfile(src) {
			if in.Line <= prev {
				t.Fatalf("instruction line %d does not follow %d", in.Line, prev)
			}
			prev = in.Line
		}
		CheckDockerfile(DefaultConfig(), filepath.Join(t.TempDir(), "Dockerfile"), src)
	})
}
