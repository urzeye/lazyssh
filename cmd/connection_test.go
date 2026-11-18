package main

import (
	"errors"
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
	"github.com/urzeye/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type directService struct {
	ports.ServerService
	selected string
	err      error
}

func (s *directService) ListServers(string) ([]domain.Server, error) {
	return []domain.Server{{Alias: "prod", Aliases: []string{"prod", "alternate"}}}, nil
}
func (s *directService) SSH(alias string) error { s.selected = alias; return s.err }

func TestDirectConnectionRequiresExactAlias(t *testing.T) {
	s := &directService{}
	if err := connectAlias(s, "pro"); err == nil || s.selected != "" {
		t.Fatal("connected using fuzzy alias")
	}
	if err := connectAlias(s, "alternate"); err != nil || s.selected != "alternate" {
		t.Fatalf("alternate alias: %s %v", s.selected, err)
	}
	s.err = errors.New("SSH failed")
	if !errors.Is(connectAlias(s, "prod"), s.err) {
		t.Fatal("SSH error lost")
	}
}

func TestCLIRejectsConflictingSelection(t *testing.T) {
	for _, args := range [][]string{{"first", "second"}, {"prod", "--filter", "other"}, {"--connect", "prod", "--filter", "prod"}, {"--connect="}} {
		cmd := newRootCommand(zap.NewNop().Sugar(), t.TempDir())
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}
