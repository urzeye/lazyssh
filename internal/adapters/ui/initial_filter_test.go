package ui

import (
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"github.com/urzeye/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type initialFilterService struct {
	ports.ServerService
	queries []string
}

func (s *initialFilterService) ListServers(query string) ([]domain.Server, error) {
	s.queries = append(s.queries, query)
	return []domain.Server{{Alias: "prod"}}, nil
}
func (s *initialFilterService) ListActiveSessions() ([]domain.ActiveSession, error) { return nil, nil }
func (s *initialFilterService) IsForwarding(string) bool                            { return false }

func TestInitialFilterNeverLoadsUnfilteredHosts(t *testing.T) {
	s := &initialFilterService{}
	view := NewTUI(zap.NewNop().Sugar(), s, "test", "test", Options{InitialFilter: "prod"}).(*tui)
	view.initializeTheme().buildComponents().buildLayout().bindEvents()
	view.loadInitialData()
	if !view.searchVisible || view.searchBar.GetText() != "prod" || len(s.queries) == 0 {
		t.Fatal("initial filter not displayed or applied")
	}
	for _, q := range s.queries {
		if q != "prod" {
			t.Fatalf("loaded unfiltered hosts: %q", s.queries)
		}
	}
}
