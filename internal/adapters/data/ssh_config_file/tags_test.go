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

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestUpdateTagsOnlyWritesMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	text := "# keep formatting\nHost example\n    HostName 192.0.2.1\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(zap.NewNop().Sugar(), path, filepath.Join(dir, "meta.json"))
	server := domain.Server{Alias: "example"}
	for _, tags := range [][]string{{"prod", "web"}, nil} {
		if err := repo.UpdateTags(server, tags); err != nil {
			t.Fatal(err)
		}
		servers, err := repo.ListServers("")
		if err != nil {
			t.Fatal(err)
		}
		if len(servers) != 1 || len(servers[0].Tags) != len(tags) {
			t.Fatalf("tags not persisted: %+v", servers)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != text {
		t.Fatalf("tag edit changed SSH config: %s", data)
	}
	if _, err := os.Stat(path + ".original.backup"); !os.IsNotExist(err) {
		t.Fatalf("tag edit created SSH backup: %v", err)
	}
}

func TestUpdateTagsPropagatesMetadataFailure(t *testing.T) {
	dir := t.TempDir()
	repo := NewRepository(zap.NewNop().Sugar(), filepath.Join(dir, "config"), dir)
	if err := repo.UpdateTags(domain.Server{Alias: "example"}, []string{"prod"}); err == nil {
		t.Fatal("metadata failure ignored")
	}
	readonly := NewReadOnlyRepository(zap.NewNop().Sugar(), filepath.Join(dir, "config"), filepath.Join(dir, "meta.json"))
	if err := readonly.UpdateTags(domain.Server{Alias: "example"}, nil); err == nil {
		t.Fatal("read-only guard bypassed")
	}
}
