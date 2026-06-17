package main

import (
	"log"
	"os"

	"github.com/tmc/specmd/internal/lsp"
)

func main() {
	if err := lsp.NewServer(os.Stdin, os.Stdout).Run(); err != nil {
		log.Fatal(err)
	}
}
