// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const maxSSHDiagnostics = 16 * 1024

type diagnosticTail struct {
	mu   sync.Mutex
	data []byte
}

func (b *diagnosticTail) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
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

func (b *diagnosticTail) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.data))
}

type interactiveCommandError struct {
	err         error
	diagnostics string
}

func (e *interactiveCommandError) Error() string {
	if e.diagnostics != "" {
		return fmt.Sprintf("%v\n%s", e.err, e.diagnostics)
	}
	return e.err.Error()
}

func (e *interactiveCommandError) Unwrap() error {
	return e.err
}

func (e *interactiveCommandError) StderrAlreadyPrinted() bool {
	return e.diagnostics != ""
}

func runInteractiveSSH(cmd *exec.Cmd) error {
	var diagnostics diagnosticTail
	// Proxy children can inherit stderr and outlive ssh. Do not wait forever
	// for their pipe handles after the SSH process has already exited.
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = 2 * time.Second
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &diagnostics)
	if err := cmd.Run(); err != nil {
		message := diagnostics.String()
		if message != "" {
			return &interactiveCommandError{err: err, diagnostics: message}
		}
		return err
	}
	return nil
}
