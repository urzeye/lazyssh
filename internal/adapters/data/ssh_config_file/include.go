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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
	"github.com/urzeye/lazyssh/internal/core/domain"
)

const maxIncludeDepth = 16

type effectiveBlock struct {
	guards []*ssh_config.Host
	host   *ssh_config.Host
	nodes  []*ssh_config.KV
}

type discoveredHost struct {
	guards     []*ssh_config.Host
	host       *ssh_config.Host
	sourceFile string
}

// loadAllServers loads servers from the main SSH config and all recursively
// included config files in OpenSSH evaluation order. Mutability is determined
// by whether a host is defined in the configured write target.
func (r *Repository) loadAllServers() ([]domain.Server, error) {
	mainPath := normalizeConfigPath(r.readConfigPath)
	writablePath := normalizeConfigPath(r.writeConfigPath)

	if _, err := r.fileSystem.Stat(mainPath); err != nil {
		if r.fileSystem.IsNotExist(err) {
			return []domain.Server{}, nil
		}
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	rootCfg, err := r.decodeConfigAt(mainPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decode main config: %w", err)
	}

	ancestors := map[string]struct{}{canonicalConfigPath(mainPath): {}}
	effective := make([]effectiveBlock, 0, len(rootCfg.Hosts)*2)
	discovered := make([]discoveredHost, 0, len(rootCfg.Hosts)*2)
	r.flattenConfig(mainPath, rootCfg, nil, ancestors, 0, &effective, &discovered)

	// Collect aliases defined in the writable target first so an earlier
	// read-only included Host block cannot shadow a writable definition of the
	// same alias (effective values are still evaluated in full OpenSSH order).
	writableAliases := make(map[string]struct{}, len(discovered))
	for _, entry := range discovered {
		if !sameConfigFile(entry.sourceFile, writablePath) {
			continue
		}
		markAliasesSeen(writableAliases, discoverHostAliases(entry))
	}

	servers := make([]domain.Server, 0, len(discovered))
	seenAliases := make(map[string]struct{}, len(discovered))
	for _, entry := range discovered {
		readonly := !sameConfigFile(entry.sourceFile, writablePath)
		aliases := filterNewAliases(discoverHostAliases(entry), seenAliases, writableAliases, readonly)
		server, ok := r.toDomainServerFromAliases(entry, aliases, effective, readonly)
		if !ok {
			continue
		}
		markAliasesSeen(seenAliases, server.Aliases)
		servers = append(servers, server)
	}

	return servers, nil
}

func (r *Repository) flattenConfig(
	filePath string,
	cfg *ssh_config.Config,
	guards []*ssh_config.Host,
	ancestors map[string]struct{},
	depth int,
	effective *[]effectiveBlock,
	discovered *[]discoveredHost,
) {
	baseDir := filepath.Dir(filePath)
	for _, host := range cfg.Hosts {
		if !host.Implicit {
			*discovered = append(*discovered, discoveredHost{
				guards:     guards,
				host:       host,
				sourceFile: filePath,
			})
		}

		activeHost := host
		currentNodes := make([]*ssh_config.KV, 0, len(host.Nodes))
		flushNodes := func() {
			if len(currentNodes) == 0 {
				return
			}
			*effective = append(*effective, effectiveBlock{
				guards: guards,
				host:   activeHost,
				nodes:  currentNodes,
			})
			currentNodes = nil
		}

		skipMatchBlock := false
		for _, node := range host.Nodes {
			switch typed := node.(type) {
			case *ssh_config.KV:
				// Match directives start a new section that ssh_config attaches to
				// the preceding Host block. Treat unconditional "Match all" as
				// resuming global scope (still bounded by outer Include guards),
				// and skip conditional Match blocks we cannot evaluate statically.
				if strings.EqualFold(typed.Key, "match") {
					flushNodes()
					if isUnconditionalMatchAll(typed.Value) {
						activeHost = &ssh_config.Host{Implicit: true}
						skipMatchBlock = false
					} else {
						skipMatchBlock = true
					}
					continue
				}
				if skipMatchBlock {
					continue
				}
				currentNodes = append(currentNodes, typed)
			case *ssh_config.Include:
				if skipMatchBlock || depth >= maxIncludeDepth {
					continue
				}
				flushNodes()

				childGuards := guards
				if !activeHost.Implicit {
					childGuards = append(append([]*ssh_config.Host(nil), guards...), activeHost)
				}

				for _, match := range r.resolveIncludeLine(baseDir, typed.String(), ancestors) {
					childCfg, err := r.decodeConfigAt(match)
					if err != nil {
						r.logger.Warnf("failed to decode included config %s: %v", match, err)
						continue
					}
					canonical := canonicalConfigPath(match)
					ancestors[canonical] = struct{}{}
					r.flattenConfig(match, childCfg, childGuards, ancestors, depth+1, effective, discovered)
					delete(ancestors, canonical)
				}
			}
		}
		flushNodes()
	}
}

func isUnconditionalMatchAll(value string) bool {
	fields := strings.Fields(value)
	return len(fields) == 1 && strings.EqualFold(fields[0], "all")
}

func (r *Repository) resolveIncludeLine(baseDir, line string, ancestors map[string]struct{}) []string {
	patterns := parseIncludePatterns(line)
	if len(patterns) == 0 {
		return nil
	}

	paths := make([]string, 0, len(patterns))
	seenInLine := make(map[string]struct{}, len(patterns))
	for _, pattern := range patterns {
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

			canonical := canonicalConfigPath(match)
			if _, active := ancestors[canonical]; active {
				continue
			}
			if _, dup := seenInLine[canonical]; dup {
				continue
			}
			seenInLine[canonical] = struct{}{}
			paths = append(paths, match)
		}
	}

	return paths
}

func parseIncludePatterns(line string) []string {
	line = stripInlineComment(strings.TrimSpace(line))
	if line == "" {
		return nil
	}
	fields := splitFieldsRespectQuotes(line)
	if len(fields) == 0 {
		return nil
	}
	first := fields[0]
	if strings.EqualFold(first, "Include") {
		if len(fields) < 2 {
			return nil
		}
		if fields[1] == "=" {
			return fields[2:]
		}
		return fields[1:]
	}
	if len(first) > len("Include=") && strings.EqualFold(first[:len("Include=")], "Include=") {
		rest := []string{first[len("Include="):]}
		return append(rest, fields[1:]...)
	}
	return nil
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

func (r *Repository) decodeConfigAt(path string) (*ssh_config.Config, error) {
	file, err := r.fileSystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	return decodeConfig(file)
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

func canonicalConfigPath(path string) string {
	normalized := normalizeConfigPath(path)
	if resolved, err := filepath.EvalSymlinks(normalized); err == nil && resolved != "" {
		return filepath.Clean(resolved)
	}
	return normalized
}

func sameConfigFile(pathA, pathB string) bool {
	if normalizeConfigPath(pathA) == normalizeConfigPath(pathB) {
		return true
	}
	return canonicalConfigPath(pathA) == canonicalConfigPath(pathB)
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
		return filepath.Join(home, filepath.FromSlash(path[2:]))
	}
	if strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, filepath.FromSlash(strings.ReplaceAll(path[2:], `\`, "/")))
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

func filterNewAliases(aliases []string, seen, writable map[string]struct{}, readonly bool) []string {
	filtered := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if _, exists := seen[alias]; exists {
			continue
		}
		if readonly {
			if _, ownedByWritable := writable[alias]; ownedByWritable {
				continue
			}
		}
		filtered = append(filtered, alias)
	}
	return filtered
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
