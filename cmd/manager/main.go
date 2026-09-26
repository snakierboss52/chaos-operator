package main

import (
	"fmt"
	"os"

	"goland-operator/internal/app"
	"goland-operator/pkg/version"
)

func main() {
	if err := app.Start(version.Version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
