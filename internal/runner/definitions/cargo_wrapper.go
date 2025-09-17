package definitions

import (
	"io"
	"os"
)

// CargoTestWrapper wraps CargoTestDefinition to implement the Definition interface from runner package
type CargoTestWrapper struct {
	*CargoTestDefinition
}

// NewCargoTestWrapper creates a new wrapper for cargo test
func NewCargoTestWrapper(impl *CargoTestDefinition) *CargoTestWrapper {
	return &CargoTestWrapper{CargoTestDefinition: impl}
}

// Matches checks if this runner can handle the given command
func (c *CargoTestWrapper) Matches(command []string) bool {
	return c.Detect(command)
}

// GetTestFiles returns list of test files (empty for dynamic discovery)
func (c *CargoTestWrapper) GetTestFiles(args []string) ([]string, error) {
	return c.CargoTestDefinition.GetTestFiles(args)
}

// BuildCommand builds the command with JSON output flags
func (c *CargoTestWrapper) BuildCommand(args []string, adapterPath string) []string {
	// cargo test uses native processing, no adapter needed
	// Pass empty strings for ipcPath and runID as they're handled elsewhere
	return c.ModifyCommand(args, "", "")
}

// GetAdapterFileName returns empty as cargo test doesn't use an adapter
func (c *CargoTestWrapper) GetAdapterFileName() string {
	return ""
}

// InterpretExitCode maps exit codes to success/failure
func (c *CargoTestWrapper) InterpretExitCode(code int) string {
	if code == 0 {
		return "success"
	}
	return "failure"
}

// IsNative returns true as cargo test processes output directly
func (c *CargoTestWrapper) IsNative() bool {
	return true
}

// GetNativeDefinition returns the underlying cargo test definition
func (c *CargoTestWrapper) GetNativeDefinition() interface{} {
	return c.CargoTestDefinition
}

// ProcessOutputWithEnv processes the output with environment variables set
func (c *CargoTestWrapper) ProcessOutputWithEnv(stdout io.Reader, ipcPath string) error {
	// Set RUSTC_BOOTSTRAP=1 for the duration of processing
	oldVal := os.Getenv("RUSTC_BOOTSTRAP")
	_ = os.Setenv("RUSTC_BOOTSTRAP", "1")
	defer func() {
		if oldVal == "" {
			_ = os.Unsetenv("RUSTC_BOOTSTRAP")
		} else {
			_ = os.Setenv("RUSTC_BOOTSTRAP", oldVal)
		}
	}()

	return c.ProcessOutput(stdout, ipcPath)
}
