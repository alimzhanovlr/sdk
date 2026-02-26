package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "microkit",
		Short:   "Microkit CLI - Generate microservices with clean architecture",
		Long:    `A CLI tool for generating Go microservices using clean architecture principles.`,
		Version: version,
	}

	rootCmd.AddCommand(
		newGenerateCmd(),
		newInitCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
