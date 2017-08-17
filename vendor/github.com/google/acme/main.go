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
	"flag"
	"log"
	"os"
	"strings"
	"sync"
)

// defaultDisco is the default CA directory endpoint.
// It should match one of discoAliases map keys.
const defaultDisco = "letsencrypt"

var (
	// discoAliases defines known ACME CAs.
	discoAliases = map[string]string{
		"letsencrypt":         "https://acme-v01.api.letsencrypt.org/directory",
		"letsencrypt-staging": "https://acme-staging.api.letsencrypt.org/directory",
	}

	// defaultDiscoFlag is the default value for -d argument
	// It must not be modified
	defaultDiscoFlag = discoAliasFlag(discoAliases[defaultDisco])

	// commands lists all available commands and help topics.
	// The order here is the order in which they are printed by 'acme help'.
	commands = []*command{
		cmdReg,
		cmdWho,
		cmdUpdate,
		cmdCert,
		// help commands, non-executable
		helpAccount,
		helpDisco,
	}

	exitMu     sync.Mutex // guards exitStatus
	exitStatus = 0
)

var logf = log.Printf

func errorf(format string, args ...interface{}) {
	logf(format, args...)
	setExitStatus(1)
}

func fatalf(format string, args ...interface{}) {
	errorf(format, args...)
	exit()
}

func setExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func exit() {
	os.Exit(exitStatus)
}

func main() {
	flag.Usage = usage
	flag.Parse() // catch -h argument
	log.SetFlags(0)

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}
	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Runnable() {
			addFlags(&cmd.flag)
			cmd.flag.Usage = func() { cmd.Usage() }
			cmd.flag.Parse(args[1:])
			cmd.run(cmd.flag.Args())
			exit()
			return
		}
	}

	fatalf("Unknown subcommand %q.\nRun 'acme help' for usage.\n", args[0])
}

// addFlags adds flags common to all goacmd subcommands.
// Common flag var names are of flagXxx form.
func addFlags(f *flag.FlagSet) {
	f.StringVar(&configDir, "c", configDir, "")
}

// A command is an implementation of a acme command
// like acme reg or acme whoami.
type command struct {
	// run runs the command.
	// The args are the arguments after the command name.
	run func(args []string)

	// flag is a set of flags specific to this command.
	flag flag.FlagSet

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'acme help' output.
	Short string

	// Long is the detailed command description template shown in
	// 'acme help <command>' output.
	// The template context is longTemplateData.
	Long string
}

// Name returns the command's name: the first word in the usage line.
func (c *command) Name() string {
	name := c.UsageLine
	i := strings.IndexRune(name, ' ')
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// Usage reports command's usage to stderr, including long description,
// and exits with code 2.
func (c *command) Usage() {
	help([]string{c.Name()})
	os.Exit(2)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command.
func (c *command) Runnable() bool {
	return c.run != nil
}

// discoAliasFlag is a flag which can resolve discoAliases.
type discoAliasFlag string

func (a *discoAliasFlag) String() string {
	return string(*a)
}

func (a *discoAliasFlag) Set(v string) error {
	if a, ok := discoAliases[v]; ok {
		v = a
	}
	*a = discoAliasFlag(v)
	return nil
}
