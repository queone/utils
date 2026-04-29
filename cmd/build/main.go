// build is based on an original build.sh Bash script from the source project
// that inspired this template.
//
// Thin wrapper. Logic lives in github.com/queone/governa-buildtool. utils is a
// consumer (not the template repo), so no PostInstallHook is registered. Kept
// in-tree (not extracted to the library's cmd/) because build.sh invokes via
// `go run ./cmd/build` — extraction would move version pinning into build.sh,
// a worse propagation surface than ~20 lines of inert boilerplate. See governa
// AC102 reasoning.
package main

import (
	"fmt"
	"os"

	"github.com/queone/governa-buildtool"
)

func main() {
	cfg, help, err := buildtool.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if help {
		fmt.Print(buildtool.Usage())
		return
	}
	if err := buildtool.Run(cfg, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
