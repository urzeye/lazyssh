package ui

import (
	"strings"
	"testing"
)

func TestSCPCommandPreservesAliasAndConfig(t *testing.T) {
	for _, upload := range []bool{true, false} {
		command, err := BuildSCPCommand("prod", "/config dir/custom ssh", "/local dir/file's name", "/remote dir/file", upload, true)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(command, "scp -F "+shellQuote("/config dir/custom ssh")) || !strings.Contains(command, " -r -- ") || !strings.Contains(command, shellQuote("prod:/remote dir/file")) {
			t.Fatalf("bad command: %s", command)
		}
		localPos, remotePos := strings.Index(command, shellQuote("/local dir/file's name")), strings.Index(command, shellQuote("prod:/remote dir/file"))
		if (localPos < remotePos) != upload {
			t.Fatalf("wrong direction: %s", command)
		}
	}
}

func TestSCPRejectsMissingPaths(t *testing.T) {
	for _, args := range [][4]string{{"prod", "config", "", "."}, {"prod", "config", "file", "\n"}, {"-bad", "config", "file", "."}} {
		if _, err := BuildSCPCommand(args[0], args[1], args[2], args[3], true, false); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
