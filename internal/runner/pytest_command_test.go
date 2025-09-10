package runner

import (
	"testing"
)

func TestPytestBuildCommand(t *testing.T) {
	pytest := NewPytestDefinition()
	adapterPath := "/tmp/pytest_adapter.py"
	
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		// Direct pytest invocations (10 examples)
		{
			name:     "direct pytest command",
			args:     []string{"pytest"},
			expected: []string{"pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "pytest with verbose",
			args:     []string{"pytest", "-v"},
			expected: []string{"pytest", "-p", "pytest_adapter", "-v"},
		},
		{
			name:     "pytest with test directory",
			args:     []string{"pytest", "tests/"},
			expected: []string{"pytest", "-p", "pytest_adapter", "tests/"},
		},
		{
			name:     "pytest with specific file",
			args:     []string{"pytest", "tests/unit/test_models.py"},
			expected: []string{"pytest", "-p", "pytest_adapter", "tests/unit/test_models.py"},
		},
		{
			name:     "pytest with test selection",
			args:     []string{"pytest", "tests/test_api.py::TestAPI::test_get"},
			expected: []string{"pytest", "-p", "pytest_adapter", "tests/test_api.py::TestAPI::test_get"},
		},
		{
			name:     "pytest with multiple files",
			args:     []string{"pytest", "test_math.py", "test_string.py"},
			expected: []string{"pytest", "-p", "pytest_adapter", "test_math.py", "test_string.py"},
		},
		{
			name:     "pytest with -x fail fast",
			args:     []string{"pytest", "-x"},
			expected: []string{"pytest", "-p", "pytest_adapter", "-x"},
		},
		{
			name:     "pytest with maxfail",
			args:     []string{"pytest", "--maxfail=2"},
			expected: []string{"pytest", "-p", "pytest_adapter", "--maxfail=2"},
		},
		{
			name:     "pytest with tb short",
			args:     []string{"pytest", "--tb=short"},
			expected: []string{"pytest", "-p", "pytest_adapter", "--tb=short"},
		},
		{
			name:     "py.test alternative command",
			args:     []string{"py.test"},
			expected: []string{"py.test", "-p", "pytest_adapter"},
		},
		
		// Python module invocations (10 examples)
		{
			name:     "python -m pytest",
			args:     []string{"python", "-m", "pytest"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "python3 -m pytest",
			args:     []string{"python3", "-m", "pytest"},
			expected: []string{"python3", "-m", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "python -m pytest with verbose",
			args:     []string{"python", "-m", "pytest", "-v"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "-v"},
		},
		{
			name:     "python -m pytest with test path",
			args:     []string{"python", "-m", "pytest", "tests/unit"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "tests/unit"},
		},
		{
			name:     "python -m pytest with coverage",
			args:     []string{"python", "-m", "pytest", "--cov=src"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "--cov=src"},
		},
		{
			name:     "python -m pytest with coverage report",
			args:     []string{"python", "-m", "pytest", "--cov=src", "--cov-report=term"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "--cov=src", "--cov-report=term"},
		},
		{
			name:     "python -m pytest with markers",
			args:     []string{"python", "-m", "pytest", "-m", "slow"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "-m", "slow"},
		},
		{
			name:     "python -m pytest with keyword",
			args:     []string{"python", "-m", "pytest", "-k", "test_login"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "-k", "test_login"},
		},
		{
			name:     "python3.11 specific version",
			args:     []string{"python3.11", "-m", "pytest"},
			expected: []string{"python3.11", "-m", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "python -m pytest with durations",
			args:     []string{"python", "-m", "pytest", "--durations=10"},
			expected: []string{"python", "-m", "pytest", "-p", "pytest_adapter", "--durations=10"},
		},
		
		// Poetry invocations (10 examples)
		{
			name:     "poetry run pytest",
			args:     []string{"poetry", "run", "pytest"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "poetry run pytest with verbose",
			args:     []string{"poetry", "run", "pytest", "-v"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "-v"},
		},
		{
			name:     "poetry run pytest with test path",
			args:     []string{"poetry", "run", "pytest", "tests/"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "tests/"},
		},
		{
			name:     "poetry run python -m pytest",
			args:     []string{"poetry", "run", "python", "-m", "pytest"},
			expected: []string{"poetry", "run", "python", "-m", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "poetry run pytest with markers",
			args:     []string{"poetry", "run", "pytest", "-m", "unit"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "-m", "unit"},
		},
		{
			name:     "poetry run pytest with env vars",
			args:     []string{"poetry", "run", "pytest", "--tb=short", "-x"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "--tb=short", "-x"},
		},
		{
			name:     "poetry shell pytest (simulated)",
			args:     []string{"pytest", "tests/integration"},
			expected: []string{"pytest", "-p", "pytest_adapter", "tests/integration"},
		},
		{
			name:     "poetry run pytest with junit xml",
			args:     []string{"poetry", "run", "pytest", "--junit-xml=report.xml"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "--junit-xml=report.xml"},
		},
		{
			name:     "poetry run pytest with capture=no",
			args:     []string{"poetry", "run", "pytest", "-s"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "-s"},
		},
		{
			name:     "poetry run pytest with pdb",
			args:     []string{"poetry", "run", "pytest", "--pdb"},
			expected: []string{"poetry", "run", "pytest", "-p", "pytest_adapter", "--pdb"},
		},
		
		// Pipenv invocations (10 examples)
		{
			name:     "pipenv run pytest",
			args:     []string{"pipenv", "run", "pytest"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "pipenv run pytest with verbose",
			args:     []string{"pipenv", "run", "pytest", "-vv"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "-vv"},
		},
		{
			name:     "pipenv run pytest with test file",
			args:     []string{"pipenv", "run", "pytest", "test_app.py"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "test_app.py"},
		},
		{
			name:     "pipenv run python -m pytest",
			args:     []string{"pipenv", "run", "python", "-m", "pytest"},
			expected: []string{"pipenv", "run", "python", "-m", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "pipenv run pytest with fixtures",
			args:     []string{"pipenv", "run", "pytest", "--fixtures"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "--fixtures"},
		},
		{
			name:     "pipenv run pytest with log level",
			args:     []string{"pipenv", "run", "pytest", "--log-level=DEBUG"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "--log-level=DEBUG"},
		},
		{
			name:     "pipenv run pytest with reruns",
			args:     []string{"pipenv", "run", "pytest", "--reruns=2"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "--reruns=2"},
		},
		{
			name:     "pipenv shell pytest (simulated)",
			args:     []string{"pytest", "-x", "tests/"},
			expected: []string{"pytest", "-p", "pytest_adapter", "-x", "tests/"},
		},
		{
			name:     "pipenv run pytest with parallel",
			args:     []string{"pipenv", "run", "pytest", "-n", "4"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "-n", "4"},
		},
		{
			name:     "pipenv run pytest with doctest",
			args:     []string{"pipenv", "run", "pytest", "--doctest-modules"},
			expected: []string{"pipenv", "run", "pytest", "-p", "pytest_adapter", "--doctest-modules"},
		},
		
		// Tox invocations (10 examples)
		{
			name:     "tox with pytest",
			args:     []string{"tox", "-e", "py39", "--", "pytest"},
			expected: []string{"tox", "-e", "py39", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox with pytest and args",
			args:     []string{"tox", "-e", "py39", "--", "pytest", "-v"},
			expected: []string{"tox", "-e", "py39", "--", "pytest", "-p", "pytest_adapter", "-v"},
		},
		{
			name:     "tox multiple envs",
			args:     []string{"tox", "-e", "py38,py39", "--", "pytest"},
			expected: []string{"tox", "-e", "py38,py39", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox with specific test",
			args:     []string{"tox", "-e", "py39", "--", "pytest", "tests/test_core.py"},
			expected: []string{"tox", "-e", "py39", "--", "pytest", "-p", "pytest_adapter", "tests/test_core.py"},
		},
		{
			name:     "tox recreate with pytest",
			args:     []string{"tox", "-r", "-e", "py39", "--", "pytest"},
			expected: []string{"tox", "-r", "-e", "py39", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox parallel",
			args:     []string{"tox", "-p", "--", "pytest"},
			expected: []string{"tox", "-p", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox with posargs",
			args:     []string{"tox", "--", "pytest", "-x", "tests/"},
			expected: []string{"tox", "--", "pytest", "-p", "pytest_adapter", "-x", "tests/"},
		},
		{
			name:     "tox quiet mode",
			args:     []string{"tox", "-q", "--", "pytest"},
			expected: []string{"tox", "-q", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox with custom config",
			args:     []string{"tox", "-c", "tox-test.ini", "--", "pytest"},
			expected: []string{"tox", "-c", "tox-test.ini", "--", "pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "tox list envs (no pytest)",
			args:     []string{"tox", "-l"},
			expected: []string{"tox", "-l", "-p", "pytest_adapter"},
		},
		
		// Virtual environment variations
		{
			name:     "venv pytest",
			args:     []string{"venv/bin/pytest"},
			expected: []string{"venv/bin/pytest", "-p", "pytest_adapter"},
		},
		{
			name:     ".venv pytest",
			args:     []string{".venv/bin/pytest"},
			expected: []string{".venv/bin/pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "env pytest",
			args:     []string{"env/bin/pytest"},
			expected: []string{"env/bin/pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "virtualenv python pytest",
			args:     []string{"venv/bin/python", "-m", "pytest"},
			expected: []string{"venv/bin/python", "-m", "pytest", "-p", "pytest_adapter"},
		},
		
		// Make and script invocations
		{
			name:     "make test (simulated as pytest)",
			args:     []string{"pytest"},
			expected: []string{"pytest", "-p", "pytest_adapter"},
		},
		{
			name:     "script with pytest",
			args:     []string{"./scripts/test.sh", "pytest"},
			expected: []string{"./scripts/test.sh", "pytest", "-p", "pytest_adapter"},
		},
		
		// Edge cases
		{
			name:     "pytest with plugin already specified",
			args:     []string{"pytest", "-p", "no:warnings"},
			expected: []string{"pytest", "-p", "pytest_adapter", "-p", "no:warnings"},
		},
		{
			name:     "pytest with complex args",
			args:     []string{"pytest", "--cov=myproj", "--cov-report=html", "--cov-report=term", "-vv", "--tb=short"},
			expected: []string{"pytest", "-p", "pytest_adapter", "--cov=myproj", "--cov-report=html", "--cov-report=term", "-vv", "--tb=short"},
		},
		{
			name:     "pytest with custom ini",
			args:     []string{"pytest", "-c", "pytest-integration.ini"},
			expected: []string{"pytest", "-p", "pytest_adapter", "-c", "pytest-integration.ini"},
		},
		{
			name:     "pytest with env var style",
			args:     []string{"PYTEST_TIMEOUT=300", "pytest"},
			expected: []string{"PYTEST_TIMEOUT=300", "pytest", "-p", "pytest_adapter"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pytest.BuildCommand(tt.args, adapterPath)
			
			if len(result) != len(tt.expected) {
				t.Errorf("BuildCommand() length = %v, want %v", len(result), len(tt.expected))
				t.Errorf("Got:      %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("BuildCommand() result[%d] = %v, want %v", i, v, tt.expected[i])
					t.Errorf("Got:      %v", result)
					t.Errorf("Expected: %v", tt.expected)
					break
				}
			}
		})
	}
}