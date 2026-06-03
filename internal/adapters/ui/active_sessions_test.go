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
	"testing"

	"github.com/urzeye/lazyssh/internal/core/domain"
)

func TestActiveSessionsSetSessionsShowsCountAndLimit(t *testing.T) {
	view := NewActiveSessions()
	view.SetSessions([]domain.ActiveSession{
		{PID: 1, Alias: "alpha"},
		{PID: 2, Alias: "beta"},
		{PID: 3, Alias: "gamma"},
		{PID: 4, Alias: "delta"},
		{PID: 5, Alias: "epsilon"},
	})

	if got := view.GetTitle(); got != " Active Sessions: 5 " {
		t.Fatalf("title = %q, want count title", got)
	}
	text := view.GetText(false)
	if !strings.Contains(text, "alpha") || !strings.Contains(text, "+1 more") {
		t.Fatalf("text = %q, want first session and overflow indicator", text)
	}
	if strings.Contains(text, "epsilon") {
		t.Fatalf("text = %q, should not include hidden overflow session", text)
	}
}

func TestActiveSessionsSetSessionsShowsEmptyState(t *testing.T) {
	view := NewActiveSessions()
	view.SetSessions(nil)

	if text := view.GetText(false); !strings.Contains(text, "No active SSH sessions") {
		t.Fatalf("text = %q, want empty state", text)
	}
}
