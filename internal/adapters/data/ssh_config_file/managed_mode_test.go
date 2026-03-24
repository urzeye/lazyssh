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

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestManagedModeMarksOnlyManagedFileEntriesWritable(t *testing.T) {
	tempDir := t.TempDir()
	rootConfigPath := filepath.Join(tempDir, "config")
	managedConfigPath := filepath.Join(tempDir, "config.local")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	confDir := filepath.Join(tempDir, "config.d")
	includePath := filepath.Join(confDir, "01-team.conf")

	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", confDir, err)
	}

	rootConfig := `IgnoreUnknown UseKeychain

Include config.local
Include config.d/*.conf

Host root-server
    HostName root.example.com
    User admin
`
	if err := os.WriteFile(rootConfigPath, []byte(rootConfig), 0o600); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	managedConfig := `Host managed-server
    HostName managed.example.com
    User deploy
`
	if err := os.WriteFile(managedConfigPath, []byte(managedConfig), 0o600); err != nil {
		t.Fatalf("write managed config: %v", err)
	}

	includedConfig := `Host included-server
    HostName included.example.com
    User ops
`
	if err := os.WriteFile(includePath, []byte(includedConfig), 0o600); err != nil {
		t.Fatalf("write included config: %v", err)
	}

	repo := NewRepositoryWithWritePath(zap.NewNop().Sugar(), rootConfigPath, managedConfigPath, metadataPath)
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("ListServers() returned %d servers, want 3", len(servers))
	}

	byAlias := make(map[string]domain.Server, len(servers))
	for _, server := range servers {
		byAlias[server.Alias] = server
	}

	rootServer := byAlias["root-server"]
	if !rootServer.Readonly {
		t.Fatalf("root-server should be read-only in managed mode")
	}
	if rootServer.SourceFile != rootConfigPath {
		t.Fatalf("root-server SourceFile = %q, want %q", rootServer.SourceFile, rootConfigPath)
	}

	managedServer := byAlias["managed-server"]
	if managedServer.Readonly {
		t.Fatalf("managed-server should remain writable")
	}
	if managedServer.SourceFile != managedConfigPath {
		t.Fatalf("managed-server SourceFile = %q, want %q", managedServer.SourceFile, managedConfigPath)
	}

	includedServer := byAlias["included-server"]
	if !includedServer.Readonly {
		t.Fatalf("included-server should be read-only")
	}
	if includedServer.SourceFile != includePath {
		t.Fatalf("included-server SourceFile = %q, want %q", includedServer.SourceFile, includePath)
	}
}

func TestManagedModeWritesMutationsOnlyToManagedConfig(t *testing.T) {
	tempDir := t.TempDir()
	rootConfigPath := filepath.Join(tempDir, "config")
	managedConfigPath := filepath.Join(tempDir, "config.local")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	confDir := filepath.Join(tempDir, "config.d")
	includePath := filepath.Join(confDir, "01-team.conf")

	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", confDir, err)
	}

	rootConfig := `Include config.local
Include config.d/*.conf

Host root-server
    HostName root.example.com
    User admin
`
	if err := os.WriteFile(rootConfigPath, []byte(rootConfig), 0o600); err != nil {
		t.Fatalf("write root config: %v", err)
	}

	managedConfig := `Host managed-server
    HostName managed.example.com
    User deploy
    Port 22
`
	if err := os.WriteFile(managedConfigPath, []byte(managedConfig), 0o600); err != nil {
		t.Fatalf("write managed config: %v", err)
	}

	includedConfig := `Host included-server
    HostName included.example.com
    User ops
`
	if err := os.WriteFile(includePath, []byte(includedConfig), 0o600); err != nil {
		t.Fatalf("write included config: %v", err)
	}

	repo := NewRepositoryWithWritePath(zap.NewNop().Sugar(), rootConfigPath, managedConfigPath, metadataPath)

	newServer := domain.Server{
		Alias: "new-server",
		Host:  "new.example.com",
		User:  "root",
		Port:  2222,
	}
	if err := repo.AddServer(newServer); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	rootAfterAdd, err := os.ReadFile(rootConfigPath)
	if err != nil {
		t.Fatalf("read root config after add: %v", err)
	}
	if strings.Contains(string(rootAfterAdd), "new-server") {
		t.Fatalf("root config should not be mutated by AddServer()\n%s", string(rootAfterAdd))
	}

	managedAfterAdd, err := os.ReadFile(managedConfigPath)
	if err != nil {
		t.Fatalf("read managed config after add: %v", err)
	}
	if !strings.Contains(string(managedAfterAdd), "Host new-server") {
		t.Fatalf("managed config should contain new-server after AddServer()\n%s", string(managedAfterAdd))
	}

	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	var managedServer domain.Server
	for _, server := range servers {
		if server.Alias == "managed-server" {
			managedServer = server
			break
		}
	}
	if managedServer.Alias == "" {
		t.Fatalf("managed-server not found")
	}
	if managedServer.Readonly {
		t.Fatalf("managed-server should remain writable")
	}

	updatedManagedServer := managedServer
	updatedManagedServer.Port = 2200
	if err := repo.UpdateServer(managedServer, updatedManagedServer); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	managedAfterUpdate, err := os.ReadFile(managedConfigPath)
	if err != nil {
		t.Fatalf("read managed config after update: %v", err)
	}
	if !strings.Contains(string(managedAfterUpdate), "Port 2200") {
		t.Fatalf("managed config should contain updated port after UpdateServer()\n%s", string(managedAfterUpdate))
	}

	if err := repo.DeleteServer(newServer); err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}

	managedAfterDelete, err := os.ReadFile(managedConfigPath)
	if err != nil {
		t.Fatalf("read managed config after delete: %v", err)
	}
	if strings.Contains(string(managedAfterDelete), "Host new-server") {
		t.Fatalf("managed config should not contain new-server after DeleteServer()\n%s", string(managedAfterDelete))
	}

	rootAfterDelete, err := os.ReadFile(rootConfigPath)
	if err != nil {
		t.Fatalf("read root config after delete: %v", err)
	}
	if string(rootAfterDelete) != string(rootAfterAdd) {
		t.Fatalf("root config should stay unchanged in managed mode\nbefore:\n%s\nafter:\n%s", string(rootAfterAdd), string(rootAfterDelete))
	}
}
