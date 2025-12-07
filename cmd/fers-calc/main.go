package main

import (
	"fmt"
	"os"

	"github.com/rpgo/retirement-calculator/internal/cli"
)

// not much here
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
