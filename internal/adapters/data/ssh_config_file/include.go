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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

// loadAllServers loads servers from the main SSH config and all recursively
// included config files. The root config always wins when aliases collide,
// while mutability is determined by the configured write target.
func (r *Repository) loadAllServers() ([]domain.Server, error) {
	mainPath := normalizeConfigPath(r.readConfigPath)
	writablePath := normalizeConfigPath(r.writeConfigPath)

	if _, err := r.fileSystem.Stat(mainPath); err != nil {
		if r.fileSystem.IsNotExist(err) {
			return []domain.Server{}, nil
		}
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	visited := map[string]struct{}{mainPath: {}}
	paths := []string{mainPath}

	rootCfg, err := r.decodeConfigAt(mainPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decode main config: %w", err)
	}

	includePaths, err := r.resolveIncludes(mainPath, visited)
	if err != nil {
		r.logger.Warnf("failed to resolve includes for %s: %v", mainPath, err)
	}
	paths = append(paths, includePaths...)

	servers := make([]domain.Server, 0, len(paths)*4)
	seenAliases := make(map[string]struct{}, len(paths)*4)

	for index, path := range paths {
		cfg, err := r.decodeConfigAt(path)
		if err != nil {
			if index == 0 {
				return nil, fmt.Errorf("failed to decode main config: %w", err)
			}
			r.logger.Warnf("failed to decode included config %s: %v", path, err)
			continue
		}

		current := r.toDomainServersFromConfig(cfg, rootCfg, path, path != writablePath)
		for _, server := range current {
			if aliasesAlreadySeen(seenAliases, server.Aliases) {
				continue
			}
			markAliasesSeen(seenAliases, server.Aliases)
			servers = append(servers, server)
		}
	}

	return servers, nil
}

func (r *Repository) findLoadedServerByAlias(alias string, excludedAliases map[string]struct{}) (domain.Server, bool, error) {
	servers, err := r.loadAllServers()
	if err != nil {
		return domain.Server{}, false, err
	}

	for _, server := range servers {
		if excludedAliases != nil && serverOverlapsAliases(server, excludedAliases) {
			continue
		}
		if serverHasAlias(server, alias) {
			return server, true, nil
		}
	}

	return domain.Server{}, false, nil
}

func (r *Repository) duplicateAliasError(alias string, server domain.Server) error {
	if server.SourceFile != "" {
		return fmt.Errorf("server with alias '%s' already exists in %s", alias, server.SourceFile)
	}
	return fmt.Errorf("server with alias '%s' already exists", alias)
}

// resolveIncludes scans a config file for Include directives and resolves them
// depth-first so nested config files are discovered in a stable order.
func (r *Repository) resolveIncludes(filePath string, visited map[string]struct{}) ([]string, error) {
	file, err := r.fileSystem.Open(filePath)
	if err != nil {
		if r.fileSystem.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open include source %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	baseDir := filepath.Dir(filePath)
	paths := make([]string, 0, 8)
	added := make(map[string]struct{}, 8)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		line = stripInlineComment(line)
		if line == "" {
			continue
		}

		fields := splitFieldsRespectQuotes(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Include") {
			continue
		}

		for _, pattern := range fields[1:] {
			pattern = unquote(strings.TrimSpace(pattern))
			if pattern == "" {
				continue
			}

			pattern = expandTilde(pattern)
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(baseDir, pattern)
			}

			matches, globErr := filepath.Glob(pattern)
			if globErr != nil || len(matches) == 0 {
				continue
			}

			for _, match := range matches {
				match = normalizeConfigPath(match)

				info, statErr := r.fileSystem.Stat(match)
				if statErr != nil || info.IsDir() {
					continue
				}
				if _, seen := visited[match]; seen {
					continue
				}

				visited[match] = struct{}{}
				if _, exists := added[match]; !exists {
					paths = append(paths, match)
					added[match] = struct{}{}
				}

				children, childErr := r.resolveIncludes(match, visited)
				if childErr != nil {
					r.logger.Warnf("failed to resolve nested includes for %s: %v", match, childErr)
					continue
				}
				for _, child := range children {
					if _, exists := added[child]; exists {
						continue
					}
					paths = append(paths, child)
					added[child] = struct{}{}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan include source %s: %w", filePath, err)
	}

	return paths, nil
}

func (r *Repository) decodeConfigAt(path string) (*ssh_config.Config, error) {
	file, err := r.fileSystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	return ssh_config.Decode(file)
}

func normalizeConfigPath(path string) string {
	path = expandTilde(path)
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		return path
	}

	if absPath, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absPath)
	}

	return path
}

func expandTilde(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}

func stripInlineComment(line string) string {
	inSingleQuotes := false
	inDoubleQuotes := false

	for index := 0; index < len(line); index++ {
		switch line[index] {
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			}
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			}
		case '#':
			if !inSingleQuotes && !inDoubleQuotes {
				return strings.TrimSpace(line[:index])
			}
		}
	}

	return strings.TrimSpace(line)
}

func splitFieldsRespectQuotes(line string) []string {
	fields := make([]string, 0, 4)
	var builder strings.Builder
	inSingleQuotes := false
	inDoubleQuotes := false

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		fields = append(fields, builder.String())
		builder.Reset()
	}

	for index := 0; index < len(line); index++ {
		switch line[index] {
		case ' ', '\t':
			if inSingleQuotes || inDoubleQuotes {
				builder.WriteByte(line[index])
			} else {
				flush()
			}
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			}
			builder.WriteByte(line[index])
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			}
			builder.WriteByte(line[index])
		default:
			builder.WriteByte(line[index])
		}
	}

	flush()
	return fields
}

func unquote(value string) string {
	if len(value) < 2 {
		return value
	}

	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}

	return value
}

func aliasesAlreadySeen(seen map[string]struct{}, aliases []string) bool {
	for _, alias := range aliases {
		if _, exists := seen[alias]; exists {
			return true
		}
	}
	return false
}

func markAliasesSeen(seen map[string]struct{}, aliases []string) {
	for _, alias := range aliases {
		seen[alias] = struct{}{}
	}
}

func serverHasAlias(server domain.Server, alias string) bool {
	for _, existingAlias := range server.Aliases {
		if existingAlias == alias {
			return true
		}
	}
	return false
}

func serverOverlapsAliases(server domain.Server, aliases map[string]struct{}) bool {
	for _, alias := range server.Aliases {
		if _, exists := aliases[alias]; exists {
			return true
		}
	}
	return false
}
