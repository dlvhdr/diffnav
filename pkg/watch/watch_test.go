package watch

import (
	"testing"
)

func TestRunCmdSuccess(t *testing.T) {
	out, err := RunCmd("echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello\n" {
		t.Fatalf("expected %q, got %q", "hello\n", out)
	}
}

func TestRunCmdError(t *testing.T) {
	_, err := RunCmd("false")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunCmdStripsANSI(t *testing.T) {
	out, err := RunCmd(`printf '\033[31mred\033[0m'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "red" {
		t.Fatalf("expected %q, got %q", "red", out)
	}
}
