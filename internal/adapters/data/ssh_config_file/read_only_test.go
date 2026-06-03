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

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestReadOnlyRepositoryMarksAllServersReadOnly(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	confDir := filepath.Join(tempDir, "config.d")
	includedPath := filepath.Join(confDir, "team.conf")

	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", confDir, err)
	}

	config := `Include config.d/*.conf

Host root-server
    HostName root.example.com
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	includedConfig := `Host included-server
    HostName included.example.com
`
	if err := os.WriteFile(includedPath, []byte(includedConfig), 0o600); err != nil {
		t.Fatalf("write included config: %v", err)
	}

	repo := NewReadOnlyRepository(zap.NewNop().Sugar(), configPath, metadataPath)
	if !repo.IsReadOnly() {
		t.Fatalf("IsReadOnly() = false, want true")
	}

	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("ListServers() returned %d servers, want 2", len(servers))
	}

	for _, server := range servers {
		if !server.Readonly {
			t.Fatalf("%s should be read-only", server.Alias)
		}
		if server.SourceFile == "" {
			t.Fatalf("%s SourceFile should be set", server.Alias)
		}
	}
}

func TestReadOnlyRepositoryRejectsSSHConfigMutations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")

	config := `Host example
    HostName example.com
    User root
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	repo := NewReadOnlyRepository(zap.NewNop().Sugar(), configPath, metadataPath)
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("ListServers() returned %d servers, want 1", len(servers))
	}

	newServer := domain.Server{Alias: "new", Host: "new.example.com"}
	if err := repo.AddServer(newServer); err == nil || !strings.Contains(err.Error(), "read-only mode") {
		t.Fatalf("AddServer() error = %v, want read-only error", err)
	}

	updated := servers[0]
	updated.User = "admin"
	if err := repo.UpdateServer(servers[0], updated); err == nil || !strings.Contains(err.Error(), "read-only mode") {
		t.Fatalf("UpdateServer() error = %v, want read-only error", err)
	}

	if err := repo.DeleteServer(servers[0]); err == nil || !strings.Contains(err.Error(), "read-only mode") {
		t.Fatalf("DeleteServer() error = %v, want read-only error", err)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(after) != config {
		t.Fatalf("config changed in read-only mode:\n%s", string(after))
	}
}
