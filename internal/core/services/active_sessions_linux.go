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

//go:build linux

package services

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func listProcesses() ([]processInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	currentUID := os.Getuid()
	processes := make([]processInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		procDir := filepath.Join("/proc", entry.Name())
		if !procStatusMatchesUID(filepath.Join(procDir, "status"), currentUID) {
			continue
		}
		args, ok := readProcCmdline(filepath.Join(procDir, "cmdline"))
		if !ok || len(args) == 0 {
			continue
		}
		processes = append(processes, processInfo{
			PID:     pid,
			Args:    args,
			Command: strings.Join(args, " "),
		})
	}
	return processes, nil
}

func procStatusMatchesUID(path string, currentUID int) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return false
		}
		uid, err := strconv.Atoi(fields[1])
		return err == nil && uid == currentUID
	}
	return false
}

func readProcCmdline(path string) ([]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}

	rawArgs := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	args := make([]string, 0, len(rawArgs))
	for _, arg := range rawArgs {
		if arg != "" {
			args = append(args, arg)
		}
	}
	return args, len(args) > 0
}
