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
	cmdUpdate = &command{
		run:       runUpdate,
		UsageLine: "update [-c config] [-accept] [contact [contact ...]]",
		Short:     "update account data",
		Long: `
Update modifies account contact info and accepts the current CA
service agreement which can be seen using whoami command.

Use -accept argument to indicate that the account holder agrees with
the proposed CA's Terms and Conditions (the agreement).

Default location of the config dir is
{{.ConfigDir}}.
		`,
	}

	updateAccept bool
)

func init() {
	cmdUpdate.flag.BoolVar(&updateAccept, "accept", updateAccept, "")
}

func runUpdate(args []string) {
	uc, err := readConfig()
	if err != nil {
		fatalf("read config: %v", err)
	}
	if uc.key == nil {
		fatalf("no key found for %s", uc.URI)
	}

	client := acme.Client{Key: uc.key}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if updateAccept {
		a, err := client.GetReg(ctx, uc.URI)
		if err != nil {
			fatalf(err.Error())
		}
		uc.Account = *a
		uc.AgreedTerms = a.CurrentTerms
	}
	if len(args) != 0 {
		uc.Contact = args
	}

	a, err := client.UpdateReg(ctx, &uc.Account)
	if err != nil {
		fatalf(err.Error())
	}
	uc.Account = *a
	if err := writeConfig(uc); err != nil {
		fatalf("write config: %v", err)
	}
	printAccount(os.Stdout, &uc.Account, filepath.Join(configDir, accountKey))
}
