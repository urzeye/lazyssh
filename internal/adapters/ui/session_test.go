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

package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/urzeye/lazyssh/internal/core/domain"
	"github.com/urzeye/lazyssh/internal/core/ports"
)

type sessionService struct {
	ports.ServerService
	err    error
	called bool
}

func (s *sessionService) SSH(string) error                   { s.called = true; return s.err }
func (s *sessionService) SSHWithArgs(string, []string) error { s.called = true; return s.err }

func TestExitAfterSessionPreservesSuccessAndFailure(t *testing.T) {
	for _, err := range []error{nil, errors.New("permission denied")} {
		svc := &sessionService{err: err}
		screen := tcell.NewSimulationScreen("")
		app := tview.NewApplication().SetScreen(screen)
		view := &tui{app: app, serverService: svc, options: Options{ExitAfterSession: true}}
		view.runSSHSession("prod", nil)
		if !svc.called || !errors.Is(view.sessionErr, err) {
			t.Fatalf("session result lost: %v", view.sessionErr)
		}
	}
}

type tagFailureService struct {
	ports.ServerService
	tags []string
}

func (s *tagFailureService) UpdateTags(_ domain.Server, tags []string) error {
	s.tags = tags
	return errors.New("metadata is unwritable")
}

func TestTagFailureShowsErrorAndPreservesInput(t *testing.T) {
	svc := &tagFailureService{}
	screen := tcell.NewSimulationScreen("")
	app := tview.NewApplication().SetScreen(screen)
	defer app.Stop()
	screen.SetSize(90, 24)
	view := &tui{app: app, serverService: svc}
	view.showEditTagsForm(domain.Server{Alias: "fresh"})
	field, ok := app.GetFocus().(*tview.InputField)
	if !ok {
		t.Fatalf("unexpected form focus %T", app.GetFocus())
	}
	field.SetText("prod")
	setFocus := func(p tview.Primitive) { app.SetFocus(p) }
	field.InputHandler()(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), setFocus)
	app.GetFocus().InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), setFocus)
	app.ForceDraw()
	cells, _, _ := screen.GetContents()
	var rendered strings.Builder
	for _, cell := range cells {
		if len(cell.Runes) > 0 {
			rendered.WriteRune(cell.Runes[0])
		}
	}
	if len(svc.tags) != 1 || svc.tags[0] != "prod" || !strings.Contains(rendered.String(), "Save tags failed") || !strings.Contains(rendered.String(), "unwritable") {
		t.Fatalf("tag error not shown: %s", rendered.String())
	}
	app.GetFocus().InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), setFocus)
	button, ok := app.GetFocus().(*tview.Button)
	if !ok || button.GetLabel() != "Save" || field.GetText() != "prod" {
		t.Fatal("tag input lost after closing the error")
	}
}
