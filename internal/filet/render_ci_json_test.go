package filet

import (
	"bytes"
	"testing"
)

var (
	jsonGoldenWant = `{
  "findings": [
    {
      "rule": "go.doc.missing",
      "file": "apps/api/main.go",
      "line": 12,
      "message": "exported Run has no doc comment",
      "severity": "warn"
    },
    {
      "rule": "docker.user.root",
      "file": "Dockerfile",
      "line": 0,
      "message": "no USER instruction",
      "severity": "error"
    },
    {
      "rule": "go.doc.missing",
      "file": "apps/api/run.go",
      "line": 3,
      "message": "exported Do has no doc comment",
      "severity": "warn"
    }
  ],
  "files": 3,
  "lines": 400,
  "counts": {
    "error": 1,
    "warn": 2,
    "info": 0
  },
  "grade": "B"
}
`

	jsonEmptyWant = `{
  "findings": [],
  "files": 0,
  "lines": 0,
  "counts": {
    "error": 0,
    "warn": 0,
    "info": 0
  },
  "grade": "A"
}
`
)

func TestJSONGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, goldenReport()); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != jsonGoldenWant {
		t.Errorf("JSON output =\n%s\nwant\n%s", got, jsonGoldenWant)
	}
}

func TestJSONEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, Report{}); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != jsonEmptyWant {
		t.Errorf("empty JSON output =\n%s\nwant\n%s", got, jsonEmptyWant)
	}
}
