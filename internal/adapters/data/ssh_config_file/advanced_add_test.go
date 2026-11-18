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
	"strings"
	"testing"

	"github.com/kevinburke/ssh_config"
	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestAddServerRetainsAdvancedOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(zap.NewNop().Sugar(), path, filepath.Join(dir, "meta.json"))
	repo.(*Repository).sshCommand = capabilityCommand(t, true)
	server := domain.Server{Alias: "clone", Host: "192.0.2.1", Port: 2222, SessionType: "none", AddressFamily: "inet", BindAddress: "127.0.0.1", ExitOnForwardFailure: "yes", IPQoS: "lowdelay", CanonicalizeHostname: "no", CanonicalDomains: "example.com", GatewayPorts: "yes", CheckHostIP: "yes", FingerprintHash: "sha256"}
	if err := repo.AddServer(server); err != nil {
		t.Fatal(err)
	}
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("wrong hosts: %+v", servers)
	}
	got := servers[0]
	if got.IPQoS != server.IPQoS || got.SessionType != server.SessionType || got.BindAddress != server.BindAddress || got.AddressFamily != server.AddressFamily || got.ExitOnForwardFailure != server.ExitOnForwardFailure || got.CanonicalizeHostname != server.CanonicalizeHostname || got.CanonicalDomains != server.CanonicalDomains || got.GatewayPorts != server.GatewayPorts || got.CheckHostIP != server.CheckHostIP || got.FingerprintHash != server.FingerprintHash {
		t.Fatalf("new host lost advanced options: %+v", got)
	}
}

func TestProxyUpdatesPreserveExistingOrder(t *testing.T) {
	repo := &Repository{}
	for _, config := range []string{"Host host\n    HostName 192.0.2.1\n", "Host host\n    ProxyCommand original\n    ProxyJump original\n"} {
		cfg, err := ssh_config.Decode(strings.NewReader(config))
		if err != nil {
			t.Fatal(err)
		}
		host := cfg.Hosts[len(cfg.Hosts)-1]
		repo.updateHostNodes(host, domain.Server{Alias: "host", Host: "192.0.2.1", ProxyJump: "bastion", ProxyCommand: "netcat"})
		text := host.String()
		jump, command := strings.Index(text, "ProxyJump "), strings.Index(text, "ProxyCommand ")
		commandWasFirst := strings.Contains(config, "ProxyCommand")
		if (command < jump) != commandWasFirst {
			t.Fatalf("proxy priority changed:\n%s", text)
		}
	}
}

func TestNewHostProxyPrecedenceIsDeterministic(t *testing.T) {
	repo := &Repository{}
	server := domain.Server{Alias: "example.invalid", Host: "192.0.2.1", ProxyJump: "bastion", ProxyCommand: "nc %h %p"}
	var config string
	for i := 0; i < 20; i++ {
		config = repo.createHostFromServer(server).String()
		jump, command := strings.Index(config, "ProxyJump "), strings.Index(config, "ProxyCommand ")
		if jump < 0 || command < 0 || jump > command {
			t.Fatalf("new-host proxy priority changed:\n%s", config)
		}
	}
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable for effective-config assertion")
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(ssh, "-G", "-F", path, server.Alias).CombinedOutput()
	if err != nil {
		t.Fatalf("ssh -G: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "proxyjump bastion") {
		t.Fatalf("OpenSSH did not retain jump priority:\n%s", output)
	}
}
