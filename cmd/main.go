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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/urzeye/lazyssh/internal/adapters/data/ssh_config_file"
	"github.com/urzeye/lazyssh/internal/adapters/ui"
	"github.com/urzeye/lazyssh/internal/core/services"
	"github.com/urzeye/lazyssh/internal/logger"
)

var (
	version              = "develop"
	gitCommit            = "unknown"
	sshConfigFile        string
	managedMode          bool
	managedSSHConfigFile string
	sshConfigReadOnly    bool
)

func main() {
	const (
		sshConfigHelp        = "path to ssh config file (default: ~/.ssh/config)"
		managedModeHelp      = "read from the root ssh config but write changes to a managed ssh config file"
		managedSSHConfigHelp = "path to the writable managed ssh config file " +
			"(default: ~/.ssh/config.local when managed mode is enabled)"
		sshConfigReadOnlyHelp = "read hosts from the ssh config without writing changes back " +
			"(can also be set with LAZYSSH_SSH_CONFIG_READONLY=1)"
	)

	log, err := logger.New("LAZYSSH")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//nolint:errcheck // log.Sync may return an error which is safe to ignore here
	defer log.Sync()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorw("failed to get user home directory", "error", err)
		//nolint:gocritic // exitAfterDefer: ensure immediate exit on unrecoverable error
		os.Exit(1)
	}
	rootCmd := &cobra.Command{
		Use:   ui.AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := sshConfigFile
			if configPath == "" {
				configPath = filepath.Join(home, ".ssh", "config")
			}

			managedConfigPath := managedSSHConfigFile
			useManagedMode := managedMode || managedConfigPath != ""
			if useManagedMode && managedConfigPath == "" {
				managedConfigPath = filepath.Join(home, ".ssh", "config.local")
			}

			metaDataFile := filepath.Join(home, ".lazyssh", "metadata.json")
			var serverRepo = ssh_config_file.NewRepositoryWithWritePath(log, configPath, managedConfigPath, metaDataFile)
			if sshConfigReadOnly {
				serverRepo = ssh_config_file.NewReadOnlyRepository(log, configPath, metaDataFile)
			}

			serverService := services.NewServerService(log, serverRepo)
			tui := ui.NewTUI(log, serverService, version, gitCommit)
			return tui.Run()
		},
	}
	rootCmd.PersistentFlags().StringVar(&sshConfigFile, "sshconfig", "", sshConfigHelp)
	rootCmd.PersistentFlags().BoolVar(&managedMode, "managed-mode", false, managedModeHelp)
	rootCmd.PersistentFlags().StringVar(&managedSSHConfigFile, "managed-sshconfig", "", managedSSHConfigHelp)
	rootCmd.PersistentFlags().BoolVar(&sshConfigReadOnly, "ssh-config-readonly", envBool("LAZYSSH_SSH_CONFIG_READONLY"), sshConfigReadOnlyHelp)
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envBool(name string) bool {
	switch os.Getenv(name) {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true
	default:
		return false
	}
}
