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
