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
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func (r *Repository) validateSessionType(value string) error {
	if value == "" {
		return nil
	}
	switch value {
	case "none", "subsystem", "default":
	default:
		return fmt.Errorf("invalid SessionType %q", value)
	}
	emptyConfig, err := os.CreateTemp("", "lazyssh-ssh-probe-*")
	if err != nil {
		return fmt.Errorf("create isolated SSH config probe: %w", err)
	}
	defer func() {
		if err := os.Remove(emptyConfig.Name()); err != nil {
			r.logger.Warnw("failed to remove isolated SSH config probe", "error", err)
		}
	}()
	if err := emptyConfig.Close(); err != nil {
		return fmt.Errorf("close isolated SSH config probe: %w", err)
	}
	// Never load the user's config or connect: unknown directives are fatal
	// even in inactive Host blocks on older clients.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := r.sshCommand(ctx, "-G", "-F", emptyConfig.Name(), "-o", "SessionType="+value, "lazyssh-capability.invalid")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return fmt.Errorf("cannot verify SessionType support; config was not changed: %w", ctx.Err())
	}
	if err != nil {
		return fmt.Errorf("cannot save SessionType=%s (requires OpenSSH 8.7+); config was not changed: %w: %s", value, err, strings.TrimSpace(string(output)))
	}
	return nil
}
