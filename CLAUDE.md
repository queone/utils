# CLAUDE.md

## Interaction Mode
- Treat all input as exploratory discussion. Only produce artifacts or make changes when explicitly authorized

## Build
  - **NEVER run `go test`, `go build`, `go vet`, `go fmt`, or any individual Go toolchain command directly** — always use `./build.sh`, because it:
    - Updates dependencies (`go mod tidy`)
    - Formats (`go fmt`) and fixes code (`go fix`) — `go fix` enforces idiomatic Go practices
    - Runs vet (`go vet`) and static analysis (`staticcheck`)
    - Runs all tests
    - Builds binaries in $GOPATH/bin
  - `./build.sh` with no arguments builds **all** utilities in the repo
  - `./build.sh <name>` (e.g. `./build.sh cash5`) builds only that utility

