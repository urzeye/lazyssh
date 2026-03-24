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

package ssh_config_file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestConvertCLIForwardToConfigFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic local forward",
			input:    "8080:localhost:80",
			expected: "8080 localhost:80",
		},
		{
			name:     "local forward with bind address",
			input:    "127.0.0.1:8080:localhost:80",
			expected: "127.0.0.1:8080 localhost:80",
		},
		{
			name:     "local forward with wildcard bind",
			input:    "*:8080:localhost:80",
			expected: "*:8080 localhost:80",
		},
		{
			name:     "remote forward",
			input:    "8080:localhost:3000",
			expected: "8080 localhost:3000",
		},
		{
			name:     "remote forward with bind address",
			input:    "0.0.0.0:80:localhost:8080",
			expected: "0.0.0.0:80 localhost:8080",
		},
		{
			name:     "forward with IPv6 address",
			input:    "8080:[2001:db8::1]:80",
			expected: "8080 [2001:db8::1]:80",
		},
		{
			name:     "forward with domain",
			input:    "3306:db.example.com:3306",
			expected: "3306 db.example.com:3306",
		},
		{
			name:     "invalid format - only one colon",
			input:    "8080:localhost",
			expected: "8080:localhost", // returned as-is
		},
		{
			name:     "invalid format - no colons",
			input:    "8080",
			expected: "8080", // returned as-is
		},
	}

	r := &Repository{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.convertCLIForwardToConfigFormat(tt.input)
			if result != tt.expected {
				t.Errorf("convertCLIForwardToConfigFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertConfigForwardToCLIFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic local forward",
			input:    "8080 localhost:80",
			expected: "8080:localhost:80",
		},
		{
			name:     "local forward with bind address",
			input:    "127.0.0.1:8080 localhost:80",
			expected: "127.0.0.1:8080:localhost:80",
		},
		{
			name:     "local forward with wildcard bind",
			input:    "*:8080 localhost:80",
			expected: "*:8080:localhost:80",
		},
		{
			name:     "remote forward",
			input:    "8080 localhost:3000",
			expected: "8080:localhost:3000",
		},
		{
			name:     "remote forward with bind address",
			input:    "0.0.0.0:80 localhost:8080",
			expected: "0.0.0.0:80:localhost:8080",
		},
		{
			name:     "forward with IPv6 address",
			input:    "8080 [2001:db8::1]:80",
			expected: "8080:[2001:db8::1]:80",
		},
		{
			name:     "forward with domain",
			input:    "3306 db.example.com:3306",
			expected: "3306:db.example.com:3306",
		},
		{
			name:     "already in CLI format",
			input:    "8080:localhost:80",
			expected: "8080:localhost:80", // returned as-is
		},
		{
			name:     "no space separator",
			input:    "8080",
			expected: "8080", // returned as-is
		},
	}

	r := &Repository{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.convertConfigForwardToCLIFormat(tt.input)
			if result != tt.expected {
				t.Errorf("convertConfigForwardToCLIFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUpdateServerDoesNotInsertExtraBlankLines(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")

	config := `Host example-host
    HostName example-ip
    Port 22
    User root
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metadataPath)
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("ListServers() returned %d servers, want 1", len(servers))
	}

	current := servers[0]
	updated := current
	updated.User = "admin"

	if err := repo.UpdateServer(current, updated); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	configAfterUpdate, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}

	if strings.Contains(string(configAfterUpdate), "\n\n    HostName") ||
		strings.Contains(string(configAfterUpdate), "\n\n    Port") ||
		strings.Contains(string(configAfterUpdate), "\n\n    User") {
		t.Fatalf("config should not contain blank lines between directives\n%s", string(configAfterUpdate))
	}

	expected := `Host example-host
    HostName example-ip
    Port 22
    User admin
`
	if string(configAfterUpdate) != expected {
		t.Fatalf("updated config = %q, want %q", string(configAfterUpdate), expected)
	}
}
