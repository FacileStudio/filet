package filet

import (
	"bytes"
	"testing"
)

var (
	sarifGoldenWant = `{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "filet",
          "version": "0.2.0",
          "informationUri": "https://github.com/FacileStudio/filet",
          "rules": [
            {
              "id": "docker.user.root",
              "name": "docker.user.root",
              "shortDescription": {
                "text": "container ends up running as root"
              }
            },
            {
              "id": "go.doc.missing",
              "name": "go.doc.missing",
              "shortDescription": {
                "text": "exported declaration without a doc comment (directive-only groups do not count)"
              }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "go.doc.missing",
          "ruleIndex": 1,
          "level": "warning",
          "message": {
            "text": "exported Run has no doc comment"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "apps/api/main.go"
                },
                "region": {
                  "startLine": 12,
                  "startColumn": 1
                }
              }
            }
          ]
        },
        {
          "ruleId": "docker.user.root",
          "ruleIndex": 0,
          "level": "error",
          "message": {
            "text": "no USER instruction"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "Dockerfile"
                },
                "region": {
                  "startLine": 1,
                  "startColumn": 1
                }
              }
            }
          ]
        },
        {
          "ruleId": "go.doc.missing",
          "ruleIndex": 1,
          "level": "warning",
          "message": {
            "text": "exported Do has no doc comment"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "apps/api/run.go"
                },
                "region": {
                  "startLine": 3,
                  "startColumn": 1
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
`

	sarifEmptyWant = `{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "filet",
          "version": "0.2.0",
          "informationUri": "https://github.com/FacileStudio/filet",
          "rules": []
        }
      },
      "results": []
    }
  ]
}
`
)

func TestSARIFGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, goldenReport(), "0.2.0"); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != sarifGoldenWant {
		t.Errorf("SARIF output =\n%s\nwant\n%s", got, sarifGoldenWant)
	}
}

func TestSARIFEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, Report{}, "0.2.0"); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != sarifEmptyWant {
		t.Errorf("empty SARIF output =\n%s\nwant\n%s", got, sarifEmptyWant)
	}
}
