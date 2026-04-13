# CLAUDE.md

## Interaction Mode
- Treat all input as exploratory discussion. Only produce artifacts or make changes when explicitly authorized

## Build & Release
  - **NEVER run `go test`, `go build`, `go vet`, `go fmt`, or any individual Go toolchain command directly** — always use `./build.sh`, because it:
    - Updates dependencies (`go mod tidy`)
    - Formats (`go fmt`) and fixes code (`go fix`) — `go fix` enforces idiomatic Go practices
    - Runs vet (`go vet`) and static analysis (`staticcheck`)
    - Runs all tests
    - Builds binaries in $GOPATH/bin
  - `./build.sh` with no arguments builds **all** utilities in the repo
  - `./build.sh <name>` (e.g. `./build.sh cash5`) builds only that utility
  - `./build.sh -h` for build help, `./build.sh v0.0.0 -h` for release help
  - **Release** (`./build.sh vX.Y.Z "message"`) is a separate pipeline that only does git add/commit/tag/push — it does **not** re-run the build/test pipeline. Always run a successful `./build.sh` first before releasing.

