# CRUSH.md

## Build/Lint/Test Commands

- **Build**: `go build -o fers-calc cmd/cli/main.go`
- **Run all tests**: `go test ./...`
- **Run tests for a specific package**: `go test ./internal/calculation` (replace with desired package path)
- **Run a single test**: `go test -run ^TestMyFunction$ ./path/to/package` (replace `TestMyFunction` and `./path/to/package`)

## Code Style Guidelines (Go)

This project follows standard Go conventions.

- **Imports**: Organize imports into standard library, external packages, and internal packages, separated by blank lines.
- **Formatting**: Use `go fmt` and `goimports` to format code.
- **Naming Conventions**:
    - **Variables/Functions**: `camelCase` for local, `PascalCase` for exported.
    - **Packages**: `lowercase` single word.
- **Error Handling**: Return errors as the last return value. Check errors immediately after function calls. Do not `panic` unless truly unrecoverable.
- **Comments**: Use comments to explain _why_ something is done, not _what_ it does. Public functions and structs should have godoc comments.
- **Types**: Use specific types over `interface{}` where possible. Ensure type safety.
- **Concurrency**: Use goroutines and channels for concurrency. Avoid shared memory by communicating.
