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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestSSHCapabilityHelper(t *testing.T) {
	switch os.Getenv("LAZYSSH_CAPABILITY_RESULT") {
	case "supported":
		os.Exit(0)
	case "unsupported":
		fmt.Fprintln(os.Stderr, "command-line: Bad configuration option: sessiontype")
		os.Exit(255)
	}
}

func capabilityCommand(t *testing.T, supported bool) func(context.Context, ...string) *exec.Cmd {
	t.Helper()
	return func(ctx context.Context, args ...string) *exec.Cmd {
		t.Helper()
		if len(args) != 6 || !slices.Equal(args[:2], []string{"-G", "-F"}) || args[3] != "-o" || !strings.HasPrefix(args[4], "SessionType=") || args[5] != "lazyssh-capability.invalid" {
			t.Fatalf("unexpected capability probe: %q", args)
		}
		data, err := os.ReadFile(args[2])
		if err != nil || len(data) != 0 || !strings.HasPrefix(filepath.Base(args[2]), "lazyssh-ssh-probe-") {
			t.Fatalf("probe did not use an isolated empty config: %q, %v", data, err)
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSSHCapabilityHelper$")
		result := "unsupported"
		if supported {
			result = "supported"
		}
		cmd.Env = append(os.Environ(), "LAZYSSH_CAPABILITY_RESULT="+result)
		return cmd
	}
}

func TestSessionTypeCheckedBeforeConfigAndMetadataWrites(t *testing.T) {
	for _, supported := range []bool{false, true} {
		for _, value := range []string{"none", "subsystem", "default"} {
			for _, operation := range []string{"add", "update"} {
				t.Run(fmt.Sprintf("%t/%s/%s", supported, value, operation), func(t *testing.T) {
					dir := t.TempDir()
					config, metadata := filepath.Join(dir, "config"), filepath.Join(dir, "metadata.json")
					if err := os.WriteFile(config, nil, 0o600); err != nil {
						t.Fatal(err)
					}
					repo := NewRepository(zap.NewNop().Sugar(), config, metadata).(*Repository)
					original := domain.Server{Alias: "existing", Host: "192.0.2.1", Tags: []string{"keep"}}
					if err := repo.AddServer(original); err != nil {
						t.Fatal(err)
					}
					beforeConfig := readCapabilityFixture(t, config)
					beforeMetadata := readCapabilityFixture(t, metadata)
					repo.sshCommand = capabilityCommand(t, supported)
					changed := original
					changed.SessionType, changed.Host, changed.Tags = value, "192.0.2.2", []string{"changed"}
					var err error
					if operation == "add" {
						changed.Alias = "new"
						err = repo.AddServer(changed)
					} else {
						err = repo.UpdateServer(original, changed)
					}
					if supported {
						if err != nil {
							t.Fatal(err)
						}
						servers, err := repo.ListServers(changed.Alias)
						if err != nil || len(servers) != 1 || servers[0].SessionType != value {
							t.Fatalf("supported option lost: %+v, %v", servers, err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), "OpenSSH 8.7+") {
						t.Fatalf("missing compatibility error: %v", err)
					}
					afterConfig := readCapabilityFixture(t, config)
					afterMetadata := readCapabilityFixture(t, metadata)
					if !bytes.Equal(beforeConfig, afterConfig) || !bytes.Equal(beforeMetadata, afterMetadata) {
						t.Fatal("rejected option changed config or metadata")
					}
				})
			}
		}
	}
}

func readCapabilityFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSessionTypeProbeUsesInstalledSSH(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable")
	}
	clients := []string{ssh}
	if runtime.GOOS == "windows" {
		native := filepath.Join(os.Getenv("WINDIR"), "System32", "OpenSSH", "ssh.exe")
		if _, err := os.Stat(native); err == nil && !strings.EqualFold(ssh, native) {
			clients = append(clients, native)
		}
	}
	for _, client := range clients {
		t.Run(client, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "config")
			const original = "Host existing\n    HostName 192.0.2.1\n"
			if err := os.WriteFile(config, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			repo := NewRepository(zap.NewNop().Sugar(), config, filepath.Join(dir, "metadata.json"))
			repo.(*Repository).sshCommand = func(ctx context.Context, args ...string) *exec.Cmd { return exec.CommandContext(ctx, client, args...) }
			_, probeErr := exec.Command(client, "-G", "-F", config, "-o", "SessionType=none", "lazyssh-capability.invalid").CombinedOutput()
			err := repo.AddServer(domain.Server{Alias: "new", Host: "192.0.2.2", SessionType: "none"})
			if (err == nil) != (probeErr == nil) {
				t.Fatalf("save result %v disagrees with installed SSH: %v", err, probeErr)
			}
			if probeErr != nil {
				contents := readCapabilityFixture(t, config)
				if string(contents) != original {
					t.Fatal("unsupported SessionType modified existing config")
				}
			}
			if output, err := exec.Command(client, "-G", "-F", config, "existing").CombinedOutput(); err != nil {
				t.Fatalf("existing host broken: %v\n%s", err, output)
			}
		})
	}
}
