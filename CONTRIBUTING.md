# Contributing

## Before a pull request

- `gofmt`, `go vet`, `go test ./...` and `golangci-lint run` pass.
- New behaviour has tests. Tests use in-memory fixtures; no game files in the repository.
- One concern per pull request.

## Rules

- No ooz or other Oodle code in this repository. Decompression runs as a separate program.
- Add a package when a task needs it, not before.

## Working with pbt-engine

The dataset format lives in [pbt-engine](https://github.com/xaroth/pbt-engine). To build against a local checkout next to this one, run `go work init . ../pbt-engine`; `go.work` is git-ignored. A change to the format lands in pbt-engine first.

## Licence

Contributions are accepted under Apache-2.0 (see [LICENSE](LICENSE), section 5).
