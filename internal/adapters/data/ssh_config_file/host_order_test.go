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

func TestAddHostBeforeMatchingDefaults(t *testing.T) {
	for _, tc := range []struct{ name, config, alias, user string }{
		{"all", "Host *\n    User wildcard\n", "lab-3", "specific"},
		{"pattern", "Host other-*\n    User other\n\nHost lab-*\n    User wildcard\n", "lab-3", "specific"},
		{"negated", "Host * !lab-3\n    User wildcard\n", "lab-3", "specific"},
		{"global", "User global\n\nHost *\n    User wildcard\n", "lab-3", "global"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config")
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			repo := NewRepository(zap.NewNop().Sugar(), path, filepath.Join(dir, "meta.json"))
			if err := repo.AddServer(domain.Server{Alias: tc.alias, Host: "192.0.2.1", User: "specific"}); err != nil {
				t.Fatal(err)
			}
			servers, err := repo.ListServers("")
			if err != nil {
				t.Fatal(err)
			}
			if len(servers) != 1 || servers[0].User != tc.user {
				t.Fatalf("effective servers = %+v, want user %s", servers, tc.user)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "global" && !strings.HasPrefix(string(data), "User global\n") {
				t.Fatalf("global options moved: %s", data)
			}
		})
	}
}

func TestManagedAddOrdersOnlyWritableFile(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "config")
	managed := filepath.Join(dir, "config.local")
	rootText := "Include " + filepath.ToSlash(managed) + "\n\nHost *\n    Compression yes\n"
	if err := os.WriteFile(root, []byte(rootText), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("Host *\n    User wildcard\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewRepositoryWithWritePath(zap.NewNop().Sugar(), root, managed, filepath.Join(dir, "meta.json"))
	if err := repo.AddServer(domain.Server{Alias: "newhost", Host: "192.0.2.1", User: "specific"}); err != nil {
		t.Fatal(err)
	}
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].User != "specific" || servers[0].Compression != "yes" {
		t.Fatalf("unexpected effective servers: %+v", servers)
	}
	data, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != rootText {
		t.Fatalf("root config modified: %s", data)
	}
}
