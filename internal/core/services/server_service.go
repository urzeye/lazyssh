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
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type serverService struct {
	serverRepository ports.ServerRepository
	logger           *zap.SugaredLogger

	fwMu     sync.Mutex
	forwards map[string][]*os.Process
}

// NewServerService creates a new instance of serverService.
func NewServerService(logger *zap.SugaredLogger, sr ports.ServerRepository) ports.ServerService {
	return &serverService{
		logger:           logger,
		serverRepository: sr,
	}
}

// ListServers returns a list of servers sorted with pinned on top.
func (s *serverService) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.serverRepository.ListServers("")
	if err != nil {
		s.logger.Errorw("failed to list servers", "error", err)
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		sortServersByDefault(servers)
		return servers, nil
	}

	return rankServersForQuery(servers, query), nil
}

func sortServersByDefault(servers []domain.Server) {
	sort.SliceStable(servers, func(i, j int) bool {
		pi := !servers[i].PinnedAt.IsZero()
		pj := !servers[j].PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj {
			return servers[i].PinnedAt.After(servers[j].PinnedAt)
		}
		return strings.ToLower(servers[i].Alias) < strings.ToLower(servers[j].Alias)
	})
}

func rankServersForQuery(servers []domain.Server, query string) []domain.Server {
	type scoredServer struct {
		server domain.Server
		score  int
	}

	scored := make([]scoredServer, 0, len(servers))
	for _, server := range servers {
		score := computeServerScore(server, query)
		if score > 0 {
			scored = append(scored, scoredServer{
				server: server,
				score:  score,
			})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}

		pi := !scored[i].server.PinnedAt.IsZero()
		pj := !scored[j].server.PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj && !scored[i].server.PinnedAt.Equal(scored[j].server.PinnedAt) {
			return scored[i].server.PinnedAt.After(scored[j].server.PinnedAt)
		}

		return strings.ToLower(scored[i].server.Alias) < strings.ToLower(scored[j].server.Alias)
	})

	ranked := make([]domain.Server, 0, len(scored))
	for _, entry := range scored {
		ranked = append(ranked, entry.server)
	}

	return ranked
}

func computeServerScore(server domain.Server, query string) int {
	bestScore := 0
	fields := []string{
		server.Alias,
		server.Host,
		server.User,
	}
	if len(server.Aliases) > 0 {
		fields = append(fields, strings.Join(server.Aliases, " "))
	}
	if len(server.Tags) > 0 {
		fields = append(fields, strings.Join(server.Tags, " "))
	}

	for _, field := range fields {
		if field == "" {
			continue
		}
		if score := fuzzyScore(query, field); score > bestScore {
			bestScore = score
		}
	}

	return bestScore
}

// fuzzyScore computes a subsequence score for query against value.
// It returns 0 when the query is not a case-insensitive subsequence.
func fuzzyScore(query, value string) int {
	if query == "" || value == "" {
		return 0
	}

	queryRunes := []rune(query)
	queryLower := []rune(strings.ToLower(query))
	valueRunes := []rune(value)
	valueLower := []rune(strings.ToLower(value))

	positions := make([]int, 0, len(queryLower))
	searchStart := 0
	for _, queryRune := range queryLower {
		matchIndex := -1
		for index := searchStart; index < len(valueLower); index++ {
			if valueLower[index] == queryRune {
				matchIndex = index
				positions = append(positions, index)
				searchStart = index + 1
				break
			}
		}
		if matchIndex == -1 {
			return 0
		}
	}

	score := len(positions)
	startIndex := positions[0]
	if startIndex < 20 {
		score += 20 - startIndex
	}

	gapPenalty := 0
	for index := 1; index < len(positions); index++ {
		if positions[index] == positions[index-1]+1 {
			score += 5
			continue
		}
		gapPenalty += positions[index] - positions[index-1] - 1
	}
	if gapPenalty > 15 {
		gapPenalty = 15
	}
	score -= gapPenalty

	for index, position := range positions {
		var previousRune rune
		if position > 0 {
			previousRune = valueRunes[position-1]
		}
		currentRune := valueRunes[position]

		if isWordBoundary(previousRune, currentRune, position) {
			if position == 0 {
				score += 8
			} else {
				score += 6
			}
		}

		if index < len(queryRunes) && queryRunes[index] == currentRune {
			score++
		}
	}

	return score
}

func isWordBoundary(previousRune, currentRune rune, index int) bool {
	if index == 0 {
		return true
	}
	if previousRune == '-' || previousRune == '_' || previousRune == '.' || previousRune == '/' || unicode.IsSpace(previousRune) {
		return true
	}
	return unicode.IsLower(previousRune) && unicode.IsUpper(currentRune)
}

// validateServer performs core validation of server fields.
func validateServer(srv domain.Server) error {
	if strings.TrimSpace(srv.Alias) == "" {
		return fmt.Errorf("alias is required")
	}
	if ok, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, srv.Alias); !ok {
		return fmt.Errorf("alias may contain letters, digits, dot, dash, underscore")
	}
	if strings.TrimSpace(srv.Host) == "" {
		return fmt.Errorf("Host/IP is required")
	}
	if ip := net.ParseIP(srv.Host); ip == nil {
		if strings.Contains(srv.Host, " ") {
			return fmt.Errorf("host must not contain spaces")
		}
		if ok, _ := regexp.MatchString(`^[A-Za-z0-9.-]+$`, srv.Host); !ok {
			return fmt.Errorf("host contains invalid characters")
		}
		if strings.HasPrefix(srv.Host, ".") || strings.HasSuffix(srv.Host, ".") {
			return fmt.Errorf("host must not start or end with a dot")
		}
		for _, lbl := range strings.Split(srv.Host, ".") {
			if lbl == "" {
				return fmt.Errorf("host must not contain empty labels")
			}
			if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
				return fmt.Errorf("hostname labels must not start or end with a hyphen")
			}
		}
	}
	if srv.Port != 0 && (srv.Port < 1 || srv.Port > 65535) {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}
	return nil
}

// UpdateServer updates an existing server with new details.
func (s *serverService) UpdateServer(server domain.Server, newServer domain.Server) error {
	if err := validateServer(newServer); err != nil {
		s.logger.Warnw("validation failed on update", "error", err, "server", newServer)
		return err
	}
	err := s.serverRepository.UpdateServer(server, newServer)
	if err != nil {
		s.logger.Errorw("failed to update server", "error", err, "server", server)
	}
	return err
}

// AddServer adds a new server to the repository.
func (s *serverService) AddServer(server domain.Server) error {
	if err := validateServer(server); err != nil {
		s.logger.Warnw("validation failed on add", "error", err, "server", server)
		return err
	}
	err := s.serverRepository.AddServer(server)
	if err != nil {
		s.logger.Errorw("failed to add server", "error", err, "server", server)
	}
	return err
}

// DeleteServer removes a server from the repository.
func (s *serverService) DeleteServer(server domain.Server) error {
	err := s.serverRepository.DeleteServer(server)
	if err != nil {
		s.logger.Errorw("failed to delete server", "error", err, "server", server)
	}
	return err
}

// SetPinned sets or clears a pin timestamp for the server alias.
func (s *serverService) SetPinned(alias string, pinned bool) error {
	err := s.serverRepository.SetPinned(alias, pinned)
	if err != nil {
		s.logger.Errorw("failed to set pin state", "error", err, "alias", alias, "pinned", pinned)
	}
	return err
}

// SSH starts an interactive SSH session to the given alias using the system's ssh client.
func (s *serverService) SSH(alias string) error {
	s.logger.Infow("ssh start", "alias", alias)
	// #nosec G204 -- lazyssh intentionally delegates to the user's local ssh client
	// using the selected alias and configured SSH config path.
	cmd := exec.Command("ssh", "-F", s.serverRepository.GetConfigFile(), alias)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh command failed", "alias", alias, "error", err)
		return err
	}

	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}

	s.logger.Infow("ssh end", "alias", alias)
	return nil
}

// SSHWithArgs runs system ssh with provided extra args (e.g., -L/-R/-D) for the given alias.
func (s *serverService) SSHWithArgs(alias string, extraArgs []string) error {
	s.logger.Infow("ssh start (with args)", "alias", alias, "args", extraArgs)
	args := append([]string{}, extraArgs...)
	args = append(args, "-F", s.serverRepository.GetConfigFile(), alias)
	// #nosec G204
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh (with args) failed", "alias", alias, "error", err)
		return err
	}
	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}
	s.logger.Infow("ssh end (with args)", "alias", alias)
	return nil
}

// StartForward starts ssh port forwarding in the background and tracks the process.
func (s *serverService) StartForward(alias string, extraArgs []string) (int, error) {
	s.fwMu.Lock()
	if s.forwards == nil {
		s.forwards = make(map[string][]*os.Process)
	}
	s.fwMu.Unlock()

	extraArgs = append(extraArgs, "-F", s.serverRepository.GetConfigFile(), "-N", alias)

	// #nosec G204
	cmd := exec.Command("ssh", extraArgs...)

	// Detach from TTY: discard stdio
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open devnull: %w", err)
	}
	defer func() {
		if devNull != nil {
			_ = devNull.Close()
		}
	}()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	// Set SysProcAttr in an OS-specific way (see sysprocattr_* files)
	sysProcAttr := &syscall.SysProcAttr{}
	setDetach(sysProcAttr)
	cmd.SysProcAttr = sysProcAttr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start ssh: %w", err)
	}

	proc := cmd.Process
	if proc == nil {
		return 0, fmt.Errorf("process is nil after start")
	}
	pid := proc.Pid

	// Track process
	s.fwMu.Lock()
	s.forwards[alias] = append(s.forwards[alias], proc)
	s.fwMu.Unlock()

	// Cleanup on exit
	go func(a string, c *exec.Cmd, dn *os.File) {
		_ = c.Wait()
		_ = dn.Close()

		s.fwMu.Lock()
		defer s.fwMu.Unlock()

		procs := s.forwards[a]
		if len(procs) == 0 {
			return
		}

		filtered := make([]*os.Process, 0, len(procs))
		for _, p := range procs {
			if p != nil && p.Pid != pid {
				filtered = append(filtered, p)
			}
		}

		if len(filtered) == 0 {
			delete(s.forwards, a)
		} else {
			s.forwards[a] = filtered
		}
	}(alias, cmd, devNull)

	devNull = nil // Prevent defer from closing it

	return pid, nil
}

// StopForwarding kills all active forward processes for the alias.
func (s *serverService) StopForwarding(alias string) error {
	s.fwMu.Lock()
	procs := s.forwards[alias]
	delete(s.forwards, alias)
	s.fwMu.Unlock()

	if len(procs) == 0 {
		return nil
	}

	var errs []error
	for _, p := range procs {
		if p != nil {
			if err := p.Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, try SIGKILL
				if killErr := p.Kill(); killErr != nil {
					errs = append(errs, fmt.Errorf("failed to kill pid %d: %w", p.Pid, killErr))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping forwards: %v", errs)
	}
	return nil
}

// IsForwarding reports whether there is at least one active forward for alias.
func (s *serverService) IsForwarding(alias string) bool {
	s.fwMu.Lock()
	defer s.fwMu.Unlock()
	return len(s.forwards[alias]) > 0
}

// Ping checks if the server is reachable on its SSH port.
func (s *serverService) Ping(server domain.Server) (bool, time.Duration, error) {
	start := time.Now()

	host, port, ok := resolveSSHDestination(server.Alias)
	if !ok {

		host = strings.TrimSpace(server.Host)
		if host == "" {
			host = server.Alias
		}
		if server.Port > 0 {
			port = server.Port
		} else {
			port = 22
		}
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return false, time.Since(start), err
	}
	_ = conn.Close()
	return true, time.Since(start), nil
}

// resolveSSHDestination uses `ssh -G <alias>` to extract HostName and Port from the user's SSH config.
// Returns host, port, ok where ok=false if resolution failed.
func resolveSSHDestination(alias string) (string, int, bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", 0, false
	}
	cmd := exec.Command("ssh", "-G", alias)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, false
	}
	host := ""
	port := 0
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "hostname ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				host = parts[1]
			}
		}
		if strings.HasPrefix(line, "port ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if p, err := strconv.Atoi(parts[1]); err == nil {
					port = p
				}
			}
		}
	}
	if host == "" {
		host = alias
	}
	if port == 0 {
		port = 22
	}
	return host, port, true
}
