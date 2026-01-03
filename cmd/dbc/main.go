package main

import (
	"fmt"
	"os"

	"github.com/ntancardoso/dbc/internal/core"
)

func main() {
	if err := core.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
