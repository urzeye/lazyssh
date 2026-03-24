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

func TestIncludeDirectivesPreservedWhenMutatingMainConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")

	initialConfig := `IgnoreUnknown UseKeychain

Include ~/.orbstack/ssh/config
Include config.d/*.conf

Host main-server
    HostName main.example.com
    User root
    Port 22
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0o600); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metadataPath)

	newServer := domain.Server{
		Alias: "new-server",
		Host:  "new.example.com",
		User:  "deploy",
		Port:  2222,
	}

	if err := repo.AddServer(newServer); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}
	assertConfigKeepsGlobalDirectives(t, configPath)

	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	var mainServer domain.Server
	for _, server := range servers {
		if server.Alias == "main-server" {
			mainServer = server
			break
		}
	}
	if mainServer.Alias == "" {
		t.Fatalf("main-server not found after AddServer")
	}

	updatedServer := mainServer
	updatedServer.Port = 2200
	if err := repo.UpdateServer(mainServer, updatedServer); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}
	assertConfigKeepsGlobalDirectives(t, configPath)

	if err := repo.DeleteServer(newServer); err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	assertConfigKeepsGlobalDirectives(t, configPath)
}

func assertConfigKeepsGlobalDirectives(t *testing.T, configPath string) {
	t.Helper()

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	configText := string(content)
	required := []string{
		"IgnoreUnknown UseKeychain",
		"Include ~/.orbstack/ssh/config",
		"Include config.d/*.conf",
	}

	for _, directive := range required {
		if !strings.Contains(configText, directive) {
			t.Fatalf("config lost directive %q\n%s", directive, configText)
		}
	}
}
