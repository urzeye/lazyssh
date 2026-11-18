package ssh_config_file

import (
	"os"
	"path/filepath"
	"testing"

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
