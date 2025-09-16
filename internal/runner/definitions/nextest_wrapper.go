package definitions

import (
	"io"
)

// NextestWrapper wraps NextestDefinition to implement the Definition interface from runner package
type NextestWrapper struct {
	*NextestDefinition
}

// NewNextestWrapper creates a new wrapper for cargo nextest
func NewNextestWrapper(impl *NextestDefinition) *NextestWrapper {
	return &NextestWrapper{NextestDefinition: impl}
}

// Matches checks if this runner can handle the given command
func (n *NextestWrapper) Matches(command []string) bool {
	return n.Detect(command)
}

// GetTestFiles returns list of test files (empty for dynamic discovery)
func (n *NextestWrapper) GetTestFiles(args []string) ([]string, error) {
	return n.NextestDefinition.GetTestFiles(args)
}

// BuildCommand builds the command with JSON output flags
func (n *NextestWrapper) BuildCommand(args []string, adapterPath string) []string {
	// nextest uses native processing, no adapter needed
	// Pass empty strings for ipcPath and runID as they're handled elsewhere
	return n.ModifyCommand(args, "", "")
}

// GetAdapterFileName returns empty as nextest doesn't use an adapter
func (n *NextestWrapper) GetAdapterFileName() string {
	return ""
}

// InterpretExitCode maps exit codes to success/failure
func (n *NextestWrapper) InterpretExitCode(code int) string {
	if code == 0 {
		return "success"
	}
	return "failure"
}

// IsNative returns true as nextest processes output directly
func (n *NextestWrapper) IsNative() bool {
	return true
}

// GetNativeDefinition returns the underlying nextest definition
func (n *NextestWrapper) GetNativeDefinition() interface{} {
	return n.NextestDefinition
}

// ProcessOutput processes the nextest output
func (n *NextestWrapper) ProcessOutput(stdout io.Reader, ipcPath string) error {
	return n.NextestDefinition.ProcessOutput(stdout, ipcPath)
}
