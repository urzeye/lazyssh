package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestPingUsesSelectedSSHConfig(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	svc := NewServerService(zap.NewNop().Sugar(), mockServerRepository{}).(*serverService)
	svc.sshCommand = func(args ...string) *exec.Cmd {
		want := []string{"-G", "-F", "/tmp/test-config", "example"}
		if !slices.Equal(args, want) {
			t.Fatalf("ssh args = %q, want %q", args, want)
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestSSHSessionHelper$")
		cmd.Env = append(os.Environ(), "LAZYSSH_TEST_SESSION_HELPER=resolve", fmt.Sprintf("LAZYSSH_TEST_PORT=%d", port))
		return cmd
	}
	up, _, err := svc.Ping(domain.Server{Alias: "example", Host: "wrong.invalid", Port: 1})
	if !up || err != nil {
		t.Fatalf("Ping() = %v, %v", up, err)
	}
}
