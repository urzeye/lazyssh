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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestSCPArgumentHelper(t *testing.T) {
	if os.Getenv("LAZYSSH_TEST_SCP_HELPER") != "1" {
		return
	}
	index := slices.Index(os.Args, "--")
	if index < 0 {
		os.Exit(1)
	}
	data, err := json.Marshal(os.Args[index+1:])
	if err != nil {
		os.Exit(1)
	}
	fmt.Print(string(data))
	os.Exit(0)
}

func TestSCPNativeArgumentsRoundTrip(t *testing.T) {
	for _, protocol := range []SCPProtocol{SCPOverSFTP, SCPLegacy} {
		remote := `/tmp/O'Brien;$(printf unexpected)`
		command, err := BuildSCPCommand("prod", "config with spaces", "local file", remote, true, false, protocol)
		if err != nil {
			t.Fatal(err)
		}
		invocation := shellQuote(os.Args[0]) + " '-test.run=^TestSCPArgumentHelper$' '--' " + strings.TrimPrefix(command, "scp ")
		var cmd *exec.Cmd
		if runtime.GOOS == windowsOS {
			cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "& "+invocation)
		} else {
			cmd = exec.Command("sh", "-c", invocation)
		}
		cmd.Env = append(os.Environ(), "LAZYSSH_TEST_SCP_HELPER=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("native invocation: %v\n%s", err, output)
		}
		var args []string
		if err := json.Unmarshal(output, &args); err != nil {
			t.Fatalf("decode native arguments: %v: %s", err, output)
		}
		wantRemote := remote
		if protocol == SCPLegacy {
			wantRemote = quoteLegacyRemotePath(remote)
		}
		want := []string{"-F", "config with spaces", "--", "local file", "prod:" + wantRemote}
		if protocol == SCPOverSFTP {
			want = append([]string{"-s"}, want...)
		}
		if !slices.Equal(args, want) {
			t.Fatalf("native args = %q, want %q", args, want)
		}
	}
}

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
