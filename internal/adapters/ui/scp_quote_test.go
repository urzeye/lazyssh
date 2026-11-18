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
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestSCPArgumentHelper(t *testing.T) {
	if os.Getenv("LAZYSSH_TEST_SCP_HELPER") != "1" {
		return
	}
	if os.Getenv("LAZYSSH_TEST_SCP_EXIT") == "17" {
		os.Exit(17)
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

func TestWindowsNativeCommandPreservesExitCode(t *testing.T) {
	if runtime.GOOS != windowsOS {
		t.Skip("Windows process launch behavior")
	}
	for _, shell := range []string{"powershell", "pwsh"} {
		if _, err := exec.LookPath(shell); err != nil {
			t.Logf("%s unavailable: %v", shell, err)
			continue
		}
		command := formatNativeCommand(os.Args[0], []string{"-test.run=^TestSCPArgumentHelper$", "--"})
		script := "$ErrorActionPreference = 'Continue'; " + command + "; if ($ErrorActionPreference -ne 'Continue') { throw 'error preference leaked' }; [Console]::Write($LASTEXITCODE)"
		cmd := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", script)
		cmd.Env = append(os.Environ(), "LAZYSSH_TEST_SCP_HELPER=1", "LAZYSSH_TEST_SCP_EXIT=17")
		output, err := cmd.CombinedOutput()
		if err != nil || string(output) != "17" {
			t.Fatalf("%s lost native exit code: %q, %v", shell, output, err)
		}
	}
}

func TestSCPNativeArgumentsRoundTrip(t *testing.T) {
	shells := []struct{ name, mode string }{{"sh", ""}}
	if runtime.GOOS == windowsOS {
		shells = []struct{ name, mode string }{{"powershell", "Legacy"}, {"pwsh", "Legacy"}, {"pwsh", "Standard"}, {"pwsh", "Windows"}}
	}
	for _, shell := range shells {
		if _, err := exec.LookPath(shell.name); err != nil {
			t.Logf("shell %s unavailable: %v", shell.name, err)
			continue
		}
		for _, protocol := range []SCPProtocol{SCPOverSFTP, SCPLegacy} {
			for _, remote := range []string{`/tmp/O'Brien;$(printf unexpected)`, `/tmp/a"b`, `/tmp/a "b" c`, `/tmp/a\"b`, `/tmp/%PATH%"end`, `/tmp/trailing \`, "/tmp/`quoted`"} {
				t.Run(fmt.Sprintf("%s/%s/%d/%s", shell.name, shell.mode, protocol, remote), func(t *testing.T) {
					command, err := BuildSCPCommand("prod", "config with spaces", `local file\`, remote, true, false, protocol)
					if err != nil {
						t.Fatal(err)
					}
					executable := shellQuote(os.Args[0]) + " '-test.run=^TestSCPArgumentHelper$' '--' "
					var cmd *exec.Cmd
					if runtime.GOOS == windowsOS {
						invocation := strings.Replace(command, "FileName = 'scp'", "FileName = "+shellQuote(os.Args[0]), 1)
						invocation = strings.Replace(invocation, "Arguments = '", "Arguments = '-test.run=^TestSCPArgumentHelper$ -- ", 1)
						script := "$PSNativeCommandArgumentPassing = '" + shell.mode + "'; " + invocation + "; if ($PSNativeCommandArgumentPassing -ne '" + shell.mode + "') { throw 'native argument preference leaked' }"
						cmd = exec.Command(shell.name, "-NoProfile", "-NonInteractive", "-Command", script)
					} else {
						cmd = exec.Command("sh", "-c", strings.Replace(command, "scp ", executable, 1))
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
					local := `local file\`
					if runtime.GOOS == windowsOS {
						local = filepath.ToSlash(local)
					}
					want := []string{"-F", "config with spaces", "--", local, "prod:" + wantRemote}
					if protocol == SCPOverSFTP {
						want = append([]string{"-s"}, want...)
					}
					if !slices.Equal(args, want) {
						t.Fatalf("native args = %q, want %q", args, want)
					}
				})
			}
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
