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

func TestUpdateAndDeleteHostPreserveFollowingMatchBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	initial := `Host app
    HostName 10.0.0.1
    User appuser
    ServerAliveInterval 30
    IdentityFile ~/.ssh/app_key

Match host *.internal
    User conditional-user
    Port 2299
    ServerAliveInterval 15
    IdentityFile ~/.ssh/internal_key
`
	if err := os.WriteFile(configPath, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), configPath, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil || len(servers) != 1 {
		t.Fatalf("ListServers() = %+v, %v", servers, err)
	}

	updated := servers[0]
	updated.User = "updated-user"
	updated.Port = 2222
	updated.ServerAliveInterval = ""
	updated.ForwardAgent = "yes"
	updated.IdentityFiles = nil
	if err := repo.UpdateServer(servers[0], updated); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	matchSection := "Match host *.internal\n    User conditional-user\n    Port 2299\n    ServerAliveInterval 15\n    IdentityFile ~/.ssh/internal_key\n"
	if !strings.HasSuffix(text, matchSection) {
		t.Fatalf("UpdateServer mutated or misordered trailing Match section:\n%s", text)
	}
	matchIdx := strings.Index(text, "Match host *.internal")
	forwardIdx := strings.Index(text, "ForwardAgent yes")
	if forwardIdx < 0 || forwardIdx > matchIdx {
		t.Fatalf("new host option was inserted after Match boundary:\n%s", text)
	}
	if strings.Contains(text[:matchIdx], "ServerAliveInterval") {
		t.Fatalf("cleared host option still present before Match boundary:\n%s", text)
	}

	if err := repo.DeleteServer(updated); err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), matchSection) {
		t.Fatalf("DeleteServer dropped trailing Match section:\n%s", string(data))
	}
}

func TestMatchAllResumesUnconditionalOptionsAndIncludes(t *testing.T) {
	dir := t.TempDir()
	rootConfig := filepath.Join(dir, "config")
	skippedConfig := filepath.Join(dir, "skipped.conf")
	childConfig := filepath.Join(dir, "child.conf")
	outerChildConfig := filepath.Join(dir, "outer-child.conf")
	scopedMidConfig := filepath.Join(dir, "scoped-mid.conf")

	if err := os.WriteFile(skippedConfig, []byte("Host skipped-box\n    HostName 10.0.0.99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(childConfig, []byte("Host child-box\n    HostName 10.2.3.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outerChildConfig, []byte("Host foo-1\n    HostName 10.8.8.8\n\nHost bar\n    HostName 10.9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scopedMidConfig, []byte("Match all\n    Include "+filepath.ToSlash(outerChildConfig)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	rootText := `Host app
    HostName 10.0.0.1

Match host *.other
    User conditional-user
    Include ` + filepath.ToSlash(skippedConfig) + `

Match all
    ServerAliveInterval 60
    Include ` + filepath.ToSlash(childConfig) + `

Host foo-*
    Include ` + filepath.ToSlash(scopedMidConfig) + `
`
	if err := os.WriteFile(rootConfig, []byte(rootText), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(zap.NewNop().Sugar(), rootConfig, filepath.Join(dir, "meta.json"))
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	byAlias := make(map[string]domain.Server, len(servers))
	for _, s := range servers {
		byAlias[s.Alias] = s
	}

	if _, exists := byAlias["skipped-box"]; exists {
		t.Fatalf("conditional Match include should not have been traversed: %+v", servers)
	}
	if _, exists := byAlias["bar"]; exists {
		t.Fatalf("outer Host guard should still exclude unmatched host under nested Match all: %+v", servers)
	}

	app, ok := byAlias["app"]
	if !ok || app.Host != "10.0.0.1" || app.User != "" || app.ServerAliveInterval != "60" {
		t.Fatalf("app = %+v, want Match all option without conditional Match leak", app)
	}
	child, ok := byAlias["child-box"]
	if !ok || child.Host != "10.2.3.4" || child.ServerAliveInterval != "60" {
		t.Fatalf("child-box = %+v, want discovered under Match all with ServerAliveInterval 60", child)
	}
	foo, ok := byAlias["foo-1"]
	if !ok || foo.Host != "10.8.8.8" || foo.ServerAliveInterval != "60" {
		t.Fatalf("foo-1 = %+v, want discovered under outer guard + Match all", foo)
	}
}

func TestWritableDuplicateAliasTakesPrecedenceOverEarlierReadOnlyInclude(t *testing.T) {
	dir := t.TempDir()
	rootConfig := filepath.Join(dir, "config")
	childConfig := filepath.Join(dir, "child.conf")

	childText := `Host app-ro app
    HostName 10.50.0.1
`
	rootText := `Include ` + filepath.ToSlash(childConfig) + `

Host app
    User root-override
`
	if err := os.WriteFile(childConfig, []byte(childText), 0o600); err != nil {
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
	if len(servers) != 2 {
		t.Fatalf("expected app-ro (readonly) and app (writable), got %+v", servers)
	}

	byAlias := make(map[string]domain.Server, len(servers))
	for _, s := range servers {
		byAlias[s.Alias] = s
	}

	ro, ok := byAlias["app-ro"]
	if !ok || !ro.Readonly || ro.SourceFile != childConfig || ro.Host != "10.50.0.1" {
		t.Fatalf("app-ro = %+v, want readonly server from child config", ro)
	}

	rw, ok := byAlias["app"]
	if !ok || rw.Readonly || rw.SourceFile != rootConfig || rw.Host != "10.50.0.1" || rw.User != "root-override" {
		t.Fatalf("app = %+v, want writable server with merged effective options", rw)
	}

	updated := rw
	updated.User = "deploy"
	if err := repo.UpdateServer(rw, updated); err != nil {
		t.Fatalf("UpdateServer() on writable duplicate alias failed: %v", err)
	}

	after, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() after update error = %v", err)
	}
	var updatedApp domain.Server
	for _, s := range after {
		if s.Alias == "app" {
			updatedApp = s
		}
	}
	if updatedApp.User != "deploy" || updatedApp.Host != "10.50.0.1" || updatedApp.Readonly {
		t.Fatalf("updated app = %+v, want User=deploy Host=10.50.0.1 Readonly=false", updatedApp)
	}
}
