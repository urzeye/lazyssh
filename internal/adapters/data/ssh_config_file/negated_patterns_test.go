package ssh_config_file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestSavePreservesNegatedPatterns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	original := "Host !bastion\n    User nobody\n\nHost * !lab-? !excluded\n    User other\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(zap.NewNop().Sugar(), path, filepath.Join(dir, "metadata.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Fatalf("negated patterns listed as servers: %+v", servers)
	}
	if err := repo.AddServer(domain.Server{Alias: "excluded", Host: "192.0.2.2"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), original) {
		t.Fatalf("existing negated patterns changed:\n%s", data)
	}
}

func TestIncludedNegatedPatternsAreNotAliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("Include included.conf\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "included.conf"), []byte("Host !excluded\n    User nobody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(zap.NewNop().Sugar(), path, filepath.Join(dir, "metadata.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Fatalf("negated include pattern listed as server: %+v", servers)
	}
}
