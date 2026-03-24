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
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestListServersLoadsIncludedConfigsRecursively(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	confDir := filepath.Join(tempDir, "config.d")
	nestedDir := filepath.Join(confDir, "nested")

	for _, dir := range []string{confDir, nestedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	mainConfig := "IgnoreUnknown UseKeychain\n\nInclude config.d/*.conf\n\nHost main-server\n    HostName main.example.com\n    User root\n"
	if err := os.WriteFile(configPath, []byte(mainConfig), 0o600); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	includedConfig := "Host included-server\n    HostName 10.0.0.1\n    User deploy\n\nHost main-server\n    HostName shadowed.example.com\n\nInclude nested/*.conf\n"
	if err := os.WriteFile(filepath.Join(confDir, "01-work.conf"), []byte(includedConfig), 0o600); err != nil {
		t.Fatalf("write included config: %v", err)
	}

	nestedConfig := "Host nested-server alias-nested\n    HostName 10.0.0.2\n    User ops\n"
	nestedPath := filepath.Join(nestedDir, "02-nested.conf")
	if err := os.WriteFile(nestedPath, []byte(nestedConfig), 0o600); err != nil {
		t.Fatalf("write nested config: %v", err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metadataPath)
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

	mainServer, ok := byAlias["main-server"]
	if !ok {
		t.Fatalf("main-server not found")
	}
	if mainServer.Host != "main.example.com" {
		t.Fatalf("main-server Host = %q, want %q", mainServer.Host, "main.example.com")
	}
	if mainServer.Readonly {
		t.Fatalf("main-server should not be read-only")
	}
	if mainServer.SourceFile != configPath {
		t.Fatalf("main-server SourceFile = %q, want %q", mainServer.SourceFile, configPath)
	}

	includedServer, ok := byAlias["included-server"]
	if !ok {
		t.Fatalf("included-server not found")
	}
	if !includedServer.Readonly {
		t.Fatalf("included-server should be read-only")
	}
	if includedServer.SourceFile != filepath.Join(confDir, "01-work.conf") {
		t.Fatalf("included-server SourceFile = %q", includedServer.SourceFile)
	}

	nestedServer, ok := byAlias["nested-server"]
	if !ok {
		t.Fatalf("nested-server not found")
	}
	if !nestedServer.Readonly {
		t.Fatalf("nested-server should be read-only")
	}
	if nestedServer.SourceFile != nestedPath {
		t.Fatalf("nested-server SourceFile = %q, want %q", nestedServer.SourceFile, nestedPath)
	}
	if len(nestedServer.Aliases) != 2 || nestedServer.Aliases[1] != "alias-nested" {
		t.Fatalf("nested-server Aliases = %#v, want secondary alias preserved", nestedServer.Aliases)
	}
}

func TestAddServerRejectsAliasFromIncludedConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	confDir := filepath.Join(tempDir, "config.d")

	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", confDir, err)
	}

	mainConfig := "Include config.d/*.conf\n"
	if err := os.WriteFile(configPath, []byte(mainConfig), 0o600); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	includedConfig := "Host shared-alias\n    HostName 10.0.0.10\n"
	if err := os.WriteFile(filepath.Join(confDir, "01.conf"), []byte(includedConfig), 0o600); err != nil {
		t.Fatalf("write included config: %v", err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metadataPath)
	err := repo.AddServer(domain.Server{
		Alias: "shared-alias",
		Host:  "new.example.com",
		User:  "root",
		Port:  22,
	})
	if err == nil {
		t.Fatalf("AddServer() error = nil, want duplicate alias error")
	}
}
