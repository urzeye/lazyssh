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
	"testing"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

type mockServerRepository struct {
	servers []domain.Server
}

func (m mockServerRepository) ListServers(query string) ([]domain.Server, error) {
	cloned := make([]domain.Server, len(m.servers))
	copy(cloned, m.servers)
	return cloned, nil
}

func (m mockServerRepository) UpdateServer(server domain.Server, newServer domain.Server) error {
	return nil
}

func (m mockServerRepository) AddServer(server domain.Server) error {
	return nil
}

func (m mockServerRepository) DeleteServer(server domain.Server) error {
	return nil
}

func (m mockServerRepository) SetPinned(alias string, pinned bool) error {
	return nil
}

func (m mockServerRepository) RecordSSH(alias string) error {
	return nil
}

func (m mockServerRepository) GetConfigFile() string {
	return "/tmp/test-config"
}

func TestListServersRanksFuzzyMatches(t *testing.T) {
	service := NewServerService(zap.NewNop().Sugar(), mockServerRepository{
		servers: []domain.Server{
			{Alias: "prod-api", Host: "10.0.0.1"},
			{Alias: "pa", Host: "10.0.0.2"},
			{Alias: "staging", Host: "10.0.0.3"},
			{Alias: "db-admin", Host: "10.0.0.4", Tags: []string{"postgres", "primary"}},
		},
	})

	servers, err := service.ListServers("pa")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) < 2 {
		t.Fatalf("ListServers() returned %d matches, want at least 2", len(servers))
	}
	if servers[0].Alias != "pa" {
		t.Fatalf("top ranked alias = %q, want %q", servers[0].Alias, "pa")
	}
	if servers[1].Alias != "prod-api" {
		t.Fatalf("second ranked alias = %q, want %q", servers[1].Alias, "prod-api")
	}

	tagMatches, err := service.ListServers("pst")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(tagMatches) == 0 || tagMatches[0].Alias != "db-admin" {
		t.Fatalf("tag-based fuzzy search should match db-admin first, got %#v", tagMatches)
	}
}

func TestListServersKeepsPinnedFirstWithoutQuery(t *testing.T) {
	now := time.Now()
	service := NewServerService(zap.NewNop().Sugar(), mockServerRepository{
		servers: []domain.Server{
			{Alias: "beta"},
			{Alias: "alpha", PinnedAt: now},
			{Alias: "gamma", PinnedAt: now.Add(-time.Hour)},
		},
	})

	servers, err := service.ListServers("")
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) != 3 {
		t.Fatalf("ListServers() returned %d servers, want 3", len(servers))
	}
	if servers[0].Alias != "alpha" || servers[1].Alias != "gamma" || servers[2].Alias != "beta" {
		t.Fatalf("unexpected order: %#v", servers)
	}
}

func TestFuzzyScoreRejectsNonSubsequence(t *testing.T) {
	if score := fuzzyScore("xyz", "prod-api"); score != 0 {
		t.Fatalf("fuzzyScore() = %d, want 0", score)
	}
}
