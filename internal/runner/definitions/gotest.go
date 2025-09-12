package definitions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// GoTestDefinition implements support for Go's native test runner
type GoTestDefinition struct {
	logger       *logger.FileLogger
	packageMap   map[string]*PackageInfo
	testStates   map[string]*TestState
	testToFileMap map[string]string  // Maps "package/test" to file path
	currentFile  string
	mu           sync.RWMutex
	ipcWriter    *IPCWriter
}

// PackageInfo holds information about a Go package
type PackageInfo struct {
	ImportPath string   `json:"ImportPath"`
	Dir        string   `json:"Dir"`
	TestGoFiles []string `json:"TestGoFiles"`
	XTestGoFiles []string `json:"XTestGoFiles"`
	IsCached   bool
	Status     string
}

// TestState tracks the state of a running test
type TestState struct {
	Name      string
	Package   string
	StartTime time.Time
	Output    []string
	IsPaused  bool
}

// GoTestEvent represents a single event from go test -json output
type GoTestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test,omitempty"`
	Output  string    `json:"Output,omitempty"`
	Elapsed float64   `json:"Elapsed,omitempty"`
}

// IPCWriter handles writing IPC events
type IPCWriter struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewGoTestDefinition creates a new Go test runner definition
func NewGoTestDefinition(logger *logger.FileLogger) *GoTestDefinition {
	return &GoTestDefinition{
		logger:        logger,
		packageMap:    make(map[string]*PackageInfo),
		testStates:    make(map[string]*TestState),
		testToFileMap: make(map[string]string),
	}
}

// Name returns the name of this test runner
func (g *GoTestDefinition) Name() string {
	return "go"
}

// Detect checks if the command is for go test
func (g *GoTestDefinition) Detect(args []string) bool {
	if len(args) < 2 {
		return false
	}
	
	// Check for "go test" command
	if args[0] == "go" && args[1] == "test" {
		return true
	}
	
	// Check for full path to go binary
	if strings.HasSuffix(args[0], "/go") && len(args) > 1 && args[1] == "test" {
		return true
	}
	
	return false
}

// ModifyCommand ensures the -json flag is present in the go test command
func (g *GoTestDefinition) ModifyCommand(cmd []string, ipcPath, runID string) []string {
	result := make([]string, 0, len(cmd)+1)
	hasJSON := false
	
	// Check if -json flag already exists
	for _, arg := range cmd {
		if arg == "-json" {
			hasJSON = true
		}
	}
	
	// Add -json flag after "go test" if not present
	for i, arg := range cmd {
		result = append(result, arg)
		
		// After "test" command, add -json if needed
		if i > 0 && cmd[i-1] == "go" && arg == "test" && !hasJSON {
			result = append(result, "-json")
		}
	}
	
	return result
}

// GetTestFiles extracts test files from command arguments or uses go list
func (g *GoTestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Build package patterns from args
	patterns := g.extractPackagePatterns(args)
	
	// If no patterns, use current directory
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	
	// Run go list to get package info
	packageMap, err := g.runGoList(patterns)
	if err != nil {
		g.logger.Debug("Failed to run go list: %v", err)
		return []string{}, nil // Return empty for dynamic discovery
	}
	
	// Store package map for later use
	g.packageMap = packageMap
	
	// Build test-to-file mapping
	if err := g.buildTestToFileMap(); err != nil {
		g.logger.Debug("Failed to build test-to-file map: %v", err)
		// Continue anyway, will fall back to package-based mapping
	}
	
	// Extract all test files
	var testFiles []string
	for _, pkg := range packageMap {
		for _, file := range pkg.TestGoFiles {
			testFiles = append(testFiles, filepath.Join(pkg.Dir, file))
		}
		for _, file := range pkg.XTestGoFiles {
			testFiles = append(testFiles, filepath.Join(pkg.Dir, file))
		}
	}
	
	return testFiles, nil
}

// RequiresAdapter returns false as Go test doesn't need an external adapter
func (g *GoTestDefinition) RequiresAdapter() bool {
	return false
}

// ProcessOutput reads go test JSON output and converts to IPC events
func (g *GoTestDefinition) ProcessOutput(stdout io.Reader, ipcPath string) error {
	// Initialize IPC writer
	var err error
	g.ipcWriter, err = NewIPCWriter(ipcPath)
	if err != nil {
		return fmt.Errorf("failed to create IPC writer: %w", err)
	}
	defer g.ipcWriter.Close()
	
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		
		// Try to parse as JSON
		var event GoTestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Non-JSON line (likely build error)
			g.logger.Debug("Non-JSON output: %s", string(line))
			// Send as stdout chunk for the current file
			if g.currentFile != "" {
				g.sendStdoutChunk(g.currentFile, string(line)+"\n")
			}
			continue
		}
		
		// Process the event
		if err := g.processEvent(&event); err != nil {
			g.logger.Error("Failed to process event: %v", err)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}
	
	return nil
}

// processEvent handles a single go test JSON event
func (g *GoTestDefinition) processEvent(event *GoTestEvent) error {
	switch event.Action {
	case "start":
		// Package execution starting
		if event.Test == "" {
			g.handlePackageStart(event)
		}
		
	case "run":
		// Test starting
		if event.Test != "" {
			g.handleTestRun(event)
		}
		
	case "pause":
		// Test paused (parallel execution)
		if event.Test != "" {
			g.handleTestPause(event)
		}
		
	case "cont":
		// Test continued
		if event.Test != "" {
			g.handleTestCont(event)
		}
		
	case "pass", "fail", "skip":
		// Test or package result
		if event.Test != "" {
			g.handleTestResult(event)
		} else {
			g.handlePackageResult(event)
		}
		
	case "output":
		// Test output
		g.handleOutput(event)
		
	case "bench":
		// Benchmark result (not supported yet)
		g.logger.Debug("Benchmark event (not supported): %+v", event)
	}
	
	return nil
}

// handlePackageStart processes package start events
func (g *GoTestDefinition) handlePackageStart(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Map package to files
	if pkg, ok := g.packageMap[event.Package]; ok {
		// Send testFileStart for each test file in the package
		for _, file := range pkg.TestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			g.sendTestFileStart(filePath)
		}
		for _, file := range pkg.XTestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			g.sendTestFileStart(filePath)
		}
	}
}

// handleTestRun processes test run events
func (g *GoTestDefinition) handleTestRun(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	g.testStates[key] = &TestState{
		Name:      event.Test,
		Package:   event.Package,
		StartTime: event.Time,
		Output:    []string{},
		IsPaused:  false,
	}
}

// handleTestPause processes test pause events
func (g *GoTestDefinition) handleTestPause(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if state, ok := g.testStates[key]; ok {
		state.IsPaused = true
	}
}

// handleTestCont processes test continuation events
func (g *GoTestDefinition) handleTestCont(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if state, ok := g.testStates[key]; ok {
		state.IsPaused = false
	}
}

// handleTestResult processes test result events
func (g *GoTestDefinition) handleTestResult(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	state, ok := g.testStates[key]
	if !ok {
		// Create state if it doesn't exist
		state = &TestState{
			Name:    event.Test,
			Package: event.Package,
		}
	}
	
	// First try to find the test in our test-to-file map
	// For subtests, we need to check the parent test
	testName := event.Test
	if strings.Contains(testName, "/") {
		// For subtests, use the parent test name for mapping
		parentTest := strings.Split(testName, "/")[0]
		key = fmt.Sprintf("%s/%s", event.Package, parentTest)
	}
	
	var filePath string
	filePath, ok = g.testToFileMap[key]
	if !ok {
		// Fall back to package-based mapping
		filePath = g.getFilePathForPackage(event.Package)
		if filePath == "" {
			g.logger.Debug("Could not map test %s in package %s to file", event.Test, event.Package)
			return
		}
	}
	
	// Determine status
	status := strings.ToUpper(event.Action)
	
	// Handle subtests - extract suite name if present
	var suiteName string
	testName = event.Test
	if strings.Contains(testName, "/") {
		parts := strings.SplitN(testName, "/", 2)
		suiteName = parts[0]
		testName = parts[1]
	}
	
	// Send test case event
	g.sendTestCase(filePath, testName, suiteName, status, event.Elapsed, state.Output)
	
	// Clean up state
	delete(g.testStates, key)
}

// handlePackageResult processes package result events
func (g *GoTestDefinition) handlePackageResult(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Update package info
	if pkg, ok := g.packageMap[event.Package]; ok {
		pkg.Status = strings.ToUpper(event.Action)
		
		// Check if cached
		if event.Output != "" && strings.Contains(event.Output, "(cached)") {
			pkg.IsCached = true
		}
		
		// Send testFileResult for each test file
		status := pkg.Status
		if pkg.IsCached {
			status = "CACH"
		}
		
		for _, file := range pkg.TestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			g.sendTestFileResult(filePath, status)
		}
		for _, file := range pkg.XTestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			g.sendTestFileResult(filePath, status)
		}
	}
}

// handleOutput processes output events
func (g *GoTestDefinition) handleOutput(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// If output is for a specific test, buffer it
	if event.Test != "" {
		key := fmt.Sprintf("%s/%s", event.Package, event.Test)
		if state, ok := g.testStates[key]; ok {
			state.Output = append(state.Output, event.Output)
		}
	} else {
		// Package-level output
		filePath := g.getFilePathForPackage(event.Package)
		if filePath != "" {
			g.sendStdoutChunk(filePath, event.Output)
		}
	}
}

// getFilePathForPackage maps a package to its first test file
func (g *GoTestDefinition) getFilePathForPackage(packageName string) string {
	if pkg, ok := g.packageMap[packageName]; ok {
		if len(pkg.TestGoFiles) > 0 {
			return filepath.Join(pkg.Dir, pkg.TestGoFiles[0])
		}
		if len(pkg.XTestGoFiles) > 0 {
			return filepath.Join(pkg.Dir, pkg.XTestGoFiles[0])
		}
	}
	return ""
}

// buildTestToFileMap builds a mapping of test names to their source files
func (g *GoTestDefinition) buildTestToFileMap() error {
	g.logger.Debug("Building test-to-file mapping")
	
	for pkgName, pkg := range g.packageMap {
		// Process regular test files
		for _, file := range pkg.TestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			tests, err := g.listTestsInFile(filePath, pkgName)
			if err != nil {
				g.logger.Debug("Failed to list tests in %s: %v", filePath, err)
				continue
			}
			
			for _, test := range tests {
				key := fmt.Sprintf("%s/%s", pkgName, test)
				g.testToFileMap[key] = filePath
				g.logger.Debug("Mapped test %s to file %s", key, filePath)
			}
		}
		
		// Process external test files
		for _, file := range pkg.XTestGoFiles {
			filePath := filepath.Join(pkg.Dir, file)
			tests, err := g.listTestsInFile(filePath, pkgName+"_test")
			if err != nil {
				g.logger.Debug("Failed to list tests in %s: %v", filePath, err)
				continue
			}
			
			for _, test := range tests {
				key := fmt.Sprintf("%s_test/%s", pkgName, test)
				g.testToFileMap[key] = filePath
				g.logger.Debug("Mapped test %s to file %s", key, filePath)
			}
		}
	}
	
	return nil
}

// listTestsInFile runs go test -list to get all tests in a specific file
func (g *GoTestDefinition) listTestsInFile(filePath string, packageName string) ([]string, error) {
	// Run go test -list for the specific file
	cmd := exec.Command("go", "test", "-list", ".", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Parse the output to get test names
	var tests []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and "ok" status lines
		if line != "" && !strings.HasPrefix(line, "ok") && !strings.HasPrefix(line, "?") {
			// Only include top-level test functions (not subtests)
			if strings.HasPrefix(line, "Test") || strings.HasPrefix(line, "Example") || strings.HasPrefix(line, "Benchmark") {
				tests = append(tests, line)
			}
		}
	}
	
	return tests, nil
}

// extractPackagePatterns extracts package patterns from go test arguments
func (g *GoTestDefinition) extractPackagePatterns(args []string) []string {
	var patterns []string
	skipNext := false
	
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		
		// Skip go and test commands
		if i < 2 {
			continue
		}
		
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			// Check if flag takes a value
			if arg == "-run" || arg == "-bench" || arg == "-count" || 
			   arg == "-cpu" || arg == "-parallel" || arg == "-timeout" ||
			   arg == "-benchtime" || arg == "-blockprofile" || arg == "-coverprofile" ||
			   arg == "-cpuprofile" || arg == "-memprofile" || arg == "-mutexprofile" ||
			   arg == "-outputdir" || arg == "-trace" {
				skipNext = true
			}
			continue
		}
		
		// This is likely a package pattern
		patterns = append(patterns, arg)
	}
	
	return patterns
}

// runGoList executes go list -json to get package information
func (g *GoTestDefinition) runGoList(patterns []string) (map[string]*PackageInfo, error) {
	args := append([]string{"list", "-json"}, patterns...)
	cmd := exec.Command("go", args...)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}
	
	return g.parseGoListOutput(output)
}

// parseGoListOutput parses the output of go list -json
func (g *GoTestDefinition) parseGoListOutput(output []byte) (map[string]*PackageInfo, error) {
	packages := make(map[string]*PackageInfo)
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	
	for {
		var pkg PackageInfo
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to parse go list output: %w", err)
		}
		
		// Only include packages with test files
		if len(pkg.TestGoFiles) > 0 || len(pkg.XTestGoFiles) > 0 {
			packages[pkg.ImportPath] = &pkg
		}
	}
	
	return packages, nil
}

// IPC event sending methods

func (g *GoTestDefinition) sendTestFileStart(filePath string) {
	event := map[string]interface{}{
		"eventType": "testFileStart",
		"payload": map[string]interface{}{
			"filePath": filePath,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testFileStart: %v", err)
	}
}

func (g *GoTestDefinition) sendTestCase(filePath, testName, suiteName, status string, duration float64, output []string) {
	payload := map[string]interface{}{
		"filePath": filePath,
		"testName": testName,
		"status":   status,
		"duration": duration,
	}
	
	if suiteName != "" {
		payload["suiteName"] = suiteName
	}
	
	if status == "FAIL" && len(output) > 0 {
		payload["error"] = strings.Join(output, "")
	}
	
	event := map[string]interface{}{
		"eventType": "testCase",
		"payload":   payload,
	}
	
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testCase: %v", err)
	}
}

func (g *GoTestDefinition) sendTestFileResult(filePath, status string) {
	event := map[string]interface{}{
		"eventType": "testFileResult",
		"payload": map[string]interface{}{
			"filePath": filePath,
			"status":   status,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testFileResult: %v", err)
	}
}

func (g *GoTestDefinition) sendStdoutChunk(filePath, chunk string) {
	event := map[string]interface{}{
		"eventType": "stdoutChunk",
		"payload": map[string]interface{}{
			"filePath": filePath,
			"chunk":    chunk,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send stdoutChunk: %v", err)
	}
}

// NewIPCWriter creates a new IPC writer
func NewIPCWriter(path string) (*IPCWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	
	return &IPCWriter{
		path: path,
		file: file,
	}, nil
}

// WriteEvent writes an IPC event to the file
func (w *IPCWriter) WriteEvent(event interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	_, err = w.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	
	return nil
}

// Close closes the IPC writer
func (w *IPCWriter) Close() error {
	return w.file.Close()
}