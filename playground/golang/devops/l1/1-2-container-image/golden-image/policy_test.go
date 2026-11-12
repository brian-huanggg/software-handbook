package goldenimage_test

import (
	"errors"
	"os/exec"
	"testing"
)

// conftest runs the real policy against a fixture, the same way CI would.
// It returns the exit code: 0 means the Dockerfile passes, 1 means a deny fired.
func conftest(t *testing.T, fixture string) int {
	t.Helper()
	path, err := exec.LookPath("conftest")
	if err != nil {
		t.Skip("conftest not installed")
	}
	cmd := exec.Command(path, "test", "--parser", "dockerfile", "-p", "policy", fixture)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("run conftest: %v\n%s", err, out)
	}
	return 0
}

func TestPolicy(t *testing.T) {
	tests := []struct {
		fixture string
		want    int
	}{
		{"testdata/good.Dockerfile", 0},
		{"testdata/public-latest.Dockerfile", 1},
		{"testdata/unpinned.Dockerfile", 1},
		{"testdata/arg-from.Dockerfile", 1},
		{"testdata/copy-from-image.Dockerfile", 1},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			if got := conftest(t, tt.fixture); got != tt.want {
				t.Errorf("conftest exit code = %d, want %d", got, tt.want)
			}
		})
	}
}
