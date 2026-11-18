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
	"strings"

	"github.com/rivo/tview"
)

func DefaultStatusText() string {
	items := []string{
		keyText("0") + " Search",
		keyText("1") + " Servers",
		keyText("2") + " Details",
		keyText("↑↓") + " Navigate",
		keyText("Enter") + " SSH",
		keyText("T") + " Theme",
		keyText("f") + " Forward",
		keyText("x") + " Stop Forward",
		keyText("c") + " Copy SSH",
		keyText("h") + " Copy Host",
		keyText("a") + " Add",
		keyText("C") + " Clone",
		keyText("v") + " Paste SSH",
		keyText("P") + " SCP",
		keyText("F") + " SFTP",
		keyText("e") + " Edit",
		keyText("g") + " Ping",
		keyText("d") + " Delete",
		keyText("p") + " Pin/Unpin",
		keyText("/") + " Search",
		keyText("q") + " Quit",
	}
	return strings.Join(items, "  • ")
}

func NewStatusBar() *tview.TextView {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBackgroundColor(CurrentTheme.StatusBackground)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText(DefaultStatusText())
	return status
}
