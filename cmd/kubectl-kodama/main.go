package main

import (
	"os"

	"github.com/illumination-k/kodama/pkg/commands"
)

func main() {
	rootCmd := commands.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
