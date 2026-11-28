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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
)

const deferredIncludeKeyPrefix = "__lazyssh_include_"

// loadConfig reads and parses the SSH config file.
// If the file does not exist, it returns an empty config without error to support first-run behavior.
func (r *Repository) loadConfig() (*ssh_config.Config, error) {
	return r.loadConfigAt(r.writeConfigPath)
}

func (r *Repository) loadConfigAt(path string) (*ssh_config.Config, error) {
	file, err := r.fileSystem.Open(path)
	if err != nil {
		if r.fileSystem.IsNotExist(err) {
			return &ssh_config.Config{Hosts: []*ssh_config.Host{}}, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close config file: %v", cerr)
		}
	}()

	cfg, err := decodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return cfg, nil
}

func decodeConfig(reader io.Reader) (*ssh_config.Config, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Defer Include directive parsing so ssh_config.Decode does not eagerly glob
	// and read files on disk (which fails on matched subdirectories, recursion
	// cycles >5 levels, or broken conditional Match includes). flattenConfig
	// expands Include directives safely in OpenSSH evaluation order instead.
	sanitized, deferred := deferIncludeDirectives(raw)
	cfg, err := ssh_config.Decode(bytes.NewReader(sanitized))
	if err != nil {
		return nil, err
	}
	if deferred {
		restoreDeferredIncludeKeys(cfg)
	}

	// The parser retains negation in its matcher but drops it from Pattern.Str.
	// Restore the spelling for listing and serialization without changing matching.
	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			single := &ssh_config.Host{Patterns: []*ssh_config.Pattern{pattern}}
			if !strings.HasPrefix(pattern.Str, "!") && !single.Matches(pattern.Str) {
				pattern.Str = "!" + pattern.Str
			}
		}
	}
	return cfg, nil
}

func deferIncludeDirectives(raw []byte) ([]byte, bool) {
	const kwLen = len("include")

	var out bytes.Buffer
	deferred := false
	lineStart := 0
	for lineStart < len(raw) {
		lineEnd := lineStart
		for lineEnd < len(raw) && raw[lineEnd] != '\n' {
			lineEnd++
		}
		contentEnd := lineEnd
		if contentEnd > lineStart && raw[contentEnd-1] == '\r' {
			contentEnd--
		}

		keyStart := lineStart
		for keyStart < contentEnd && (raw[keyStart] == ' ' || raw[keyStart] == '\t') {
			keyStart++
		}

		isIncludeLine := false
		if contentEnd-keyStart >= kwLen && strings.EqualFold(string(raw[keyStart:keyStart+kwLen]), "include") {
			afterKey := keyStart + kwLen
			if afterKey == contentEnd || raw[afterKey] == ' ' || raw[afterKey] == '\t' || raw[afterKey] == '=' {
				isIncludeLine = true
			}
		}

		nextLine := lineEnd
		if nextLine < len(raw) && raw[nextLine] == '\n' {
			nextLine++
		}

		if isIncludeLine {
			if !deferred {
				out.Grow(len(raw) + 64)
				out.Write(raw[:lineStart])
				deferred = true
			}
			out.Write(raw[lineStart:keyStart])
			out.WriteString(deferredIncludeKeyPrefix)
			out.Write(raw[keyStart:nextLine])
		} else if deferred {
			out.Write(raw[lineStart:nextLine])
		}

		lineStart = nextLine
	}

	if !deferred {
		return raw, false
	}
	return out.Bytes(), true
}

func restoreDeferredIncludeKeys(cfg *ssh_config.Config) {
	for _, host := range cfg.Hosts {
		for _, node := range host.Nodes {
			kv, ok := node.(*ssh_config.KV)
			if !ok || !strings.HasPrefix(kv.Key, deferredIncludeKeyPrefix) {
				continue
			}
			restored := kv.Key[len(deferredIncludeKeyPrefix):]
			if strings.EqualFold(restored, "include") {
				kv.Key = restored
			}
		}
	}
}

// saveConfig writes the SSH config back to the file with atomic operations and backup management.
func (r *Repository) saveConfig(cfg *ssh_config.Config) error {
	writePath := resolvedWriteConfigPath(r.writeConfigPath)
	configDir := filepath.Dir(writePath)
	if err := r.fileSystem.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tempFile, err := r.createTempFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	renamed := false
	defer func() {
		if renamed {
			return
		}
		if removeErr := r.fileSystem.Remove(tempFile); removeErr != nil && !r.fileSystem.IsNotExist(removeErr) {
			r.logger.Warnf("failed to remove temporary file %s: %v", tempFile, removeErr)
		}
	}()

	if err := r.writeConfigToFile(tempFile, cfg); err != nil {
		return fmt.Errorf("failed to write config to temporary file: %w", err)
	}

	// Ensure a one-time original backup exists before any modifications managed by lazyssh.
	if err := r.createOriginalBackupIfNeeded(); err != nil {
		return fmt.Errorf("failed to create original backup: %w", err)
	}

	if err := r.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := r.fileSystem.Rename(tempFile, writePath); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}
	renamed = true

	r.logger.Infof("SSH config successfully updated: %s", r.writeConfigPath)
	return nil
}

// writeConfigToFile writes the SSH config content to the specified file
func (r *Repository) writeConfigToFile(filePath string, cfg *ssh_config.Config) error {
	file, err := r.fileSystem.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close file %s: %v", filePath, cerr)
		}
	}()

	configContent := cfg.String()
	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file to disk: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the specified directory
func (r *Repository) createTempFile(dir string) (string, error) {
	timestamp := time.Now().UnixNano()
	baseName := filepath.Base(r.writeConfigPath)
	tempFileName := fmt.Sprintf("%s.%d%s", baseName, timestamp, TempSuffix)
	tempFilePath := filepath.Join(dir, tempFileName)

	// Create the temp file with explicit 0600 permissions
	f, err := r.fileSystem.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return "", err
	}
	if cerr := f.Close(); cerr != nil {
		r.logger.Warnf("failed to close temporary file %s: %v", tempFilePath, cerr)
	}

	return tempFilePath, nil
}

func resolvedWriteConfigPath(path string) string {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return path
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil || resolvedPath == "" {
		return path
	}

	return resolvedPath
}
