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
	"fmt"
	"strings"

	"github.com/rivo/tview"
	"github.com/urzeye/lazyssh/internal/core/domain"
)

const maxActiveSessionsShown = 4

type ActiveSessions struct {
	*tview.TextView
}

func NewActiveSessions() *ActiveSessions {
	sessions := &ActiveSessions{
		TextView: tview.NewTextView(),
	}
	sessions.build()
	return sessions
}

func (as *ActiveSessions) build() {
	as.TextView.SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(false).
		SetBorder(true).
		SetTitle(" Active Sessions ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(CurrentTheme.Border).
		SetTitleColor(CurrentTheme.Title)
	as.SetSessions(nil)
}

func (as *ActiveSessions) SetSessions(sessions []domain.ActiveSession) {
	as.SetTitle(fmt.Sprintf(" Active Sessions: %d ", len(sessions)))
	if len(sessions) == 0 {
		as.SetText(colorText(CurrentTheme.DimText, "No active SSH sessions"))
		return
	}

	lines := make([]string, 0, min(len(sessions), maxActiveSessionsShown)+1)
	for index, session := range sessions {
		if index >= maxActiveSessionsShown {
			lines = append(lines, colorText(CurrentTheme.DimText, fmt.Sprintf("+%d more", len(sessions)-index)))
			break
		}
		lines = append(lines, formatActiveSessionLine(session))
	}
	as.SetText(strings.Join(lines, "\n"))
}

func formatActiveSessionLine(session domain.ActiveSession) string {
	alias := session.Alias
	if alias == "" {
		alias = session.Destination
	}

	parts := []string{
		colorText(CurrentTheme.MetaText, fmt.Sprintf("%d", session.PID)),
		colorText(CurrentTheme.AliasText, tview.Escape(alias)),
	}
	if session.Destination != "" && session.Destination != alias {
		parts = append(parts, colorText(CurrentTheme.DimText, tview.Escape(session.Destination)))
	}
	if len(session.Forwards) > 0 {
		parts = append(parts, colorText(CurrentTheme.HostText, "fwd "+tview.Escape(strings.Join(session.Forwards, ", "))))
	}
	return strings.Join(parts, "  ")
}
