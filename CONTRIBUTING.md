# Contributing to Nexus

Thank you for contributing.

## Development requirements

- Go version declared in `go.mod`
- `gofmt`
- `go vet ./...`
- `go test -race ./...`

## Before opening a pull request

Run:

```bash
gofmt -w .
go mod tidy
go vet ./...
go test -race ./...