// rel runs the utils release pipeline.
// Invoke via: go run ./cmd/rel vX.Y.Z "release message"
// Or via the convenience wrapper: ./build.sh vX.Y.Z "release message"
package main

import (
	"fmt"
	"os"

	"github.com/queone/utils/internal/reltool"
)

func main() {
	cfg, help, err := reltool.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if help {
		fmt.Print(reltool.Usage())
		return
	}
	if err := reltool.Run(cfg, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
