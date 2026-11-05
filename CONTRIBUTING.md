# Contributing

## Before a pull request

- `gofmt`, `go vet`, `go test ./...` and `golangci-lint run` pass.
- New behaviour has tests. Tests use in-memory fixtures; no game files in the repository.
- One concern per pull request.

## Rules

- No ooz or other Oodle code in this repository. Decompression runs as a separate program.
- Add a package when a task needs it, not before.

## Licence

Contributions are accepted under Apache-2.0 (see [LICENSE](LICENSE), section 5).
