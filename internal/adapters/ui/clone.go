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
	"slices"
	"strings"
	"time"

	"github.com/urzeye/lazyssh/internal/core/domain"
)

func uniqueAlias(base string, servers []domain.Server) string {
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r) {
			return r
		}
		return '_'
	}, base)
	if base == "" {
		base = "server"
	}
	used := make(map[string]bool)
	for _, server := range servers {
		used[server.Alias] = true
		for _, alias := range server.Aliases {
			used[alias] = true
		}
	}
	alias := base
	for n := 2; used[alias]; n++ {
		alias = fmt.Sprintf("%s_%d", base, n)
	}
	return alias
}

func cloneServer(source domain.Server, servers []domain.Server) domain.Server {
	clone := source
	clone.Alias = uniqueAlias(source.Alias+"_copy", servers)
	clone.Aliases = nil
	clone.SourceFile = ""
	clone.Readonly = false
	clone.LastSeen = time.Time{}
	clone.PinnedAt = time.Time{}
	clone.SSHCount = 0
	clone.IdentityFiles = slices.Clone(source.IdentityFiles)
	clone.Tags = slices.Clone(source.Tags)
	clone.LocalForward = slices.Clone(source.LocalForward)
	clone.RemoteForward = slices.Clone(source.RemoteForward)
	clone.DynamicForward = slices.Clone(source.DynamicForward)
	clone.SendEnv = slices.Clone(source.SendEnv)
	clone.SetEnv = slices.Clone(source.SetEnv)
	return clone
}
