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

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/urzeye/lazyssh/internal/adapters/data/ssh_config_file"
	"github.com/urzeye/lazyssh/internal/adapters/ui"
	"github.com/urzeye/lazyssh/internal/core/ports"
	"github.com/urzeye/lazyssh/internal/core/services"
	"github.com/urzeye/lazyssh/internal/logger"
	"go.uber.org/zap"
)

var (
	version   = "develop"
	gitCommit = "unknown"
)

func newRootCommand(log *zap.SugaredLogger, home string) *cobra.Command {
	var sshConfigFile, managedSSHConfigFile, filter, connect string
	var managedMode, sshConfigReadOnly, exitAfterSession bool
	const (
		sshConfigHelp        = "path to ssh config file (default: ~/.ssh/config)"
		managedModeHelp      = "read from the root ssh config but write changes to a managed ssh config file"
		managedSSHConfigHelp = "path to the writable managed ssh config file " +
			"(default: ~/.ssh/config.local when managed mode is enabled)"
		sshConfigReadOnlyHelp = "read hosts from the ssh config without writing changes back " +
			"(can also be set with LAZYSSH_SSH_CONFIG_READONLY=1)"
	)

	rootCmd := &cobra.Command{
		Use:     ui.AppName + " [filter]",
		Short:   "Lazy SSH server picker TUI",
		Version: formatVersion(version, gitCommit),
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if cmd.Flags().Changed("filter") {
					return fmt.Errorf("use either [filter] or --filter")
				}
				filter = args[0]
			}
			if connect != "" && filter != "" {
				return fmt.Errorf("--connect cannot be combined with a filter")
			}
			if cmd.Flags().Changed("connect") && connect == "" {
				return fmt.Errorf("--connect requires a nonempty alias")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := sshConfigFile
			if configPath == "" {
				configPath = filepath.Join(home, ".ssh", "config")
			}
			configPath, err := absoluteConfigPath(configPath, home)
			if err != nil {
				return err
			}

			managedConfigPath := managedSSHConfigFile
			useManagedMode := managedMode || managedConfigPath != ""
			if useManagedMode && managedConfigPath == "" {
				managedConfigPath = filepath.Join(home, ".ssh", "config.local")
			}
			if managedConfigPath != "" {
				managedConfigPath, err = absoluteConfigPath(managedConfigPath, home)
				if err != nil {
					return err
				}
			}

			metaDataFile := filepath.Join(home, ".lazyssh", "metadata.json")
			serverRepo := ssh_config_file.NewRepositoryWithWritePath(log, configPath, managedConfigPath, metaDataFile)
			if sshConfigReadOnly {
				serverRepo = ssh_config_file.NewReadOnlyRepository(log, configPath, metaDataFile)
			}

			serverService := services.NewServerService(log, serverRepo)
			if connect != "" {
				return connectAlias(serverService, connect)
			}
			options := ui.Options{InitialFilter: filter, ExitAfterSession: exitAfterSession}
			tui := ui.NewTUI(log, serverService, version, gitCommit, options)
			return tui.Run()
		},
	}
	rootCmd.PersistentFlags().StringVar(&sshConfigFile, "sshconfig", "", sshConfigHelp)
	rootCmd.PersistentFlags().BoolVar(&managedMode, "managed-mode", false, managedModeHelp)
	rootCmd.PersistentFlags().StringVar(&managedSSHConfigFile, "managed-sshconfig", "", managedSSHConfigHelp)
	rootCmd.PersistentFlags().BoolVar(
		&sshConfigReadOnly,
		"ssh-config-readonly",
		envBool("LAZYSSH_SSH_CONFIG_READONLY"),
		sshConfigReadOnlyHelp,
	)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "initial host filter")
	rootCmd.Flags().StringVarP(&connect, "connect", "c", "", "connect to an exact alias without opening the TUI")
	rootCmd.Flags().BoolVar(&exitAfterSession, "exit-after-session", false, "exit after the selected SSH session finishes")
	return rootCmd
}

func main() {
	os.Exit(run())
}

func run() int {
	log, err := logger.New("LAZYSSH")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = log.Sync() }()
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := newRootCommand(log, home).Execute(); err != nil {
		var alreadyReported interface{ StderrAlreadyPrinted() bool }
		if !errors.As(err, &alreadyReported) || !alreadyReported.StderrAlreadyPrinted() {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func formatVersion(ver, commit string) string {
	commit = strings.TrimSpace(commit)
	if commit == "" || commit == "unknown" || commit == "(devel)" {
		return ver
	}
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("%s (%s)", ver, commit)
}

func absoluteConfigPath(configPath, home string) (string, error) {
	if configPath == "~" {
		configPath = home
	}
	if strings.HasPrefix(configPath, "~/") || strings.HasPrefix(configPath, "~\\") {
		configPath = filepath.Join(home, configPath[2:])
	}
	return filepath.Abs(configPath)
}

func connectAlias(service ports.ServerService, alias string) error {
	servers, err := service.ListServers("")
	if err != nil {
		return err
	}
	for _, server := range servers {
		if server.Alias == alias || slices.Contains(server.Aliases, alias) {
			return service.SSH(alias)
		}
	}
	return fmt.Errorf("no configured host has alias %q", alias)
}

func envBool(name string) bool {
	switch os.Getenv(name) {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true
	default:
		return false
	}
}
