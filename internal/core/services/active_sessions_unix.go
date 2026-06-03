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

//go:build !windows && !linux

package services

import (
	"os/exec"
	"os/user"
	"strings"
)

func listProcesses() ([]processInfo, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	// #nosec G204 -- currentUser.Username comes from the local OS account and scopes ps to this user.
	out, err := exec.Command("ps", "-U", currentUser.Username, "-o", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}

	processes := make([]processInfo, 0)
	for _, line := range strings.Split(string(out), "\n") {
		process, ok := parseProcessLine(line)
		if ok {
			processes = append(processes, process)
		}
	}
	return processes, nil
}
