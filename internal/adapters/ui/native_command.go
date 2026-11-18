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
	"runtime"
	"strings"
)

func formatNativeCommand(executable string, args []string) string {
	if runtime.GOOS == windowsOS {
		encoded := make([]string, 0, len(args))
		for _, arg := range args {
			encoded = append(encoded, quoteWindowsArgument(arg))
		}
		// ProcessStartInfo bypasses PowerShell's version-dependent native argv
		// marshaling. Inherit the console for interactive authentication.
		return "& { $ErrorActionPreference = 'Stop'; $lazysshStart = New-Object System.Diagnostics.ProcessStartInfo -Property @{ FileName = " + shellQuote(executable) +
			"; Arguments = " + shellQuote(strings.Join(encoded, " ")) +
			"; UseShellExecute = $false }; $lazysshProcess = [System.Diagnostics.Process]::Start($lazysshStart); try { $lazysshProcess.WaitForExit(); $global:LASTEXITCODE = $lazysshProcess.ExitCode } finally { $lazysshProcess.Dispose() } }"
	}
	parts := []string{executable}
	for _, arg := range args {
		if arg == "--" || (len(arg) == 2 && arg[0] == '-' && ((arg[1] >= 'A' && arg[1] <= 'Z') || (arg[1] >= 'a' && arg[1] <= 'z'))) {
			parts = append(parts, arg)
		} else {
			parts = append(parts, shellQuote(arg))
		}
	}
	return strings.Join(parts, " ")
}

func quoteWindowsArgument(value string) string {
	var result strings.Builder
	result.WriteByte('"')
	slashes := 0
	writeSlashes := func() {
		if slashes == 0 {
			return
		}
		// Put backslashes outside quoted text and immediately before a
		// quote. CRT and MSYS then both consume each doubled pair once.
		result.WriteByte('"')
		result.WriteString(strings.Repeat("\\", 2*slashes))
		result.WriteByte('"')
		slashes = 0
	}
	for _, ch := range value {
		if ch == '\\' {
			slashes++
			continue
		}
		writeSlashes()
		if ch == '"' {
			result.WriteByte('\\')
		}
		result.WriteRune(ch)
	}
	writeSlashes()
	result.WriteByte('"')
	return result.String()
}
