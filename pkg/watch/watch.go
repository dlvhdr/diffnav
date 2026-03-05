package watch

import (
	"os/exec"

	"github.com/charmbracelet/x/ansi"
)

// RunCmd executes a shell command and returns its stdout with ANSI codes stripped.
func RunCmd(cmd string) (string, error) {
	c := exec.Command("sh", "-c", cmd)
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return ansi.Strip(string(out)), nil
}
