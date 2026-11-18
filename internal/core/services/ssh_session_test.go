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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSSHSessionHelper(t *testing.T) {
	mode := os.Getenv("LAZYSSH_TEST_SESSION_HELPER")
	if mode == "" {
		return
	}
	if mode == "resolve" {
		fmt.Println("hostname 127.0.0.1")
		fmt.Println("port " + os.Getenv("LAZYSSH_TEST_PORT"))
		os.Exit(0)
	}
	if mode == "fail" {
		fmt.Fprintln(os.Stderr, "Permission denied (publickey).")
		os.Exit(255)
	}
	os.Exit(0)
}

func TestInteractiveSSHRetainsFailureDiagnostics(t *testing.T) {
	for _, mode := range []string{"fail", "success"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSSHSessionHelper$")
			cmd.Env = append(os.Environ(), "LAZYSSH_TEST_SESSION_HELPER="+mode)
			err := runInteractiveSSH(cmd)
			if mode == "success" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 255 || !strings.Contains(err.Error(), "Permission denied") {
				t.Fatalf("diagnostics or exit code lost: %v", err)
			}
		})
	}
}

func TestSSHDiagnosticsAreBounded(t *testing.T) {
	var tail diagnosticTail
	_, _ = tail.Write([]byte(strings.Repeat("a", maxSSHDiagnostics*2)))
	_, _ = tail.Write([]byte("final error"))
	if len(tail.data) != maxSSHDiagnostics || !strings.HasSuffix(string(tail.data), "final error") {
		t.Fatal("diagnostics not bounded or lost the final error")
	}
}
