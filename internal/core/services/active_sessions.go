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
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/urzeye/lazyssh/internal/core/domain"
)

type processInfo struct {
	PID     int
	Args    []string
	Command string
}

// ListActiveSessions returns running local ssh client processes.
func (s *serverService) ListActiveSessions() ([]domain.ActiveSession, error) {
	processes, err := listProcesses()
	if err != nil {
		s.logger.Warnw("failed to list active ssh sessions", "error", err)
		return nil, err
	}

	sessions := make([]domain.ActiveSession, 0)
	for _, process := range processes {
		session, ok := parseSSHProcess(process)
		if ok {
			sessions = append(sessions, session)
		}
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].Alias != sessions[j].Alias {
			return strings.ToLower(sessions[i].Alias) < strings.ToLower(sessions[j].Alias)
		}
		return sessions[i].PID < sessions[j].PID
	})

	return sessions, nil
}

func parseSSHProcessLine(line string) (domain.ActiveSession, bool) {
	process, ok := parseProcessLine(line)
	if !ok {
		return domain.ActiveSession{}, false
	}
	return parseSSHProcess(process)
}

func parseProcessLine(line string) (processInfo, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return processInfo{}, false
	}

	pidText, command, ok := strings.Cut(line, " ")
	if !ok {
		return processInfo{}, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(pidText))
	if err != nil {
		return processInfo{}, false
	}

	command = strings.TrimSpace(command)
	return processInfo{
		PID:     pid,
		Args:    splitCommandLine(command),
		Command: command,
	}, true
}

func parseSSHProcess(process processInfo) (domain.ActiveSession, bool) {
	if len(process.Args) == 0 || !isSSHExecutable(process.Args[0]) {
		return domain.ActiveSession{}, false
	}

	destination, forwards := parseSSHDestination(process.Args[1:])
	if destination == "" {
		return domain.ActiveSession{}, false
	}

	return domain.ActiveSession{
		PID:         process.PID,
		Alias:       displayAlias(destination),
		Destination: destination,
		Command:     process.Command,
		Forwards:    forwards,
	}, true
}

func isSSHExecutable(arg string) bool {
	base := path.Base(strings.ReplaceAll(strings.TrimSpace(arg), "\\", "/"))
	return strings.EqualFold(base, "ssh") || strings.EqualFold(base, "ssh.exe")
}

func parseSSHDestination(args []string) (string, []string) {
	forwards := make([]string, 0)

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "" {
			continue
		}
		if arg == "--" {
			if index+1 < len(args) {
				return args[index+1], forwards
			}
			return "", forwards
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return arg, forwards
		}
		if strings.HasPrefix(arg, "--") {
			continue
		}

		for j := 1; j < len(arg); j++ {
			ch := arg[j]
			// Non-session inspection flags (config dump, version, query) are not active sessions.
			if ch == 'G' || ch == 'V' || ch == 'Q' {
				return "", nil
			}
			if ch == 'L' || ch == 'R' || ch == 'D' {
				forwarding := arg[j+1:]
				if forwarding == "" && index+1 < len(args) {
					forwarding = args[index+1]
					index++
				}
				if forwarding != "" {
					forwards = append(forwards, forwarding)
				}
				break
			}
			if strings.ContainsRune("BbcEeFIiJlmoOpPSWw", rune(ch)) {
				if j+1 == len(arg) && index+1 < len(args) {
					index++
				}
				break
			}
		}
	}

	return "", forwards
}

func displayAlias(destination string) string {
	alias := strings.TrimSpace(destination)
	if len(alias) >= 6 && strings.EqualFold(alias[:6], "ssh://") {
		alias = alias[6:]
	}
	if at := strings.LastIndex(alias, "@"); at >= 0 && at < len(alias)-1 {
		alias = alias[at+1:]
	}
	alias = strings.TrimPrefix(alias, "[")
	if end := strings.Index(alias, "]"); end >= 0 {
		return alias[:end]
	}
	if strings.Count(alias, ":") == 1 {
		colon := strings.LastIndex(alias, ":")
		return alias[:colon]
	}
	return alias
}

func splitWindowsCommandLine(command string) []string {
	fields := make([]string, 0)
	var current strings.Builder
	inQuotes := false
	tokenStarted := false

	flush := func() {
		if tokenStarted {
			fields = append(fields, current.String())
			current.Reset()
			tokenStarted = false
		}
	}

	for i := 0; i < len(command); {
		ch := command[i]
		if !inQuotes && (ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n') {
			flush()
			i++
			continue
		}
		tokenStarted = true
		if ch == '\\' {
			slashes := 0
			for i < len(command) && command[i] == '\\' {
				slashes++
				i++
			}
			if i < len(command) && command[i] == '"' {
				current.WriteString(strings.Repeat("\\", slashes/2))
				if slashes%2 == 1 {
					current.WriteByte('"')
				} else {
					inQuotes = !inQuotes
				}
				i++
				continue
			}
			current.WriteString(strings.Repeat("\\", slashes))
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			i++
			continue
		}
		current.WriteByte(ch)
		i++
	}
	flush()
	return fields
}

func splitCommandLine(command string) []string {
	fields := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range command {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		current.WriteRune('\\')
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}
