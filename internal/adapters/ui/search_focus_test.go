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
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestSearchMouseFocusPreservesPrintableCommands(t *testing.T) {
	app := tview.NewApplication()
	search := NewSearchBar()
	search.SetRect(0, 0, 60, 3)
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)
	app.SetFocus(search)
	search.Draw(screen)
	search.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(15, 1, tcell.Button1, tcell.ModNone), func(p tview.Primitive) { app.SetFocus(p) })
	if !search.HasFocus() {
		t.Fatal("mouse did not focus search")
	}
	view := &tui{app: app, searchBar: search}
	if view.captureMainUIState().focus != focusSearch {
		t.Fatal("mouse search focus lost during state capture")
	}
	for _, r := range "qaedpstfgrjkcShTCvPF012/" {
		event := tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
		if got := view.handleGlobalKeys(event); got != event {
			t.Fatalf("search rune %q consumed by global handler", r)
		}
	}
}
