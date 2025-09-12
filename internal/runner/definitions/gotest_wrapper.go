package definitions

import (
	"io"

	"github.com/zk/3pio/internal/logger"
)

// NativeDefinition interface for test runners that process output directly
type NativeDefinition interface {
	// Name returns the name of the test runner
	Name() string
	
	// Detect checks if this runner can handle the given command
	Detect(args []string) bool
	
	// ModifyCommand modifies the command for proper execution
	ModifyCommand(cmd []string, ipcPath, runID string) []string
	
	// GetTestFiles returns list of test files (empty for dynamic discovery)
	GetTestFiles(args []string) ([]string, error)
	
	// RequiresAdapter returns whether this runner needs an external adapter
	RequiresAdapter() bool
	
	// ProcessOutput processes the test runner output and generates IPC events
	ProcessOutput(stdout io.Reader, ipcPath string) error
}

// GoTestWrapper wraps GoTestDefinition to implement the Definition interface from runner package
type GoTestWrapper struct {
	*GoTestDefinition
}

// NewGoTestWrapper creates a new wrapper for Go test support
func NewGoTestWrapper(logger *logger.FileLogger) *GoTestWrapper {
	return &GoTestWrapper{
		GoTestDefinition: NewGoTestDefinition(logger),
	}
}

// Matches checks if the command is for go test
func (g *GoTestWrapper) Matches(command []string) bool {
	return g.GoTestDefinition.Detect(command)
}

// BuildCommand builds the command with -json flag
func (g *GoTestWrapper) BuildCommand(args []string, adapterPath string) []string {
	// adapterPath is ignored for Go test
	return g.GoTestDefinition.ModifyCommand(args, "", "")
}

// GetAdapterFileName returns empty string as Go doesn't need an adapter
func (g *GoTestWrapper) GetAdapterFileName() string {
	return ""
}

// InterpretExitCode provides exit code interpretation for Go test
func (g *GoTestWrapper) InterpretExitCode(code int) string {
	if code == 0 {
		return "success"
	}
	return "failure"
}

// IsNative returns true if this is a native runner (no adapter needed)
func (g *GoTestWrapper) IsNative() bool {
	return true
}

// GetNativeDefinition returns the underlying native definition
func (g *GoTestWrapper) GetNativeDefinition() NativeDefinition {
	return g.GoTestDefinition
}