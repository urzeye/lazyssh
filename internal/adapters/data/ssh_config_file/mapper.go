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
	"strconv"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/urzeye/lazyssh/internal/core/domain"
)

type effectiveKeyGroup struct {
	canonical string
	aliases   []string
}

var effectiveSingleValueGroups = []effectiveKeyGroup{
	{canonical: "HostName"},
	{canonical: "User"},
	{canonical: "Port"},
	{canonical: "ProxyCommand"},
	{canonical: "ProxyJump"},
	{canonical: "RemoteCommand"},
	{canonical: "RequestTTY"},
	{canonical: "SessionType"},
	{canonical: "ConnectTimeout"},
	{canonical: "ConnectionAttempts"},
	{canonical: "BindAddress"},
	{canonical: "BindInterface"},
	{canonical: "AddressFamily"},
	{canonical: "ExitOnForwardFailure"},
	{canonical: "IPQoS"},
	{canonical: "CanonicalizeHostname"},
	{canonical: "CanonicalDomains"},
	{canonical: "CanonicalizeFallbackLocal"},
	{canonical: "CanonicalizeMaxDots"},
	{canonical: "CanonicalizePermittedCNAMEs"},
	{canonical: "ClearAllForwardings"},
	{canonical: "GatewayPorts"},
	{canonical: "ForwardAgent"},
	{canonical: "ForwardX11"},
	{canonical: "ForwardX11Trusted"},
	{canonical: "PubkeyAuthentication"},
	{canonical: "PubkeyAcceptedAlgorithms", aliases: []string{"PubkeyAcceptedKeyTypes"}},
	{canonical: "HostbasedAcceptedAlgorithms", aliases: []string{"HostbasedKeyTypes", "HostbasedAcceptedKeyTypes"}},
	{canonical: "PasswordAuthentication"},
	{canonical: "PreferredAuthentications"},
	{canonical: "IdentitiesOnly"},
	{canonical: "AddKeysToAgent"},
	{canonical: "IdentityAgent"},
	{canonical: "KbdInteractiveAuthentication", aliases: []string{"ChallengeResponseAuthentication"}},
	{canonical: "NumberOfPasswordPrompts"},
	{canonical: "ControlMaster"},
	{canonical: "ControlPath"},
	{canonical: "ControlPersist"},
	{canonical: "ServerAliveInterval"},
	{canonical: "ServerAliveCountMax"},
	{canonical: "Compression"},
	{canonical: "TCPKeepAlive"},
	{canonical: "BatchMode"},
	{canonical: "StrictHostKeyChecking"},
	{canonical: "CheckHostIP"},
	{canonical: "FingerprintHash"},
	{canonical: "UserKnownHostsFile"},
	{canonical: "HostKeyAlgorithms"},
	{canonical: "MACs"},
	{canonical: "Ciphers"},
	{canonical: "KexAlgorithms"},
	{canonical: "VerifyHostKeyDNS"},
	{canonical: "UpdateHostKeys"},
	{canonical: "HashKnownHosts"},
	{canonical: "VisualHostKey"},
	{canonical: "LocalCommand"},
	{canonical: "PermitLocalCommand"},
	{canonical: "EscapeChar"},
	{canonical: "LogLevel"},
}

var effectiveMultiValueKeys = []string{
	"IdentityFile",
	"LocalForward",
	"RemoteForward",
	"DynamicForward",
	"SendEnv",
	"SetEnv",
}

func discoverHostAliases(entry discoveredHost) []string {
	aliases := make([]string, 0, len(entry.host.Patterns))
	seenInHost := make(map[string]struct{}, len(entry.host.Patterns))
	for _, pattern := range entry.host.Patterns {
		alias := normalizeHostAlias(pattern.String())
		// Skip if alias contains wildcards or negation (not a concrete Host)
		if alias == "" || strings.ContainsAny(alias, "!*?[]") {
			continue
		}
		// Only discover aliases that satisfy all enclosing conditional Include guards.
		if !guardsMatchAlias(entry.guards, alias) {
			continue
		}
		if _, dup := seenInHost[alias]; dup {
			continue
		}
		seenInHost[alias] = struct{}{}
		aliases = append(aliases, alias)
	}
	return aliases
}

func (r *Repository) toDomainServerFromAliases(entry discoveredHost, aliases []string, effective []effectiveBlock, readonly bool) (domain.Server, bool) {
	if len(aliases) == 0 {
		return domain.Server{}, false
	}

	server := domain.Server{
		Alias:         aliases[0],
		Aliases:       aliases,
		Port:          22,
		IdentityFiles: []string{},
		SourceFile:    entry.sourceFile,
		Readonly:      readonly,
	}

	if len(effective) > 0 {
		r.applyEffectiveBlocks(&server, effective)
	} else {
		r.applyHostNodes(&server, entry.host)
	}

	return server, true
}

func (r *Repository) applyHostNodes(server *domain.Server, host *ssh_config.Host) {
	for _, node := range host.Nodes {
		kvNode, ok := node.(*ssh_config.KV)
		if !ok {
			continue
		}
		if strings.EqualFold(kvNode.Key, "match") {
			break
		}

		r.mapKVToServer(server, kvNode)
	}
}

func (r *Repository) applyEffectiveBlocks(server *domain.Server, blocks []effectiveBlock) {
	matchingNodes := make([]*ssh_config.KV, 0, 16)
	for _, block := range blocks {
		if !effectiveBlockMatches(block, server.Alias) {
			continue
		}
		matchingNodes = append(matchingNodes, block.nodes...)
	}

	for _, group := range effectiveSingleValueGroups {
		if value, ok := firstMatchingNodeValue(matchingNodes, group); ok {
			r.mapKVToServer(server, &ssh_config.KV{Key: group.canonical, Value: value})
		}
	}

	for _, key := range effectiveMultiValueKeys {
		for _, node := range matchingNodes {
			if strings.EqualFold(node.Key, key) && node.Value != "" {
				r.mapKVToServer(server, node)
			}
		}
	}
}

func guardsMatchAlias(guards []*ssh_config.Host, alias string) bool {
	for _, guard := range guards {
		if !hostMatchesAlias(guard, alias) {
			return false
		}
	}
	return true
}

func effectiveBlockMatches(block effectiveBlock, alias string) bool {
	return guardsMatchAlias(block.guards, alias) && hostMatchesAlias(block.host, alias)
}

func firstMatchingNodeValue(nodes []*ssh_config.KV, group effectiveKeyGroup) (string, bool) {
	for _, node := range nodes {
		if node.Value == "" {
			continue
		}
		if strings.EqualFold(node.Key, group.canonical) {
			return node.Value, true
		}
		for _, aliasKey := range group.aliases {
			if strings.EqualFold(node.Key, aliasKey) {
				return node.Value, true
			}
		}
	}
	return "", false
}

func hostMatchesAlias(host *ssh_config.Host, alias string) bool {
	if host == nil {
		return false
	}
	if host.Implicit {
		return true
	}

	matched := false
	for _, pattern := range host.Patterns {
		raw := strings.TrimSpace(pattern.String())
		negated := false
		if strings.HasPrefix(raw, "!") {
			negated = true
			raw = raw[1:]
		}
		normalized := normalizeHostAlias(raw)
		if !negated && strings.HasPrefix(normalized, "!") {
			negated = true
			normalized = normalized[1:]
		}
		if normalized == "" {
			continue
		}
		if matchSSHPattern(normalized, alias) {
			if negated {
				return false
			}
			matched = true
		}
	}
	return matched
}

// matchSSHPattern implements OpenSSH '*' and '?' wildcard matching.
func matchSSHPattern(pattern, target string) bool {
	p, t := 0, 0
	starIdx, matchIdx := -1, 0
	for t < len(target) {
		switch {
		case p < len(pattern) && (pattern[p] == '?' || pattern[p] == target[t]):
			p++
			t++
		case p < len(pattern) && pattern[p] == '*':
			starIdx = p
			matchIdx = t
			p++
		case starIdx != -1:
			p = starIdx + 1
			matchIdx++
			t = matchIdx
		default:
			return false
		}
	}
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}
	return p == len(pattern)
}

func normalizeHostAlias(alias string) string {
	return strings.TrimSpace(unquote(strings.TrimSpace(alias)))
}

// mapKVToServer maps an ssh_config.KV node to the corresponding fields in domain.Server.
func (r *Repository) mapKVToServer(server *domain.Server, kvNode *ssh_config.KV) {
	key := strings.ToLower(kvNode.Key)
	value := kvNode.Value

	// Try mapping in order of categories
	if r.mapBasicConfig(server, key, value) {
		return
	}
	if r.mapConnectionConfig(server, key, value) {
		return
	}
	if r.mapForwardingConfig(server, key, value) {
		return
	}
	if r.mapAuthenticationConfig(server, key, value) {
		return
	}
	if r.mapSecurityConfig(server, key, value) {
		return
	}
	if r.mapEnvironmentConfig(server, key, value) {
		return
	}
	r.mapDebugConfig(server, key, value)
}

// mapBasicConfig maps basic SSH configuration fields
func (r *Repository) mapBasicConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "hostname":
		server.Host = value
	case "user":
		server.User = value
	case "port":
		port, err := strconv.Atoi(value)
		if err == nil {
			server.Port = port
		}
	case "identityfile":
		server.IdentityFiles = append(server.IdentityFiles, value)
	default:
		return false
	}
	return true
}

// mapConnectionConfig maps connection and proxy configuration fields
func (r *Repository) mapConnectionConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "proxycommand":
		server.ProxyCommand = value
	case "proxyjump":
		server.ProxyJump = value
	case "remotecommand":
		server.RemoteCommand = value
	case "requesttty":
		server.RequestTTY = value
	case "sessiontype":
		server.SessionType = value
	case "connecttimeout":
		server.ConnectTimeout = value
	case "connectionattempts":
		server.ConnectionAttempts = value
	case "bindaddress":
		server.BindAddress = value
	case "bindinterface":
		server.BindInterface = value
	case "addressfamily":
		server.AddressFamily = value
	case "exitonforwardfailure":
		server.ExitOnForwardFailure = value
	case "ipqos":
		server.IPQoS = value
	case "canonicalizehostname":
		server.CanonicalizeHostname = value
	case "canonicaldomains":
		server.CanonicalDomains = value
	case "canonicalizefallbacklocal":
		server.CanonicalizeFallbackLocal = value
	case "canonicalizemaxdots":
		server.CanonicalizeMaxDots = value
	case "canonicalizepermittedcnames":
		server.CanonicalizePermittedCNAMEs = value
	case "serveraliveinterval":
		server.ServerAliveInterval = value
	case "serveralivecountmax":
		server.ServerAliveCountMax = value
	case "compression":
		server.Compression = value
	case "tcpkeepalive":
		server.TCPKeepAlive = value
	case "batchmode":
		server.BatchMode = value
	case "controlmaster":
		server.ControlMaster = value
	case "controlpath":
		server.ControlPath = value
	case "controlpersist":
		server.ControlPersist = value
	default:
		return false
	}
	return true
}

// mapForwardingConfig maps port forwarding and agent forwarding fields
func (r *Repository) mapForwardingConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "localforward":
		cliFormat := r.convertConfigForwardToCLIFormat(value)
		server.LocalForward = append(server.LocalForward, cliFormat)
	case "remoteforward":
		cliFormat := r.convertConfigForwardToCLIFormat(value)
		server.RemoteForward = append(server.RemoteForward, cliFormat)
	case "dynamicforward":
		server.DynamicForward = append(server.DynamicForward, value)
	case "clearallforwardings":
		server.ClearAllForwardings = value
	case "gatewayports":
		server.GatewayPorts = value
	case "forwardagent":
		server.ForwardAgent = value
	case "forwardx11":
		server.ForwardX11 = value
	case "forwardx11trusted":
		server.ForwardX11Trusted = value
	default:
		return false
	}
	return true
}

// mapAuthenticationConfig maps authentication-related fields
func (r *Repository) mapAuthenticationConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "pubkeyauthentication":
		server.PubkeyAuthentication = value
	case "pubkeyacceptedalgorithms", "pubkeyacceptedkeytypes":
		// PubkeyAcceptedKeyTypes is deprecated alias for PubkeyAcceptedAlgorithms (since OpenSSH 8.5)
		server.PubkeyAcceptedAlgorithms = value
	case "hostbasedacceptedalgorithms", "hostbasedkeytypes", "hostbasedacceptedkeytypes":
		// HostbasedKeyTypes and HostbasedAcceptedKeyTypes are deprecated aliases (since OpenSSH 8.5)
		server.HostbasedAcceptedAlgorithms = value
	case "passwordauthentication":
		server.PasswordAuthentication = value
	case "preferredauthentications":
		server.PreferredAuthentications = value
	case "identitiesonly":
		server.IdentitiesOnly = value
	case "addkeystoagent":
		server.AddKeysToAgent = value
	case "identityagent":
		server.IdentityAgent = value
	case "kbdinteractiveauthentication", "challengeresponseauthentication":
		// ChallengeResponseAuthentication is deprecated alias for KbdInteractiveAuthentication
		server.KbdInteractiveAuthentication = value
	case "numberofpasswordprompts":
		server.NumberOfPasswordPrompts = value
	default:
		return false
	}
	return true
}

// mapSecurityConfig maps security-related fields
func (r *Repository) mapSecurityConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "stricthostkeychecking":
		server.StrictHostKeyChecking = value
	case "checkhostip":
		server.CheckHostIP = value
	case "fingerprinthash":
		server.FingerprintHash = value
	case "userknownhostsfile":
		server.UserKnownHostsFile = value
	case "hostkeyalgorithms":
		server.HostKeyAlgorithms = value
	case "macs":
		server.MACs = value
	case "ciphers":
		server.Ciphers = value
	case "kexalgorithms":
		server.KexAlgorithms = value
	case "verifyhostkeydns":
		server.VerifyHostKeyDNS = value
	case "updatehostkeys":
		server.UpdateHostKeys = value
	case "hashknownhosts":
		server.HashKnownHosts = value
	case "visualhostkey":
		server.VisualHostKey = value
	default:
		return false
	}
	return true
}

// mapEnvironmentConfig maps environment and command execution fields
func (r *Repository) mapEnvironmentConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "localcommand":
		server.LocalCommand = value
	case "permitlocalcommand":
		server.PermitLocalCommand = value
	case "escapechar":
		server.EscapeChar = value
	case "sendenv":
		server.SendEnv = append(server.SendEnv, value)
	case "setenv":
		server.SetEnv = append(server.SetEnv, value)
	default:
		return false
	}
	return true
}

// mapDebugConfig maps debugging-related fields
func (r *Repository) mapDebugConfig(server *domain.Server, key, value string) bool {
	switch key {
	case "loglevel":
		server.LogLevel = value
	default:
		return false
	}
	return true
}

// mergeMetadata merges additional metadata into the servers.
func (r *Repository) mergeMetadata(servers []domain.Server, metadata map[string]ServerMetadata) []domain.Server {
	for i, server := range servers {
		servers[i].LastSeen = time.Time{}

		if meta, exists := metadata[server.Alias]; exists {
			servers[i].Tags = meta.Tags
			servers[i].SSHCount = meta.SSHCount

			if meta.LastSeen != "" {
				if lastSeen, err := time.Parse(time.RFC3339, meta.LastSeen); err == nil {
					servers[i].LastSeen = lastSeen
				}
			}

			if meta.PinnedAt != "" {
				if pinnedAt, err := time.Parse(time.RFC3339, meta.PinnedAt); err == nil {
					servers[i].PinnedAt = pinnedAt
				}
			}
		}
	}
	return servers
}

// convertConfigForwardToCLIFormat converts SSH config format forwarding spec to CLI format.
// Config format: [bind_address:]port host:hostport
// CLI format: [bind_address:]port:host:hostport
func (r *Repository) convertConfigForwardToCLIFormat(forward string) string {
	// Find the last space which separates the local part from the remote part
	lastSpace := strings.LastIndex(forward, " ")
	if lastSpace != -1 {
		localPart := forward[:lastSpace]
		remotePart := forward[lastSpace+1:]
		// Join them with a colon for CLI format
		return localPart + ":" + remotePart
	}
	// If no space found, return as-is (might already be in CLI format)
	return forward
}
