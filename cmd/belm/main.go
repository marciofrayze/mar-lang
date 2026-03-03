package main

import (
	"fmt"
	"os"

	"belm/internal/cli"
)

func main() {
	if err := cli.Run("belm", os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
