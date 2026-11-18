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

// scp_transport supplies a controlled server to the real SCP receiver, without a network.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := send(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func send() error {
	in := bufio.NewReader(os.Stdin)
	ack := func() error {
		code, err := in.ReadByte()
		if err != nil {
			return err
		}
		if code != 0 {
			message, _ := in.ReadString('\n')
			return fmt.Errorf("SCP rejected transfer: %s", message)
		}
		return nil
	}
	if err := ack(); err != nil {
		return err
	}
	const contents = "SCP regression payload\n"
	if _, err := fmt.Fprintf(os.Stdout, "C0644 %d %s\n", len(contents), os.Getenv("LAZYSSH_SCP_FILENAME")); err != nil {
		return err
	}
	if err := ack(); err != nil {
		return err
	}
	if _, err := io.WriteString(os.Stdout, contents+"\x00"); err != nil {
		return err
	}
	return ack()
}
