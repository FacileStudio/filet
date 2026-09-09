#!/bin/sh
set -e

# project quality gate for the filet repo

echo "▶ gofmt check"
if [ -n "$(gofmt -l .)" ]; then
  echo "gofmt failures:"
  gofmt -l .
  exit 1
fi

echo "▶ go vet"
go vet ./...

echo "▶ go test"
go test ./...

echo "✓ gate passed"
