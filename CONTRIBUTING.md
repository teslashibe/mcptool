# Contributing

Issues and pull requests are welcome.

Before proposing a change, search the existing issues. For substantial or
breaking changes, open an issue first so the approach can be discussed.

Use Go 1.25 or newer on a currently supported patched release. Keep changes
focused, add or update tests when behavior changes, and run:

```sh
go mod tidy
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

Pull requests should explain the motivation, describe the change, and note any
compatibility impact.
