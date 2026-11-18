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
	for _, r := range "qaedpstfgrjkcShT012/" {
		event := tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
		if got := view.handleGlobalKeys(event); got != event {
			t.Fatalf("search rune %q consumed by global handler", r)
		}
	}
}
