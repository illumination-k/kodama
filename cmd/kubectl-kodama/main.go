package main

import (
	"fmt"
	"os"

	"github.com/illumination-k/kodama/pkg/application"
	"github.com/illumination-k/kodama/pkg/presentation/commands"
)

func main() {
	// Initialize application with all dependencies
	app, err := application.NewApp("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	// Create and execute root command with dependency injection
	rootCmd := commands.NewRootCommand(app)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
