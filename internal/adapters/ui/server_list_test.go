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

	"github.com/urzeye/lazyssh/internal/core/domain"
)

func TestServerListSelectServerByAlias(t *testing.T) {
	list := NewServerList()
	list.UpdateServers([]domain.Server{
		{Alias: "alpha"},
		{Alias: "beta"},
		{Alias: "gamma"},
	})

	if !list.SelectServerByAlias("beta") {
		t.Fatalf("SelectServerByAlias() = false, want true")
	}

	selected, ok := list.GetSelectedServer()
	if !ok {
		t.Fatalf("GetSelectedServer() ok = false, want true")
	}
	if selected.Alias != "beta" {
		t.Fatalf("selected alias = %q, want beta", selected.Alias)
	}
}

func TestServerListSelectIndexRejectsOutOfRange(t *testing.T) {
	list := NewServerList()
	list.UpdateServers([]domain.Server{{Alias: "alpha"}})

	if list.SelectIndex(3) {
		t.Fatalf("SelectIndex() = true, want false")
	}
}

func TestServerListPreservesSelectionAcrossRefresh(t *testing.T) {
	list := NewServerList()
	list.UpdateServers([]domain.Server{{Alias: "alpha"}, {Alias: "beta"}})
	list.SelectServerByAlias("beta")
	list.UpdateServers([]domain.Server{{Alias: "beta"}, {Alias: "alpha"}})
	selected, ok := list.GetSelectedServer()
	if !ok || selected.Alias != "beta" {
		t.Fatalf("selection lost: %+v", selected)
	}
	list.UpdateServers([]domain.Server{{Alias: "alpha"}})
	selected, ok = list.GetSelectedServer()
	if !ok || selected.Alias != "alpha" {
		t.Fatalf("removed selection not clamped: %+v", selected)
	}
	list.UpdateServers(nil)
	if _, ok := list.GetSelectedServer(); ok {
		t.Fatal("empty list retains selection")
	}
}
