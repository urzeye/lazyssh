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
	"slices"
	"strings"

	"github.com/kevinburke/ssh_config"
	"github.com/urzeye/lazyssh/internal/core/domain"
)

const (
	MaxBackups     = 10
	TempSuffix     = ".tmp"
	BackupSuffix   = "lazyssh.backup"
	SSHConfigPerms = 0o600
)

func insertHost(hosts []*ssh_config.Host, host *ssh_config.Host, alias string) []*ssh_config.Host {
	for i, existing := range hosts {
		if existing.Implicit || !hostMatchesAlias(existing, alias) {
			continue
		}
		// OpenSSH takes the first value, so specific hosts must precede defaults.
		host.Nodes = append(host.Nodes, &ssh_config.Empty{})
		return slices.Insert(hosts, i, host)
	}
	return append(hosts, host)
}

// filterServers filters servers based on the query string.
func (r *Repository) filterServers(servers []domain.Server, query string) []domain.Server {
	query = strings.ToLower(query)
	filtered := make([]domain.Server, 0)

	for _, server := range servers {
		if r.matchesQuery(server, query) {
			filtered = append(filtered, server)
		}
	}

	return filtered
}

// matchesQuery checks if any field of the server matches the query string.
func (r *Repository) matchesQuery(server domain.Server, query string) bool {
	fields := []string{
		strings.ToLower(server.Host),
		strings.ToLower(server.User),
	}
	for _, tag := range server.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	for _, alias := range server.Aliases {
		fields = append(fields, strings.ToLower(alias))
	}

	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

// findHostByAlias finds a host by its alias in the SSH config.
func (r *Repository) findHostByAlias(cfg *ssh_config.Config, alias string) *ssh_config.Host {
	for _, host := range cfg.Hosts {
		if r.hostContainsPattern(host, alias) {
			return host
		}
	}
	return nil
}

// hostContainsPattern checks if a host contains a specific pattern.
func (r *Repository) hostContainsPattern(host *ssh_config.Host, target string) bool {
	target = normalizeHostAlias(target)
	for _, pattern := range host.Patterns {
		if normalizeHostAlias(pattern.String()) == target {
			return true
		}
	}
	return false
}

// createHostFromServer creates a new ssh_config.Host from a domain.Server.
func (r *Repository) createHostFromServer(server domain.Server) *ssh_config.Host {
	host := &ssh_config.Host{
		Patterns: []*ssh_config.Pattern{
			{Str: server.Alias},
		},
		Nodes:              make([]ssh_config.Node, 0),
		EOLComment:         "Added by lazyssh",
		SpaceBeforeComment: strings.Repeat(" ", 4),
	}

	// Basic config - always present
	r.addKVNodeIfNotEmpty(host, "HostName", server.Host)
	r.addKVNodeIfNotEmpty(host, "User", server.User)
	if server.Port != 0 {
		r.addKVNodeIfNotEmpty(host, "Port", fmt.Sprintf("%d", server.Port))
	}
	for _, identityFile := range server.IdentityFiles {
		r.addKVNodeIfNotEmpty(host, "IdentityFile", normalizeIdentityFilePath(identityFile))
	}
	r.updateHostNodes(host, server)

	return host
}

// addKVNodeIfNotEmpty adds a key-value node to the host if the value is not empty.
func (r *Repository) addKVNodeIfNotEmpty(host *ssh_config.Host, key, value string) {
	if value == "" {
		return
	}

	kvNode := &ssh_config.KV{
		Key:          key,
		Value:        value,
		LeadingSpace: 4,
	}
	r.insertKVNodeAfterLastKV(host, kvNode)
}

// matchBoundaryIndex returns the index of the first Match directive in nodes,
// or len(nodes) if none exists. The ssh_config parser attaches Match blocks to
// the preceding Host's Nodes slice; host edits must not cross that boundary.
func matchBoundaryIndex(nodes []ssh_config.Node) int {
	for i, node := range nodes {
		if kv, ok := node.(*ssh_config.KV); ok && strings.EqualFold(kv.Key, "match") {
			return i
		}
	}
	return len(nodes)
}

// insertKVNodeAfterLastKV inserts a KV node immediately after the last existing KV node in the host.
// This preserves any trailing non-KV nodes (blank lines, comments) that may exist after the host's
// configuration block, preventing formatting shifts when adding new fields to a host entry.
//
// Example: If a host block has trailing blank lines separating it from the next host entry,
// this function ensures new fields are inserted before those blank lines, maintaining the
// visual separation between host blocks.
func (r *Repository) insertKVNodeAfterLastKV(host *ssh_config.Host, kvNode *ssh_config.KV) {
	boundary := matchBoundaryIndex(host.Nodes)

	// STEP 1: Find the last KV node before any Match boundary (search backwards)
	lastKVIndex := -1
	for i := boundary - 1; i >= 0; i-- {
		if _, ok := host.Nodes[i].(*ssh_config.KV); ok {
			lastKVIndex = i
			break
		}
	}

	// STEP 2: Handle case where no KV nodes exist before the Match boundary
	if lastKVIndex == -1 {
		if len(host.Nodes) == 0 {
			// Case A: Empty host - just append
			host.Nodes = append(host.Nodes, kvNode)
		} else {
			// Case B: Only comments/blanks or Match nodes exist - prepend before them
			host.Nodes = append([]ssh_config.Node{kvNode}, host.Nodes...)
		}
		return
	}

	// STEP 3: We found KV nodes - insert after the last one
	insertAt := lastKVIndex + 1

	if insertAt == len(host.Nodes) {
		// Case C: Last KV is at the end - just append
		host.Nodes = append(host.Nodes, kvNode)
		return
	}

	// Case D: Last KV has trailing nodes (blanks/comments/Match) - insert between them
	host.Nodes = append(host.Nodes[:insertAt], append([]ssh_config.Node{kvNode}, host.Nodes[insertAt:]...)...)
}

// removeNodesByKey removes all nodes with the specified key before any Match boundary.
func removeNodesByKey(nodes []ssh_config.Node, key string) []ssh_config.Node {
	boundary := matchBoundaryIndex(nodes)
	filtered := make([]ssh_config.Node, 0, len(nodes))
	for _, node := range nodes[:boundary] {
		if kv, ok := node.(*ssh_config.KV); ok {
			if strings.EqualFold(kv.Key, key) {
				continue // skip nodes with matching key
			}
		}
		filtered = append(filtered, node)
	}
	return append(filtered, nodes[boundary:]...)
}

type hostOptionUpdate struct {
	key     string
	aliases []string
	value   string
}

// updateHostNodes updates the nodes of an existing host with new server details.
func (r *Repository) updateHostNodes(host *ssh_config.Host, newServer domain.Server) {
	// Handle Port - include if explicitly set (even if it's 22)
	portValue := ""
	if newServer.Port != 0 {
		portValue = fmt.Sprintf("%d", newServer.Port)
	}

	// OpenSSH uses the first value for single-value directives, including proxy
	// directives. Keeping updates in an ordered slice preserves existing node
	// positions and makes new-host serialization deterministic.
	updates := []hostOptionUpdate{
		{key: "hostname", value: newServer.Host},
		{key: "user", value: newServer.User},
		{key: "port", value: portValue},
		{key: "proxyjump", value: newServer.ProxyJump},
		{key: "proxycommand", value: newServer.ProxyCommand},
		{key: "remotecommand", value: newServer.RemoteCommand},
		{key: "requesttty", value: newServer.RequestTTY},
		{key: "sessiontype", value: newServer.SessionType},
		{key: "connecttimeout", value: newServer.ConnectTimeout},
		{key: "connectionattempts", value: newServer.ConnectionAttempts},
		{key: "bindaddress", value: newServer.BindAddress},
		{key: "bindinterface", value: newServer.BindInterface},
		{key: "addressfamily", value: newServer.AddressFamily},
		{key: "exitonforwardfailure", value: newServer.ExitOnForwardFailure},
		{key: "ipqos", value: newServer.IPQoS},
		{key: "canonicalizehostname", value: newServer.CanonicalizeHostname},
		{key: "canonicaldomains", value: newServer.CanonicalDomains},
		{key: "canonicalizefallbacklocal", value: newServer.CanonicalizeFallbackLocal},
		{key: "canonicalizemaxdots", value: newServer.CanonicalizeMaxDots},
		{key: "canonicalizepermittedcnames", value: newServer.CanonicalizePermittedCNAMEs},
		{key: "clearallforwardings", value: newServer.ClearAllForwardings},
		{key: "gatewayports", value: newServer.GatewayPorts},
		{key: "pubkeyauthentication", value: newServer.PubkeyAuthentication},
		{key: "passwordauthentication", value: newServer.PasswordAuthentication},
		{key: "preferredauthentications", value: newServer.PreferredAuthentications},
		{key: "pubkeyacceptedalgorithms", aliases: []string{"pubkeyacceptedkeytypes"}, value: newServer.PubkeyAcceptedAlgorithms},
		{key: "hostbasedacceptedalgorithms", aliases: []string{"hostbasedkeytypes", "hostbasedacceptedkeytypes"}, value: newServer.HostbasedAcceptedAlgorithms},
		{key: "identitiesonly", value: newServer.IdentitiesOnly},
		{key: "addkeystoagent", value: newServer.AddKeysToAgent},
		{key: "identityagent", value: newServer.IdentityAgent},
		{key: "kbdinteractiveauthentication", aliases: []string{"challengeresponseauthentication"}, value: newServer.KbdInteractiveAuthentication},
		{key: "numberofpasswordprompts", value: newServer.NumberOfPasswordPrompts},
		{key: "forwardagent", value: newServer.ForwardAgent},
		{key: "forwardx11", value: newServer.ForwardX11},
		{key: "forwardx11trusted", value: newServer.ForwardX11Trusted},
		{key: "controlmaster", value: newServer.ControlMaster},
		{key: "controlpath", value: newServer.ControlPath},
		{key: "controlpersist", value: newServer.ControlPersist},
		{key: "serveraliveinterval", value: newServer.ServerAliveInterval},
		{key: "serveralivecountmax", value: newServer.ServerAliveCountMax},
		{key: "compression", value: newServer.Compression},
		{key: "tcpkeepalive", value: newServer.TCPKeepAlive},
		{key: "batchmode", value: newServer.BatchMode},
		{key: "stricthostkeychecking", value: newServer.StrictHostKeyChecking},
		{key: "checkhostip", value: newServer.CheckHostIP},
		{key: "fingerprinthash", value: newServer.FingerprintHash},
		{key: "userknownhostsfile", value: newServer.UserKnownHostsFile},
		{key: "hostkeyalgorithms", value: newServer.HostKeyAlgorithms},
		{key: "macs", value: newServer.MACs},
		{key: "ciphers", value: newServer.Ciphers},
		{key: "kexalgorithms", value: newServer.KexAlgorithms},
		{key: "verifyhostkeydns", value: newServer.VerifyHostKeyDNS},
		{key: "updatehostkeys", value: newServer.UpdateHostKeys},
		{key: "hashknownhosts", value: newServer.HashKnownHosts},
		{key: "visualhostkey", value: newServer.VisualHostKey},
		{key: "localcommand", value: newServer.LocalCommand},
		{key: "permitlocalcommand", value: newServer.PermitLocalCommand},
		{key: "escapechar", value: newServer.EscapeChar},
		{key: "loglevel", value: newServer.LogLevel},
	}

	// Update or remove nodes based on value
	for _, update := range updates {
		r.setHostOptionGroup(host, update.key, update.aliases, update.value)
	}

	// Replace multi-value entries entirely to reflect the new state
	host.Nodes = removeNodesByKey(host.Nodes, "IdentityFile")
	for _, identityFile := range newServer.IdentityFiles {
		r.addKVNodeIfNotEmpty(host, "IdentityFile", normalizeIdentityFilePath(identityFile))
	}

	host.Nodes = removeNodesByKey(host.Nodes, "LocalForward")
	for _, forward := range newServer.LocalForward {
		configFormat := r.convertCLIForwardToConfigFormat(forward)
		r.addKVNodeIfNotEmpty(host, "LocalForward", configFormat)
	}

	host.Nodes = removeNodesByKey(host.Nodes, "RemoteForward")
	for _, forward := range newServer.RemoteForward {
		configFormat := r.convertCLIForwardToConfigFormat(forward)
		r.addKVNodeIfNotEmpty(host, "RemoteForward", configFormat)
	}

	host.Nodes = removeNodesByKey(host.Nodes, "DynamicForward")
	for _, forward := range newServer.DynamicForward {
		r.addKVNodeIfNotEmpty(host, "DynamicForward", forward)
	}

	host.Nodes = removeNodesByKey(host.Nodes, "SendEnv")
	for _, env := range newServer.SendEnv {
		r.addKVNodeIfNotEmpty(host, "SendEnv", env)
	}

	host.Nodes = removeNodesByKey(host.Nodes, "SetEnv")
	for _, env := range newServer.SetEnv {
		r.addKVNodeIfNotEmpty(host, "SetEnv", env)
	}
}

func matchesOptionGroup(nodeKey, canonical string, aliases []string) bool {
	if strings.EqualFold(nodeKey, canonical) {
		return true
	}
	for _, alias := range aliases {
		if strings.EqualFold(nodeKey, alias) {
			return true
		}
	}
	return false
}

func (r *Repository) setHostOptionGroup(host *ssh_config.Host, key string, aliases []string, value string) {
	boundary := matchBoundaryIndex(host.Nodes)
	hostNodes := host.Nodes[:boundary]
	matchTail := host.Nodes[boundary:]

	if value == "" {
		filtered := make([]ssh_config.Node, 0, len(host.Nodes))
		for _, node := range hostNodes {
			if kvNode, ok := node.(*ssh_config.KV); ok && matchesOptionGroup(kvNode.Key, key, aliases) {
				continue
			}
			filtered = append(filtered, node)
		}
		filtered = append(filtered, matchTail...)
		host.Nodes = filtered
		return
	}

	// Update the first matching directive (canonical or deprecated alias) in place
	// and remove any redundant duplicate directives in the same group, without
	// crossing into any trailing Match section attached to host.Nodes.
	updated := false
	filtered := make([]ssh_config.Node, 0, len(host.Nodes))
	for _, node := range hostNodes {
		if kvNode, ok := node.(*ssh_config.KV); ok && matchesOptionGroup(kvNode.Key, key, aliases) {
			if !updated {
				kvNode.Value = value
				filtered = append(filtered, kvNode)
				updated = true
			}
			continue
		}
		filtered = append(filtered, node)
	}
	filtered = append(filtered, matchTail...)
	host.Nodes = filtered
	if updated {
		return
	}

	kvNode := &ssh_config.KV{
		Key:          r.getProperKeyCase(key),
		Value:        value,
		LeadingSpace: 4,
	}
	r.insertKVNodeAfterLastKV(host, kvNode)
}

// getProperKeyCase returns the proper case for known SSH config keys.
// Reference: https://www.ssh.com/academy/ssh/config
func (r *Repository) getProperKeyCase(key string) string {
	keyMap := map[string]string{
		"hostname":                        "HostName",
		"user":                            "User",
		"port":                            "Port",
		"identityfile":                    "IdentityFile",
		"proxycommand":                    "ProxyCommand",
		"proxyjump":                       "ProxyJump",
		"remotecommand":                   "RemoteCommand",
		"requesttty":                      "RequestTTY",
		"sessiontype":                     "SessionType",
		"connecttimeout":                  "ConnectTimeout",
		"connectionattempts":              "ConnectionAttempts",
		"bindaddress":                     "BindAddress",
		"bindinterface":                   "BindInterface",
		"addressfamily":                   "AddressFamily",
		"exitonforwardfailure":            "ExitOnForwardFailure",
		"ipqos":                           "IPQoS",
		"canonicalizehostname":            "CanonicalizeHostname",
		"canonicaldomains":                "CanonicalDomains",
		"canonicalizefallbacklocal":       "CanonicalizeFallbackLocal",
		"canonicalizemaxdots":             "CanonicalizeMaxDots",
		"canonicalizepermittedcnames":     "CanonicalizePermittedCNAMEs",
		"localforward":                    "LocalForward",
		"remoteforward":                   "RemoteForward",
		"dynamicforward":                  "DynamicForward",
		"clearallforwardings":             "ClearAllForwardings",
		"gatewayports":                    "GatewayPorts",
		"pubkeyauthentication":            "PubkeyAuthentication",
		"passwordauthentication":          "PasswordAuthentication",
		"preferredauthentications":        "PreferredAuthentications",
		"pubkeyacceptedalgorithms":        "PubkeyAcceptedAlgorithms",
		"pubkeyacceptedkeytypes":          "PubkeyAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"hostbasedacceptedalgorithms":     "HostbasedAcceptedAlgorithms",
		"hostbasedkeytypes":               "HostbasedAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"hostbasedacceptedkeytypes":       "HostbasedAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"identitiesonly":                  "IdentitiesOnly",
		"addkeystoagent":                  "AddKeysToAgent",
		"identityagent":                   "IdentityAgent",
		"kbdinteractiveauthentication":    "KbdInteractiveAuthentication",
		"challengeresponseauthentication": "KbdInteractiveAuthentication", // Deprecated alias
		"numberofpasswordprompts":         "NumberOfPasswordPrompts",
		"forwardagent":                    "ForwardAgent",
		"forwardx11":                      "ForwardX11",
		"forwardx11trusted":               "ForwardX11Trusted",
		"controlmaster":                   "ControlMaster",
		"controlpath":                     "ControlPath",
		"controlpersist":                  "ControlPersist",
		"serveraliveinterval":             "ServerAliveInterval",
		"serveralivecountmax":             "ServerAliveCountMax",
		"compression":                     "Compression",
		"tcpkeepalive":                    "TCPKeepAlive",
		"stricthostkeychecking":           "StrictHostKeyChecking",
		"checkhostip":                     "CheckHostIP",
		"fingerprinthash":                 "FingerprintHash",
		"verifyhostkeydns":                "VerifyHostKeyDNS",
		"updatehostkeys":                  "UpdateHostKeys",
		"hashknownhosts":                  "HashKnownHosts",
		"visualhostkey":                   "VisualHostKey",
		"userknownhostsfile":              "UserKnownHostsFile",
		"hostkeyalgorithms":               "HostKeyAlgorithms",
		"macs":                            "MACs",
		"ciphers":                         "Ciphers",
		"kexalgorithms":                   "KexAlgorithms",
		"localcommand":                    "LocalCommand",
		"permitlocalcommand":              "PermitLocalCommand",
		"escapechar":                      "EscapeChar",
		"sendenv":                         "SendEnv",
		"setenv":                          "SetEnv",
		"loglevel":                        "LogLevel",
		"batchmode":                       "BatchMode",
	}

	if properCase, exists := keyMap[strings.ToLower(key)]; exists {
		return properCase
	}
	return key
}

// convertCLIForwardToConfigFormat converts CLI format forwarding spec to SSH config format.
// CLI format: [bind_address:]port:host:hostport
// Config format: [bind_address:]port host:hostport
func (r *Repository) convertCLIForwardToConfigFormat(forward string) string {
	// Handle IPv6 addresses in brackets like [2001:db8::1]
	// These should be treated as a single unit

	// Find the last `:digits` that represents the final port
	lastPortStart := -1
	for i := len(forward) - 1; i >= 0; i-- {
		if forward[i] == ':' {
			// Check if everything after this colon is digits
			if i+1 < len(forward) {
				allDigits := true
				hasDigits := false
				for j := i + 1; j < len(forward); j++ {
					if forward[j] >= '0' && forward[j] <= '9' {
						hasDigits = true
					} else {
						allDigits = false
						break
					}
				}
				if allDigits && hasDigits {
					lastPortStart = i
					break
				}
			}
		}
	}

	if lastPortStart == -1 {
		// No port at the end, return as-is
		return forward
	}

	// Now find the split point between local and remote parts
	// We need to handle bracket-enclosed addresses specially
	inBrackets := 0
	for i := lastPortStart - 1; i >= 0; i-- {
		switch forward[i] {
		case ']':
			inBrackets++
		case '[':
			inBrackets--
		case ':':
			if inBrackets != 0 {
				continue
			}
			// This colon is not inside brackets
			// Check if this looks like it could be the split point
			// The split point would be after a port number (digits after a colon)

			// Look ahead to see what comes after this colon
			nextChar := byte(' ')
			if i+1 < len(forward) {
				nextChar = forward[i+1]
			}

			// If the next character could be start of a host (letter, digit, bracket)
			// then this is our split point
			if nextChar != ':' {
				localPart := forward[:i]
				remotePart := forward[i+1:]
				return localPart + " " + remotePart
			}
		}
	}

	// If no split point found, return as-is
	return forward
}

// removeHostByAlias removes a host by its alias from the list of hosts while
// preserving any trailing Match section attached to that host by the parser.
func (r *Repository) removeHostByAlias(hosts []*ssh_config.Host, alias string) ([]*ssh_config.Host, bool) {
	for i, host := range hosts {
		if r.hostContainsPattern(host, alias) {
			boundary := matchBoundaryIndex(host.Nodes)
			if boundary < len(host.Nodes) {
				if i > 0 {
					hosts[i-1].Nodes = append(hosts[i-1].Nodes, host.Nodes[boundary:]...)
				} else {
					hosts[0] = &ssh_config.Host{
						Implicit: true,
						Nodes:    host.Nodes[boundary:],
					}
					return hosts, true
				}
			}
			return append(hosts[:i], hosts[i+1:]...), true
		}
	}
	return hosts, false
}

func normalizeIdentityFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		return filepath.ToSlash(path)
	}

	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.ToSlash(path)
	}

	relativePath, err := filepath.Rel(home, path)
	if err != nil || relativePath == "." || relativePath == "" {
		return filepath.ToSlash(path)
	}

	prefix := ".." + string(filepath.Separator)
	if relativePath == ".." || strings.HasPrefix(relativePath, prefix) {
		return filepath.ToSlash(path)
	}

	return "~/" + filepath.ToSlash(relativePath)
}
