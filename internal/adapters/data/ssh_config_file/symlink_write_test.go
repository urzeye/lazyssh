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

func TestAddServerPreservesSymlinkedConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	realConfigDir := filepath.Join(tempDir, "real")
	linkDir := filepath.Join(tempDir, "linked")
	metadataPath := filepath.Join(tempDir, "metadata.json")

	for _, dir := range []string{realConfigDir, linkDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	realConfigPath := filepath.Join(realConfigDir, "config")
	linkConfigPath := filepath.Join(linkDir, "config")
	configContent := "Host existing\n    HostName 10.0.0.1\n    User root\n"
	if err := os.WriteFile(realConfigPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write real config: %v", err)
	}

	if err := os.Symlink(realConfigPath, linkConfigPath); err != nil {
		t.Fatalf("symlink %s -> %s: %v", linkConfigPath, realConfigPath, err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), linkConfigPath, metadataPath)
	err := repo.AddServer(domain.Server{
		Alias: "new-server",
		Host:  "10.0.0.2",
		User:  "deploy",
		Port:  22,
	})
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	info, err := os.Lstat(linkConfigPath)
	if err != nil {
		t.Fatalf("lstat symlinked config: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config path should remain a symlink, mode = %v", info.Mode())
	}

	realConfigAfter, err := os.ReadFile(realConfigPath)
	if err != nil {
		t.Fatalf("read real config after add: %v", err)
	}
	if !strings.Contains(string(realConfigAfter), "Host new-server") {
		t.Fatalf("real config should contain new server entry\n%s", string(realConfigAfter))
	}

	linkTarget, err := os.Readlink(linkConfigPath)
	if err != nil {
		t.Fatalf("readlink config path: %v", err)
	}
	if linkTarget != realConfigPath {
		t.Fatalf("symlink target = %q, want %q", linkTarget, realConfigPath)
	}
}
