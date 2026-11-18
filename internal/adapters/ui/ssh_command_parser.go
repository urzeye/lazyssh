package ui

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/urzeye/lazyssh/internal/core/domain"
)

// ParseSSHCommand reads command text without evaluating shell expansions or commands.
func ParseSSHCommand(text string) (domain.Server, error) {
	args, err := splitSSHCommand(text)
	if err != nil {
		return domain.Server{}, err
	}
	if len(args) < 2 || !strings.EqualFold(strings.TrimSuffix(path.Base(strings.ReplaceAll(args[0], "\\", "/")), ".exe"), "ssh") {
		return domain.Server{}, fmt.Errorf("paste a command beginning with ssh")
	}
	server := domain.Server{}
	target := ""
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if target != "" || i+1 >= len(args) {
				return domain.Server{}, fmt.Errorf("missing or ambiguous SSH destination")
			}
			target = args[i+1]
			server.RemoteCommand = strings.Join(args[i+2:], " ")
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			if target == "" {
				target = arg
				continue
			}
			server.RemoteCommand = strings.Join(args[i:], " ")
			break
		}
		for j := 1; j < len(arg); j++ {
			flag := arg[j]
			if strings.ContainsRune("pilJLRDoboecmB", rune(flag)) {
				value := arg[j+1:]
				if value == "" {
					i++
					if i >= len(args) {
						return domain.Server{}, fmt.Errorf("-%c requires a value", flag)
					}
					value = args[i]
				}
				key := map[byte]string{'p': "Port", 'i': "IdentityFile", 'l': "User", 'J': "ProxyJump", 'L': "LocalForward", 'R': "RemoteForward", 'D': "DynamicForward", 'b': "BindAddress", 'B': "BindInterface", 'e': "EscapeChar", 'c': "Ciphers", 'm': "MACs"}[flag]
				if flag == 'o' {
					pos := strings.IndexAny(value, "= \t")
					if pos < 1 {
						return domain.Server{}, fmt.Errorf("-o requires Key=Value")
					}
					key, value = value[:pos], strings.TrimSpace(strings.TrimLeft(value[pos:], "= \t"))
				}
				if err := applyPastedSSHOption(&server, key, value); err != nil {
					return domain.Server{}, err
				}
				break
			}
			if err := applyPastedSSHFlag(&server, flag); err != nil {
				return domain.Server{}, err
			}
		}
	}
	if target == "" || strings.HasPrefix(target, "-") {
		return domain.Server{}, fmt.Errorf("missing or invalid SSH destination")
	}
	if err := applySSHDestination(&server, target); err != nil {
		return domain.Server{}, err
	}
	server.Alias = uniqueAlias(server.Host, nil)
	if server.Port == 0 {
		server.Port = 22
	}
	return server, nil
}

func applySSHDestination(server *domain.Server, target string) error {
	if strings.HasPrefix(target, "ssh://") {
		u, err := url.Parse(target)
		if err != nil || u.Hostname() == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid SSH URI")
		}
		if u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				return fmt.Errorf("SSH URI passwords cannot be imported")
			}
			if err := applyPastedSSHOption(server, "User", u.User.Username()); err != nil {
				return err
			}
		}
		if u.Port() != "" {
			if err := applyPastedSSHOption(server, "Port", u.Port()); err != nil {
				return err
			}
		}
		target = u.Hostname()
	} else if index := strings.LastIndex(target, "@"); index >= 0 {
		if index == 0 {
			return fmt.Errorf("SSH destination has an empty user")
		}
		if err := applyPastedSSHOption(server, "User", target[:index]); err != nil {
			return err
		}
		target = target[index+1:]
	}
	target = strings.TrimPrefix(strings.TrimSuffix(target, "]"), "[")
	if target == "" {
		return fmt.Errorf("SSH destination has an empty host")
	}
	if server.Host == "" {
		server.Host = target
	}
	return nil
}

func applyPastedSSHFlag(server *domain.Server, flag byte) error {
	options := map[byte][2]string{
		'A': {"ForwardAgent", "yes"}, 'a': {"ForwardAgent", "no"},
		'X': {"ForwardX11", "yes"}, 'x': {"ForwardX11", "no"}, 'Y': {"ForwardX11Trusted", "yes"},
		'C': {"Compression", "yes"}, '4': {"AddressFamily", "inet"}, '6': {"AddressFamily", "inet6"},
		'T': {"RequestTTY", "no"}, 'N': {"SessionType", "none"}, 's': {"SessionType", "subsystem"}, 'q': {"LogLevel", "QUIET"},
	}
	if flag == 't' {
		if server.RequestTTY == "yes" {
			server.RequestTTY = "force"
		} else if server.RequestTTY == "" {
			server.RequestTTY = "yes"
		}
		return nil
	}
	if flag == 'v' {
		switch server.LogLevel {
		case "DEBUG1":
			server.LogLevel = "DEBUG2"
		case "DEBUG2", "DEBUG3":
			server.LogLevel = "DEBUG3"
		default:
			server.LogLevel = "DEBUG1"
		}
		return nil
	}
	if option, ok := options[flag]; ok {
		return applyPastedSSHOption(server, option[0], option[1])
	}
	return fmt.Errorf("unsupported SSH option -%c; remove it or enter its settings in the form", flag)
}

func applyPastedSSHOption(server *domain.Server, key, value string) error {
	if value == "" {
		return fmt.Errorf("SSH option %s has an empty value", key)
	}
	key = strings.ToLower(key)
	if key == "port" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		if server.Port == 0 {
			server.Port = port
		}
		return nil
	}
	multi := map[string]*[]string{"identityfile": &server.IdentityFiles, "localforward": &server.LocalForward, "remoteforward": &server.RemoteForward, "dynamicforward": &server.DynamicForward, "sendenv": &server.SendEnv, "setenv": &server.SetEnv}
	if field, ok := multi[key]; ok {
		if key == "localforward" || key == "remoteforward" {
			value = strings.Join(strings.Fields(value), ":")
		}
		*field = append(*field, value)
		return nil
	}
	fields := map[string]*string{
		"hostname": &server.Host, "user": &server.User, "proxyjump": &server.ProxyJump, "proxycommand": &server.ProxyCommand,
		"remotecommand": &server.RemoteCommand, "requesttty": &server.RequestTTY, "sessiontype": &server.SessionType,
		"connecttimeout": &server.ConnectTimeout, "connectionattempts": &server.ConnectionAttempts,
		"bindaddress": &server.BindAddress, "bindinterface": &server.BindInterface, "addressfamily": &server.AddressFamily,
		"exitonforwardfailure": &server.ExitOnForwardFailure, "ipqos": &server.IPQoS,
		"canonicalizehostname": &server.CanonicalizeHostname, "canonicaldomains": &server.CanonicalDomains,
		"canonicalizefallbacklocal": &server.CanonicalizeFallbackLocal, "canonicalizemaxdots": &server.CanonicalizeMaxDots, "canonicalizepermittedcnames": &server.CanonicalizePermittedCNAMEs,
		"clearallforwardings": &server.ClearAllForwardings, "gatewayports": &server.GatewayPorts,
		"pubkeyauthentication": &server.PubkeyAuthentication, "pubkeyacceptedalgorithms": &server.PubkeyAcceptedAlgorithms, "pubkeyacceptedkeytypes": &server.PubkeyAcceptedAlgorithms,
		"hostbasedacceptedalgorithms": &server.HostbasedAcceptedAlgorithms, "identitiesonly": &server.IdentitiesOnly,
		"addkeystoagent": &server.AddKeysToAgent, "identityagent": &server.IdentityAgent,
		"passwordauthentication": &server.PasswordAuthentication, "kbdinteractiveauthentication": &server.KbdInteractiveAuthentication, "numberofpasswordprompts": &server.NumberOfPasswordPrompts, "preferredauthentications": &server.PreferredAuthentications,
		"forwardagent": &server.ForwardAgent, "forwardx11": &server.ForwardX11, "forwardx11trusted": &server.ForwardX11Trusted,
		"controlmaster": &server.ControlMaster, "controlpath": &server.ControlPath, "controlpersist": &server.ControlPersist,
		"serveraliveinterval": &server.ServerAliveInterval, "serveralivecountmax": &server.ServerAliveCountMax, "compression": &server.Compression, "tcpkeepalive": &server.TCPKeepAlive, "batchmode": &server.BatchMode,
		"stricthostkeychecking": &server.StrictHostKeyChecking, "checkhostip": &server.CheckHostIP, "fingerprinthash": &server.FingerprintHash, "userknownhostsfile": &server.UserKnownHostsFile,
		"hostkeyalgorithms": &server.HostKeyAlgorithms, "macs": &server.MACs, "ciphers": &server.Ciphers, "kexalgorithms": &server.KexAlgorithms,
		"verifyhostkeydns": &server.VerifyHostKeyDNS, "updatehostkeys": &server.UpdateHostKeys, "hashknownhosts": &server.HashKnownHosts, "visualhostkey": &server.VisualHostKey,
		"localcommand": &server.LocalCommand, "permitlocalcommand": &server.PermitLocalCommand, "escapechar": &server.EscapeChar, "loglevel": &server.LogLevel,
	}
	field, ok := fields[key]
	if !ok {
		return fmt.Errorf("unsupported SSH configuration option %q", key)
	}
	if *field == "" {
		*field = value
	}
	return nil
}

func splitSSHCommand(text string) ([]string, error) {
	text = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\\\r\n", ""), "\\\n", ""))
	var args []string
	var token strings.Builder
	var quote rune
	started := false
	runes := []rune(text)
	flush := func() {
		if started {
			args = append(args, token.String())
			token.Reset()
			started = false
		}
	}
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0 || r == '\n' || r == '\r' {
			return nil, fmt.Errorf("paste a single SSH command; use backslash continuation for multiple lines")
		}
		if r == '\\' && quote != '\'' {
			if i+1 == len(runes) {
				return nil, fmt.Errorf("unfinished escape in SSH command")
			}
			next := runes[i+1]
			if quote == '"' && !strings.ContainsRune("$`\"\\", next) {
				token.WriteRune(r)
				started = true
				continue
			}
			i++
			token.WriteRune(next)
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				token.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		if strings.ContainsRune(";|&<>`", r) {
			return nil, fmt.Errorf("shell operators are not supported in pasted SSH commands")
		}
		token.WriteRune(r)
		started = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in SSH command")
	}
	flush()
	return args, nil
}
