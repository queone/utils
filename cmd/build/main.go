// build runs the utils build/test pipeline.
// Invoke via: go run ./cmd/build [target ...] [-v|--verbose]
// Or via the convenience wrapper: ./build.sh [target ...] [-v|--verbose]
package main

import (
	"fmt"
	"os"

	"github.com/queone/utils/internal/buildtool"
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
