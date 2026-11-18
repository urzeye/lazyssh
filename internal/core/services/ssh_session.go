package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const maxSSHDiagnostics = 16 * 1024

type diagnosticTail struct{ data []byte }

func (b *diagnosticTail) Write(p []byte) (int, error) {
	n := len(p)
	if n >= maxSSHDiagnostics {
		b.data = append(b.data[:0], p[n-maxSSHDiagnostics:]...)
		return n, nil
	}
	if excess := len(b.data) + n - maxSSHDiagnostics; excess > 0 {
		copy(b.data, b.data[excess:])
		b.data = b.data[:len(b.data)-excess]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func runInteractiveSSH(cmd *exec.Cmd) error {
	var diagnostics diagnosticTail
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &diagnostics)
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(string(diagnostics.data))
		if message != "" {
			return fmt.Errorf("%w\n%s", err, message)
		}
		return err
	}
	return nil
}
