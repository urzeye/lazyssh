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

//go:build windows

package services

import (
	"os/exec"
	"strconv"
	"strings"
)

func listProcesses() ([]processInfo, error) {
	command := "Get-CimInstance Win32_Process -Filter \"Name = 'ssh.exe'\" | " +
		"ForEach-Object { [string]$_.ProcessId + \"`t\" + $_.CommandLine }"
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err != nil {
		return nil, err
	}

	processes := make([]processInfo, 0)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pidText, commandLine, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(pidText))
		if err != nil {
			continue
		}
		commandLine = strings.TrimSpace(commandLine)
		processes = append(processes, processInfo{
			PID:     pid,
			Args:    splitCommandLine(commandLine),
			Command: commandLine,
		})
	}
	return processes, nil
}
