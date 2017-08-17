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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
)

var (
	cmdReg = &command{
		run:       runReg,
		UsageLine: "reg [-c config] [-gen] [-accept] [-d url] [contact [contact ...]]",
		Short:     "new account registration",
		Long: `
Reg creates a new account at a CA using the discovery URL
specified with -d argument. The default value is {{.DefaultDisco}}.
For more information about the discovery run acme help disco.

Upon successful registration, a new config will be written to {{.AccountFile}}
in the directory specified with -c argument. Default location of the config dir
is {{.ConfigDir}}.
If the config dir does not exist, it will be created.

Contact arguments can be anything: email, phone number, etc.

The -gen flag will generate an ECDSA P-256 keypair to use as the account key.

If -gen flag is not specified, and a file named account.key containing
a PEM-encoded ECDSA or RSA private key does not exist, the command will exit
with an error.

The registration may require the user to agree to the CA Terms of Service (TOS).
If so, and the -accept argument is not provided, the command prompts the user
with a TOS URL provided by the CA.

See also: acme help account.
		`,
	}

	regDisco  = defaultDiscoFlag
	regGen    bool
	regAccept bool
)

func init() {
	cmdReg.flag.Var(&regDisco, "d", "")
	cmdReg.flag.BoolVar(&regGen, "gen", regGen, "")
	cmdReg.flag.BoolVar(&regAccept, "accept", regAccept, "")
}

func runReg(args []string) {
	key, err := anyKey(filepath.Join(configDir, accountKey), regGen)
	if err != nil {
		fatalf("account key: %v", err)
	}
	uc := &userConfig{
		Account: acme.Account{Contact: args},
		key:     key,
	}

	prompt := ttyPrompt
	if regAccept {
		prompt = acme.AcceptTOS
	}
	client := &acme.Client{
		Key:          uc.key,
		DirectoryURL: string(regDisco),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	a, err := client.Register(ctx, &uc.Account, prompt)
	if err != nil {
		fatalf("%v", err)
	}
	uc.Account = *a
	if err := writeConfig(uc); err != nil {
		errorf("write config: %v", err)
	}
}

func ttyPrompt(tos string) bool {
	fmt.Println("CA requires acceptance of their Terms and Services agreement:")
	fmt.Println(tos)
	fmt.Print("Do you accept? (Y/n) ")
	var a string
	if _, err := fmt.Scanln(&a); err != nil {
		return false
	}
	a = strings.ToLower(a)
	return strings.HasPrefix(a, "y")
}
