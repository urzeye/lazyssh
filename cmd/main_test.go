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

package main

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestVersionFlagOutputsVersion(t *testing.T) {
	cmd := newRootCommand(zap.NewNop().Sugar(), t.TempDir())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if !strings.Contains(out.String(), version) {
		t.Fatalf("--version output = %q, want version %q", out.String(), version)
	}
}

func TestFormatVersion(t *testing.T) {
	if got := formatVersion("v2026.11.28", "unknown"); got != "v2026.11.28" {
		t.Fatalf("formatVersion unknown = %q", got)
	}
	if got := formatVersion("v2026.11.28", "abcdef123456"); got != "v2026.11.28 (abcdef1)" {
		t.Fatalf("formatVersion commit = %q", got)
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "one", value: "1", want: true},
		{name: "true", value: "true", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "on", value: "on", want: true},
		{name: "zero", value: "0", want: false},
		{name: "false", value: "false", want: false},
		{name: "empty", value: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LAZYSSH_TEST_BOOL", tt.value)
			if got := envBool("LAZYSSH_TEST_BOOL"); got != tt.want {
				t.Fatalf("envBool() = %v, want %v", got, tt.want)
			}
		})
	}
}
