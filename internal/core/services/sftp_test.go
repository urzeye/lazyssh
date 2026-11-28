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
		if !slices.Equal(args, []string{"-F", "/tmp/test-config", "--", "prod"}) {
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
