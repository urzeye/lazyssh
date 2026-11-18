package ui

import (
	"testing"
	"time"

	"github.com/urzeye/lazyssh/internal/core/domain"
)

func TestCloneServerResetsMetadataAndCopiesSlices(t *testing.T) {
	source := domain.Server{Alias: "prod", Aliases: []string{"prod", "other"}, Host: "192.0.2.1", User: "root", Port: 2222, Readonly: true, SourceFile: "external", LastSeen: time.Now(), PinnedAt: time.Now(), SSHCount: 4, IdentityFiles: []string{"key"}, Tags: []string{"prod"}, LocalForward: []string{"8080:localhost:80"}, ProxyJump: "jump", IPQoS: "lowdelay"}
	clone := cloneServer(source, []domain.Server{{Aliases: []string{"prod_copy"}}})
	if clone.Alias != "prod_copy_2" || clone.Readonly || clone.SourceFile != "" || !clone.LastSeen.IsZero() || !clone.PinnedAt.IsZero() || clone.SSHCount != 0 || len(clone.Aliases) != 0 {
		t.Fatalf("bad clone metadata: %+v", clone)
	}
	clone.Tags[0] = "dev"
	clone.IdentityFiles[0] = "different"
	clone.LocalForward[0] = "different"
	if source.Tags[0] != "prod" || source.IdentityFiles[0] != "key" || source.LocalForward[0] != "8080:localhost:80" {
		t.Fatal("clone shares mutable slices")
	}
	if clone.ProxyJump != source.ProxyJump || clone.IPQoS != source.IPQoS {
		t.Fatal("clone lost connection settings")
	}
}

func TestPrefilledAddFormCreatesNewServer(t *testing.T) {
	seed := domain.Server{Alias: "clone", Host: "192.0.2.1", Port: 22, IPQoS: "lowdelay", CheckHostIP: "yes", FingerprintHash: "sha256", GatewayPorts: "yes", CanonicalDomains: "example.com"}
	form := NewServerForm(ServerFormAdd, &seed).SetVersionInfo("test", "test")
	data := form.getDefaultValues()
	if data.Alias != "clone" || data.Host != seed.Host {
		t.Fatal("Add form not prefilled")
	}
	got := form.dataToServer(data)
	if got.IPQoS != seed.IPQoS || got.CheckHostIP != seed.CheckHostIP || got.CanonicalDomains != seed.CanonicalDomains {
		t.Fatalf("prefill lost fields: %+v", got)
	}
	called := false
	form.OnSave(func(server domain.Server, original *domain.Server) {
		called = true
		if server.IPQoS != seed.IPQoS || server.CheckHostIP != seed.CheckHostIP || server.FingerprintHash != seed.FingerprintHash || server.GatewayPorts != seed.GatewayPorts || server.CanonicalDomains != seed.CanonicalDomains {
			t.Fatalf("saved clone lost fields: %+v", server)
		}
		if original != nil {
			t.Fatal("clone treated as update")
		}
	})
	if !form.handleSave() || !called {
		t.Fatal("prefilled form did not save")
	}
}
