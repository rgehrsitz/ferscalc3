# Repository Guidelines

This guide summarizes how to work effectively inside the `ferscalc3` Go workspace while keeping builds reproducible and reviews efficient.

## Project Structure & Module Organization
- `cmd/fers-calc/`: CLI entrypoint; wire flags and dispatch calculations.
- `internal/`: Core logic split into `domain`, `calculation`, `config`, and report helpers; keep new engines here.
- `pkg/`: Shared utilities (decimal math, date helpers) that must stay dependency-free.
- `data/` and root `*.yaml` configs: Reference stress libraries and runnable examples; avoid editing checked-in golden data unless requirements change.
- `docs/`, `test/`, and `tools/`: Long-form guidance, sample inputs/outputs, and maintenance scripts.

## Build, Test, and Development Commands
- `make build` / `make build-all`: Compile CLI for the host or all supported targets with embedded version metadata.
- `go run ./cmd/fers-calc calculate example_config.yaml`: Fast manual smoke test of calculation flow without producing binaries.
- `make test`, `make test-unit`, `make test-all`: Run the Go test suites (default, `-short -tags=unit`, and full `unit` tag).
- `make test-coverage`: Produce `coverage.out` and `coverage.html` for diff review.
- `make fmt`, `make vet`, `make lint`: Format, vet, and lint; `make verify` chains the first two plus `test` for pre-push assurance.

## Coding Style & Naming Conventions
Use Go 1.21 standards with tabs for indentation. Apply `go fmt ./...` (or `make fmt`) before sending a PR; keep imports grouped by standard/library/local. Exported identifiers remain CamelCase, package-internal helpers stay lowerCamel. Align YAML keys with the existing snake_case patterns (`tsp_allocation`, `medicare_config`).

## Testing Guidelines
All behavior changes should ship with `go test ./...` success locally. Add or update tests close to the logic (e.g., `internal/calculation` or `pkg/decimal`). Name files with `_test.go` and functions `Test<Feature>_<Scenario>`. Use build tags only when necessary (`//go:build unit`). Keep coverage roughly at current levels by extending golden configs in `test/` when new fields appear.

## Commit & Pull Request Guidelines
Recent history mixes structured messages (`feat:`, `refactor:`) with descriptive prose; default to Conventional Commits so change logs stay scannable. Each PR should describe the user-facing impact, reference any GitHub issues, and include command output or screenshots for report/UI shifts. Mention whether `make verify` was run.

## Configuration & Security Tips
Sample configs often contain realistic cash-flow figures. Never commit personal data—use anonymized copies under `data/` or `test/`. When sharing reports, drop them in `docs/` or a git-ignored `reports/` folder; rely on `make clean` or `make clean-local` to remove artifacts before committing.
