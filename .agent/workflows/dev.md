---
description: Development workflow for the DevKit Suite project
---

// turbo-all

## Development Workflow

1. Initialize Go module and create directory structure
2. Install dependencies with `go get`
3. Write source code files
4. Build with `go build ./...`
5. Run tests with `go test ./... -v`
6. Run linter with `golangci-lint run ./...` (if available)
7. Fix any build or test errors
8. Repeat until all phases are complete

## Project Info

- Go version: 1.25
- Module path: github.com/RobinCoderZhao/API-Change-Sentinel
- Project root: /Users/robin/myworkdir/robin-go/src/API-Change-Sentinel
