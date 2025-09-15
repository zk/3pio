package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/orchestrator"
)

var (
	version = "0.0.1-go"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "3pio [your full test command] | [flags]",
		Short: "Context-optimized test runner adapter",
		Long: `3pio translates test runs into a format optimized for AI agents, providing
context-optimized console output and file-based records.

Structured reports are written to .3pio/runs/[timestamp]-[memorable-name]/:
• test-run.md  - Main report with test summary and individual test results
• output.log   - Complete stdout/stderr output from the entire test run  
• logs/*.log   - Per-file output with test case demarcation

Examples:
  3pio npm test                    # Run npm test script
  3pio npm test -- tests/unit      # Pass arguments to npm test
  3pio npx jest                    # Run Jest directly
  3pio npx vitest run              # Run Vitest
  3pio pytest                      # Run pytest`,
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

	// Custom help template to put Usage before Examples
	rootCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}

Flags:
{{.Flags.FlagUsages | trimTrailingWhitespaces}}
`)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTestsCore contains the core logic for running tests (testable)
func runTestsCore(args []string) (int, error) {
	// Check for unsupported modes
	if err := checkUnsupportedModes(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1, err
	}

	// Create file logger
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create debug logger: %v\n", err)
		return 1, err
	}
	defer func() {
		if err := fileLogger.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close debug log: %v\n", err)
		}
	}()

	// Create orchestrator configuration
	config := orchestrator.Config{
		Command: args,
		Logger:  fileLogger,
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
			fmt.Fprintf(os.Stderr, "\nTest Runners:\n")
			fmt.Fprintf(os.Stderr, "  • Jest\n")
			fmt.Fprintf(os.Stderr, "  • Vitest (requires v3.0+)\n")
			fmt.Fprintf(os.Stderr, "  • pytest\n")
			fmt.Fprintf(os.Stderr, "  • go test\n")
			fmt.Fprintf(os.Stderr, "\nPackage Managers:\n")
			fmt.Fprintf(os.Stderr, "  • npm\n")
			fmt.Fprintf(os.Stderr, "  • yarn\n")
			fmt.Fprintf(os.Stderr, "  • pnpm\n")
			fmt.Fprintf(os.Stderr, "  • bun\n")
			fmt.Fprintf(os.Stderr, "\nExample usage:\n")
			fmt.Fprintf(os.Stderr, "  3pio npm test\n")
			fmt.Fprintf(os.Stderr, "  3pio yarn test\n")
			fmt.Fprintf(os.Stderr, "  3pio pnpm test\n")
			fmt.Fprintf(os.Stderr, "  3pio npx jest\n")
			fmt.Fprintf(os.Stderr, "  3pio npx vitest run\n")
			fmt.Fprintf(os.Stderr, "  3pio pytest\n")
			fmt.Fprintf(os.Stderr, "  3pio go test ./...\n")
			return 1, err
		}

		fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
		return orch.GetExitCode(), err
	}

	// Return the exit code
	return orch.GetExitCode(), nil
}

func runTests(args []string) error {
	exitCode, _ := runTestsCore(args)
	os.Exit(exitCode)
	return nil // Never reached, but needed for signature
}

// checkUnsupportedModes checks for watch mode and coverage mode
func checkUnsupportedModes(args []string) error {
	// Join all args to check for flags
	cmdStr := strings.Join(args, " ")

	// Check for watch mode
	watchPatterns := []string{
		"--watch",
		"--watchAll",
		" -w ", // Short form with spaces to avoid matching --workspace
		" -w\t", // Short form with tab
		"pytest-watch",
		"ptw", // pytest-watch alias
	}

	// Also check for -w at the end or beginning of args
	for _, arg := range args {
		if arg == "-w" {
			return fmt.Errorf("watch mode is not supported. Please run tests in single-run mode without -w flag")
		}
	}

	// Special case: plain 'vitest' without 'run' defaults to watch mode
	if len(args) > 0 && strings.Contains(cmdStr, "vitest") && !strings.Contains(cmdStr, "run") {
		// Check if it's actually vitest command
		for _, arg := range args {
			if strings.HasSuffix(arg, "vitest") || arg == "vitest" {
				// Check if 'run' is in subsequent args
				hasRun := false
				for _, a := range args {
					if a == "run" {
						hasRun = true
						break
					}
				}
				if !hasRun {
					return fmt.Errorf("watch mode is not supported. Vitest defaults to watch mode. Please use 'vitest run' for single-run mode")
				}
			}
		}
	}

	for _, pattern := range watchPatterns {
		if strings.Contains(cmdStr, pattern) {
			return fmt.Errorf("watch mode is not supported. Please run tests in single-run mode without %s flag", pattern)
		}
	}

	// Check for coverage mode
	coveragePatterns := []string{
		"--coverage",
		"--collectCoverage",
		"--cov",
		"--cov-report",
		"nyc",
		"c8",
		"tarpaulin",
		"llvm-cov",
	}

	for _, pattern := range coveragePatterns {
		if strings.Contains(cmdStr, pattern) {
			return fmt.Errorf("coverage mode is not supported. Please run tests without coverage flags")
		}
	}

	return nil
}
