// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
)

var (
	cmdWho = &command{
		run:       runWhoami,
		UsageLine: "whoami [-c config]",
		Short:     "display info about the key holder",
		Long: `
Whoami makes a request to the ACME server signed with a private key
found in the config file and displays the formatted results.

It is a simple way to verify the validity of an account key.

Default location of the config dir is {{.ConfigDir}}.
		`,
	}
)

func runWhoami([]string) {
	uc, err := readConfig()
	if err != nil {
		fatalf("read config: %v", err)
	}
	if uc.key == nil {
		fatalf("no key found for %s", uc.URI)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := acme.Client{Key: uc.key}
	a, err := client.GetReg(ctx, uc.URI)
	if err != nil {
		fatalf(err.Error())
	}
	printAccount(os.Stdout, a, filepath.Join(configDir, accountKey))
}
