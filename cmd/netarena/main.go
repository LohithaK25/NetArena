package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "netarena: %v\n", err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	_, err := fmt.Fprintln(out, "NetArena Go foundation is ready.")
	return err
}
