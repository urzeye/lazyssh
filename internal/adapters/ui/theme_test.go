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

import "testing"

func TestThemeModeToggle(t *testing.T) {
	if got := ThemeDark.Toggle(); got != ThemeLight {
		t.Fatalf("ThemeDark.Toggle() = %q, want %q", got, ThemeLight)
	}
	if got := ThemeLight.Toggle(); got != ThemeDark {
		t.Fatalf("ThemeLight.Toggle() = %q, want %q", got, ThemeDark)
	}
}

func TestSetCurrentThemeFallsBackToDark(t *testing.T) {
	SetCurrentTheme(ThemeMode("unknown"))
	if CurrentTheme.Mode != ThemeDark {
		t.Fatalf("CurrentTheme.Mode = %q, want %q", CurrentTheme.Mode, ThemeDark)
	}
}
