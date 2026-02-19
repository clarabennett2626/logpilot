package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestPipeMode_JSON(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader(`{"level":"info","msg":"hello","ts":"2024-01-01T00:00:00Z"}` + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "hello") {
		t.Errorf("expected output to contain 'hello', got: %q", output)
	}
}

func TestPipeMode_MultiFormat(t *testing.T) {
	input := `{"level":"info","msg":"json line"}
level=warn msg="logfmt line"
plain text line
`
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := out.String()
	for _, want := range []string{"json line", "logfmt line", "plain text line"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got: %q", want, output)
		}
	}
}

func TestPipeMode_EmptyInput(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader("")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("expected no output for empty input, got: %q", out.String())
	}
}

func TestPipeMode_LongLine(t *testing.T) {
	long := strings.Repeat("x", 500_000)
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader(long + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if !strings.Contains(out.String(), "xxx") {
		t.Error("expected long line to be processed")
	}
}
