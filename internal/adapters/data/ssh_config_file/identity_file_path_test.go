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

func TestAddServerStoresIdentityFileRelativeToHome(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configPath := filepath.Join(tempHome, ".ssh", "config")
	metadataPath := filepath.Join(tempHome, "metadata.json")
	keyPath := filepath.Join(tempHome, ".ssh", "id_ed25519")

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metadataPath)
	err := repo.AddServer(domain.Server{
		Alias:         "portable-key",
		Host:          "10.0.0.20",
		User:          "root",
		Port:          22,
		IdentityFiles: []string{keyPath},
	})
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(configContent), "IdentityFile ~/.ssh/id_ed25519") {
		t.Fatalf("config should store IdentityFile with ~/ prefix\n%s", string(configContent))
	}
	if strings.Contains(string(configContent), keyPath) {
		t.Fatalf("config should not store absolute IdentityFile path\n%s", string(configContent))
	}
}

func TestUpdateServerStoresIdentityFileRelativeToHome(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configPath := filepath.Join(tempHome, ".ssh", "config")
	metadataPath := filepath.Join(tempHome, "metadata.json")
	oldKeyPath := filepath.Join(tempHome, ".ssh", "id_old")
	newKeyPath := filepath.Join(tempHome, ".ssh", "id_new")

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	for _, keyPath := range []string{oldKeyPath, newKeyPath} {
		if err := os.WriteFile(keyPath, []byte("test"), 0o600); err != nil {
			t.Fatalf("write key file %s: %v", keyPath, err)
		}
	}

	config := "Host portable-key\n    HostName 10.0.0.20\n    User root\n    IdentityFile ~/.ssh/id_old\n"
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
	updated.IdentityFiles = []string{newKeyPath}

	if err := repo.UpdateServer(current, updated); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(configContent), "IdentityFile ~/.ssh/id_new") {
		t.Fatalf("config should store updated IdentityFile with ~/ prefix\n%s", string(configContent))
	}
	if strings.Contains(string(configContent), newKeyPath) {
		t.Fatalf("config should not store absolute updated IdentityFile path\n%s", string(configContent))
	}
}
