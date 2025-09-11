package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zk/3pio/internal/orchestrator"
)

var (
	version = "0.0.1-go"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "3pio",
		Short: "AI-first test runner adapter",
		Long: `3pio translates traditional test runner output into a format optimized for AI agents,
providing persistent, structured, file-based records that are context-efficient and searchable.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// No subcommands - 3pio works as a direct wrapper

	// Allow running without "run" subcommand
	rootCmd.DisableFlagParsing = true
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Check if first arg is a known subcommand
		if len(args) > 0 {
			firstArg := args[0]
			// Check for help flags
			if firstArg == "--help" || firstArg == "-h" || firstArg == "help" {
				return cmd.Help()
			}
			// Check for version flags
			if firstArg == "--version" || firstArg == "-v" || firstArg == "version" {
				fmt.Printf("3pio version %s\n", version)
				fmt.Printf("Commit: %s\n", commit)
				fmt.Printf("Built: %s\n", date)
				return nil
			}
			// Otherwise, assume it's a test command
			return runTests(args)
		}
		// No arguments, show help
		return cmd.Help()
	}

	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTestsCore contains the core logic for running tests (testable)
func runTestsCore(args []string) (int, error) {
	// Create orchestrator configuration
	config := orchestrator.Config{
		Command: args,
	}

	// Create and run orchestrator
	orch, err := orchestrator.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create orchestrator: %v\n", err)
		return 1, err
	}

	// Run tests
	if err := orch.Run(); err != nil {
		// Check if it's a test runner not found error
		if strings.Contains(err.Error(), "no test runner detected") {
			fmt.Fprintf(os.Stderr, "\nError: Could not detect test runner from command: %s\n", strings.Join(args, " "))
			fmt.Fprintf(os.Stderr, "\n3pio currently supports:\n")
			fmt.Fprintf(os.Stderr, "  • Jest\n")
			fmt.Fprintf(os.Stderr, "  • Vitest (requires v3.0+)\n")
			fmt.Fprintf(os.Stderr, "  • pytest\n")
			fmt.Fprintf(os.Stderr, "\nExample usage:\n")
			fmt.Fprintf(os.Stderr, "  3pio npm test\n")
			fmt.Fprintf(os.Stderr, "  3pio npx jest\n")
			fmt.Fprintf(os.Stderr, "  3pio npx vitest run\n")
			fmt.Fprintf(os.Stderr, "  3pio pytest\n")
			return 1, err
		}
		
		fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
		return orch.GetExitCode(), err
	}

	// Return the exit code
	return orch.GetExitCode(), nil
}

func runTests(args []string) error {
	exitCode, err := runTestsCore(args)
	os.Exit(exitCode)
	return err // Never reached, but needed for signature
}