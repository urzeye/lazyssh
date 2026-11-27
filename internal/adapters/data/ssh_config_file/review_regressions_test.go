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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestTildeAndEqualsIncludesResolveEffectiveValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	sshDir := filepath.Join(home, ".ssh")
	confDir := filepath.Join(sshDir, "conf.d")
	if err := os.MkdirAll(confDir, 0o700); err != nil {
		t.Fatal(err)
	}

	rootConfig := filepath.Join(sshDir, "config")
	localConfig := filepath.Join(sshDir, "config.local")
	extraConfig := filepath.Join(confDir, "01-extra.conf")

	rootText := `Include ~/.ssh/config.local
Include=~/.ssh/conf.d/*.conf

Host *
    ServerAliveInterval 60
`
	localText := `Host managed-web
    HostName 10.1.2.3
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
`
	extraText := `Host extra-db
    HostName 10.1.2.4
    User postgres
`
	for path, content := range map[string]string{
		rootConfig:  rootText,
		localConfig: localText,
		extraConfig: extraText,
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewRepositoryWithWritePath(zap.NewNop().Sugar(), rootConfig, localConfig, filepath.Join(home, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("ListServers() returned %d servers, want 2: %+v", len(servers), servers)
	}

	byAlias := make(map[string]domain.Server, len(servers))
	for _, s := range servers {
		byAlias[s.Alias] = s
	}

	web := byAlias["managed-web"]
	if web.Host != "10.1.2.3" || web.User != "deploy" || web.Port != 2222 || web.ServerAliveInterval != "60" || web.Readonly {
		t.Fatalf("managed-web = %+v", web)
	}
	if len(web.IdentityFiles) != 1 || web.IdentityFiles[0] != "~/.ssh/id_ed25519" {
		t.Fatalf("managed-web IdentityFiles = %#v", web.IdentityFiles)
	}

	db := byAlias["extra-db"]
	if db.Host != "10.1.2.4" || db.User != "postgres" || db.ServerAliveInterval != "60" || !db.Readonly {
		t.Fatalf("extra-db = %+v", db)
	}
}

func TestIncludePrecedenceMatchesOpenSSHOrder(t *testing.T) {
	dir := t.TempDir()
	rootConfig := filepath.Join(dir, "config")
	incConfig := filepath.Join(dir, "first.conf")

	if err := os.WriteFile(incConfig, []byte("Host shared\n    HostName from-include.example\n    User incuser\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rootText := "Include " + filepath.ToSlash(incConfig) + "\n\nHost shared\n    HostName from-root.example\n    User rootuser\n"
	if err := os.WriteFile(rootConfig, []byte(rootText), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), rootConfig, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].Host != "from-include.example" || servers[0].User != "incuser" {
		t.Fatalf("servers = %+v, want include-first precedence", servers)
	}

	if sshPath, err := exec.LookPath("ssh"); err == nil {
		out, err := exec.Command(sshPath, "-G", "-F", rootConfig, "shared").CombinedOutput()
		if err != nil {
			t.Fatalf("ssh -G failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "hostname from-include.example") {
			t.Fatalf("OpenSSH disagrees with include precedence:\n%s", out)
		}
	}
}

func TestMatchDirectiveDoesNotPanicOrLeakOptions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config := `Host app
    HostName 10.0.0.10
    User appuser

Match host *.internal
    User leaked-user
    Port 2299

Host db
    HostName 10.0.0.20
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("len(servers) = %d, want 2", len(servers))
	}
	if servers[0].Alias != "app" || servers[0].User != "appuser" || servers[0].Port != 22 {
		t.Fatalf("Match options leaked into app: %+v", servers[0])
	}
	if servers[1].Alias != "db" || servers[1].Host != "10.0.0.20" || servers[1].User != "" {
		t.Fatalf("db parsed unexpectedly: %+v", servers[1])
	}
}

func TestDeprecatedAliasesDoNotDuplicateOrShadow(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	initial := `Host web
    HostName 10.0.0.1
    PubkeyAcceptedAlgorithms ssh-ed25519
    HostbasedAcceptedAlgorithms ssh-ed25519
    KbdInteractiveAuthentication no

Host *
    PubkeyAcceptedKeyTypes ssh-rsa
    HostbasedKeyTypes ssh-rsa
    ChallengeResponseAuthentication yes
`
	if err := os.WriteFile(configPath, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	got := servers[0]
	if got.PubkeyAcceptedAlgorithms != "ssh-ed25519" ||
		got.HostbasedAcceptedAlgorithms != "ssh-ed25519" ||
		got.KbdInteractiveAuthentication != "no" {
		t.Fatalf("deprecated wildcard shadowed canonical host option: %+v", got)
	}

	updated := got
	updated.PubkeyAcceptedAlgorithms = "+ssh-rsa"
	updated.HostbasedAcceptedAlgorithms = "+ssh-rsa"
	updated.KbdInteractiveAuthentication = "yes"
	if err := repo.UpdateServer(got, updated); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, "PubkeyAcceptedAlgorithms") != 1 {
		t.Fatalf("duplicate PubkeyAcceptedAlgorithms written:\n%s", text)
	}
	if strings.Count(text, "HostbasedAcceptedAlgorithms") != 1 {
		t.Fatalf("duplicate HostbasedAcceptedAlgorithms written:\n%s", text)
	}
	if strings.Count(text, "KbdInteractiveAuthentication") != 1 {
		t.Fatalf("duplicate KbdInteractiveAuthentication written:\n%s", text)
	}

	if err := repo.AddServer(domain.Server{
		Alias:                        "fresh",
		Host:                         "10.0.0.2",
		PubkeyAcceptedAlgorithms:     "ssh-ed25519",
		HostbasedAcceptedAlgorithms:  "ssh-ed25519",
		KbdInteractiveAuthentication: "no",
	}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text = string(data)
	if strings.Count(text, "PubkeyAcceptedAlgorithms") != 2 ||
		strings.Count(text, "HostbasedAcceptedAlgorithms") != 2 ||
		strings.Count(text, "KbdInteractiveAuthentication") != 2 {
		t.Fatalf("AddServer wrote duplicate deprecated directives:\n%s", text)
	}
}

func TestManagedModeSymlinkedWritableTargetIsEditable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	dir := t.TempDir()
	realDir := filepath.Join(dir, "dotfiles")
	sshDir := filepath.Join(dir, "ssh")
	for _, d := range []string{realDir, sshDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	realManaged := filepath.Join(realDir, "config.local")
	linkManaged := filepath.Join(sshDir, "config.local")
	if err := os.WriteFile(realManaged, []byte("Host local-box\n    HostName 10.0.0.5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realManaged, linkManaged); err != nil {
		t.Fatal(err)
	}

	rootConfig := filepath.Join(sshDir, "config")
	if err := os.WriteFile(rootConfig, []byte("Include "+realManaged+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepositoryWithWritePath(zap.NewNop().Sugar(), rootConfig, linkManaged, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].Readonly {
		t.Fatalf("symlinked managed target should be writable: %+v", servers)
	}
}

func TestRecordSSHNormalizesSecondaryAliasAndRenameDeduplicates(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	metaPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(configPath, []byte("Host web web-prod\n    HostName 10.0.0.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, metaPath)
	if err := repo.RecordSSH("web-prod"); err != nil {
		t.Fatal(err)
	}

	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].SSHCount != 1 || servers[0].LastSeen.IsZero() {
		t.Fatalf("secondary alias SSH was not merged into primary server metadata: %+v", servers)
	}

	updated := servers[0]
	updated.Alias = "web-prod"
	if err := repo.UpdateServer(servers[0], updated); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Host web-prod web-prod") {
		t.Fatalf("renaming to secondary alias duplicated pattern:\n%s", data)
	}
}

func TestConditionalIncludeExcludesUnmatchedHosts(t *testing.T) {
	dir := t.TempDir()
	rootConfig := filepath.Join(dir, "config")
	conditionalConfig := filepath.Join(dir, "conditional.conf")

	conditionalText := `Host foo-1
    HostName 10.10.0.1

Host bar
    HostName 10.10.0.2
`
	rootText := `Host foo-*
    User scoped-user
    Include ` + filepath.ToSlash(conditionalConfig) + `
`
	if err := os.WriteFile(conditionalConfig, []byte(conditionalText), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootConfig, []byte(rootText), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), rootConfig, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected only guard-matched host foo-1, got %+v", servers)
	}
	if servers[0].Alias != "foo-1" || servers[0].Host != "10.10.0.1" || servers[0].User != "scoped-user" {
		t.Fatalf("unexpected server from conditional include: %+v", servers[0])
	}
}
