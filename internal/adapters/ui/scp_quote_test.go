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

package ui

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestShellQuoteRoundTrip(t *testing.T) {
	value := "a path's value; $(echo unwanted) `quoted`"
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		shell, err := exec.LookPath("powershell")
		if err != nil {
			t.Skip("PowerShell not installed")
		}
		cmd = exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", "& { param([string]$value) [Console]::Out.Write($value) } "+shellQuote(value))
	} else {
		cmd = exec.Command("sh", "-c", "printf '%s' "+shellQuote(value))
	}
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != value {
		t.Fatalf("quoted value changed: %q, want %q", output, value)
	}
}
