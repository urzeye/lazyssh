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

func TestUsernameAllowsDigitLeadingAccounts(t *testing.T) {
	validator := GetFieldValidators()["User"]
	for _, name := range []string{"123user", "123", "_service", "user@example.com", "domain:user"} {
		if !validator.Pattern.MatchString(name) {
			t.Errorf("username %q rejected", name)
		}
	}
	for _, name := range []string{"user name", "-option", "user\nname", "$(cmd)", "user;cmd"} {
		if validator.Pattern.MatchString(name) {
			t.Errorf("invalid username %q accepted", name)
		}
	}
}
