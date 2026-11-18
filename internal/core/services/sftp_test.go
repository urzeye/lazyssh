package services

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestSFTPUsesSelectedConfigAndRetainsErrors(t *testing.T) {
	svc := NewServerService(zap.NewNop().Sugar(), mockServerRepository{}).(*serverService)
	svc.sftpCommand = func(args ...string) *exec.Cmd {
		if !slices.Equal(args, []string{"-F", "/tmp/test-config", "prod"}) {
			t.Fatalf("SFTP args: %q", args)
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestSSHSessionHelper$")
		cmd.Env = append(os.Environ(), "LAZYSSH_TEST_SESSION_HELPER=fail")
		return cmd
	}
	if err := svc.SFTP("prod"); err == nil || !strings.Contains(err.Error(), "Permission denied") {
		t.Fatalf("SFTP diagnostic lost: %v", err)
	}
}
