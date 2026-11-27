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
	"slices"
	"testing"
)

func TestParseSSHProcessLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantAlias   string
		wantDest    string
		wantForward []string
	}{
		{
			name:      "simple alias",
			line:      "12345 ssh prod",
			wantAlias: "prod",
			wantDest:  "prod",
		},
		{
			name:      "config file option",
			line:      "12346 ssh -F /tmp/config prod",
			wantAlias: "prod",
			wantDest:  "prod",
		},
		{
			name:        "local forward",
			line:        "12347 ssh -L 8080:localhost:80 prod",
			wantAlias:   "prod",
			wantDest:    "prod",
			wantForward: []string{"8080:localhost:80"},
		},
		{
			name:        "compact forward and user destination",
			line:        "12348 /usr/bin/ssh -N -D1080 ops@example.com",
			wantAlias:   "example.com",
			wantDest:    "ops@example.com",
			wantForward: []string{"1080"},
		},
		{
			name:      "quoted destination",
			line:      "12349 ssh -o StrictHostKeyChecking=no 'prod web'",
			wantAlias: "prod web",
			wantDest:  "prod web",
		},
		{
			name:      "bind interface option",
			line:      "12350 ssh -B en0 prod",
			wantAlias: "prod",
			wantDest:  "prod",
		},
		{
			name:      "escape char option",
			line:      "12351 ssh -e none prod",
			wantAlias: "prod",
			wantDest:  "prod",
		},
		{
			name:      "unbracketed ipv6 destination",
			line:      "12352 ssh user@2001:db8::1",
			wantAlias: "2001:db8::1",
			wantDest:  "user@2001:db8::1",
		},
		{
			name:        "combined short flags with local forward",
			line:        "12353 ssh -NfL 8080:localhost:80 prod",
			wantAlias:   "prod",
			wantDest:    "prod",
			wantForward: []string{"8080:localhost:80"},
		},
		{
			name:        "combined short flags with attached dynamic forward",
			line:        "12354 ssh -fND1080 prod",
			wantAlias:   "prod",
			wantDest:    "prod",
			wantForward: []string{"1080"},
		},
		{
			name:      "combined short flags with port option",
			line:      "12355 ssh -4p 2222 prod",
			wantAlias: "prod",
			wantDest:  "prod",
		},
		{
			name:      "ssh uri without user",
			line:      "12356 ssh ssh://example.com:2222",
			wantAlias: "example.com",
			wantDest:  "ssh://example.com:2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSSHProcessLine(tt.line)
			if !ok {
				t.Fatalf("parseSSHProcessLine() ok = false, want true")
			}
			if got.Alias != tt.wantAlias {
				t.Fatalf("Alias = %q, want %q", got.Alias, tt.wantAlias)
			}
			if got.Destination != tt.wantDest {
				t.Fatalf("Destination = %q, want %q", got.Destination, tt.wantDest)
			}
			if !slices.Equal(got.Forwards, tt.wantForward) {
				t.Fatalf("Forwards = %#v, want %#v", got.Forwards, tt.wantForward)
			}
		})
	}
}

func TestParseSSHProcessPreservesArgvBoundaries(t *testing.T) {
	got, ok := parseSSHProcess(processInfo{
		PID: 12345,
		Args: []string{
			"ssh",
			"-F",
			"/tmp/ssh configs/config",
			"prod",
		},
		Command: "ssh -F /tmp/ssh configs/config prod",
	})
	if !ok {
		t.Fatalf("parseSSHProcess() ok = false, want true")
	}
	if got.Destination != "prod" {
		t.Fatalf("Destination = %q, want prod", got.Destination)
	}
}

func TestParseSSHProcessLineRejectsNonSessions(t *testing.T) {
	tests := []string{
		"",
		"not-a-pid ssh prod",
		"12345 ssh-agent",
		"12346 scp file prod:/tmp",
		"12347 ssh",
		"12348 ssh -G prod",
		"12349 ssh -V",
	}

	for _, line := range tests {
		if got, ok := parseSSHProcessLine(line); ok {
			t.Fatalf("parseSSHProcessLine(%q) = %#v, true; want false", line, got)
		}
	}
}

func TestParseWindowsSSHCommandLine(t *testing.T) {
	cmdLine := `"C:\Windows\System32\OpenSSH\ssh.exe" -F "C:\Users\Alice Smith\.ssh\config" -L 8080:localhost:80 prod`
	args := splitWindowsCommandLine(cmdLine)
	session, ok := parseSSHProcess(processInfo{
		PID:     4242,
		Args:    args,
		Command: cmdLine,
	})
	if !ok {
		t.Fatalf("parseSSHProcess() ok = false for Windows ssh.exe; args=%q", args)
	}
	if session.Alias != "prod" || session.Destination != "prod" || !slices.Equal(session.Forwards, []string{"8080:localhost:80"}) {
		t.Fatalf("unexpected Windows session: %+v", session)
	}
}
