package orchestrator

import (
	"testing"
	"github.com/zk/3pio/internal/report"
)

func TestSanitizePathConsistency(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // What report.SanitizeGroupName produces
	}{
		{
			name:     "Simple JS file",
			input:    "./math.test.js",
			expected: "_math_test_js",
		},
		{
			name:     "Nested path with JS extension and dashes",
			input:    "./test/system/mcp-tools/scroll.test.js",
			expected: "_test_system_mcp_tools_scroll_test_js",
		},
		{
			name:     "Path without leading dot",
			input:    "test/system/api.test.ts",
			expected: "test_system_api_test_ts",
		},
		{
			name:     "Python test file",
			input:    "./tests/unit/test_main.py",
			expected: "_tests_unit_test_main_py",
		},
		{
			name:     "File with multiple dots",
			input:    "./src/utils.test.integration.js",
			expected: "_src_utils_test_integration_js",
		},
		{
			name:     "File with dashes and dots",
			input:    "./my-component.spec.tsx",
			expected: "_my_component_spec_tsx",
		},
		{
			name:     "Kebab case directory",
			input:    "./test-utils/mock-data/users.test.js",
			expected: "_test_utils_mock_data_users_test_js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that report.SanitizeGroupName produces expected results
			reportResult := report.SanitizeGroupName(tt.input)

			if reportResult != tt.expected {
				t.Errorf("report.SanitizeGroupName(%q) = %q, expected %q",
					tt.input, reportResult, tt.expected)
			}
		})
	}
}