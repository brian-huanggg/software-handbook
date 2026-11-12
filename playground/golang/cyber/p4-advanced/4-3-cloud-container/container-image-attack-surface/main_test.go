package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"privileged container fails the gate", []string{"run", "testdata/inspect-privileged.json"}, 1},
		{"hardened container passes", []string{"run", "testdata/inspect-hardened.json"}, 0},
		{"leaky Dockerfile fails", []string{"dockerfile", "lab/Dockerfile.leaky"}, 1},
		{"fixed Dockerfile passes", []string{"dockerfile", "lab/Dockerfile.fixed"}, 0},
		{"missing file is a usage error, not a finding", []string{"run", "testdata/nope.json"}, 2},
		{"unknown subcommand", []string{"deploy", "x"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if got := run(tt.args, strings.NewReader(""), &out, &errOut); got != tt.want {
				t.Errorf("exit = %d, want %d\nstdout: %s\nstderr: %s", got, tt.want, out.String(), errOut.String())
			}
		})
	}
}
