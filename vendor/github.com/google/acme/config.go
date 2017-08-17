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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"golang.org/x/crypto/acme"
)

const (
	// accountFile is the default user config file name.
	accountFile = "account.json"
	// accountKey is the default user account private key file.
	accountKey = "account.key"

	rsaPrivateKey = "RSA PRIVATE KEY"
	ecPrivateKey  = "EC PRIVATE KEY"
)

// configDir is acme configuration dir.
// It may be empty string.
//
// The value is initialized at startup and is also allowed to be modified
// using -c flag, common to all subcommands.
var configDir string

func init() {
	configDir = os.Getenv("ACME_CONFIG")
	if configDir != "" {
		return
	}
	if u, err := user.Current(); err == nil {
		configDir = filepath.Join(u.HomeDir, ".config", "acme")
	}
}

// userConfig is configuration for a single ACME CA account.
type userConfig struct {
	acme.Account
	CA string `json:"ca"` // CA discovery URL

	// key is stored separately
	key crypto.Signer
}

// readConfig reads userConfig from path and a private key.
// It expects to find the key at the same location,
// by replacing path extention with ".key".
//func readConfig(name string) (*userConfig, error) {
func readConfig() (*userConfig, error) {
	b, err := ioutil.ReadFile(filepath.Join(configDir, accountFile))
	if err != nil {
		return nil, err
	}
	uc := &userConfig{}
	if err := json.Unmarshal(b, uc); err != nil {
		return nil, err
	}
	if key, err := readKey(filepath.Join(configDir, accountKey)); err == nil {
		uc.key = key
	}
	return uc, nil
}

// writeConfig writes uc to a file specified by path, creating paret dirs
// along the way. If file does not exists, it will be created with 0600 mod.
// This function does not store uc.key.
//func writeConfig(path string, uc *userConfig) error {
func writeConfig(uc *userConfig) error {
	b, err := json.MarshalIndent(uc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(configDir, accountFile), b, 0600)
}

// readKey reads a private rsa key from path.
// The key is expected to be in PEM format.
func readKey(path string) (crypto.Signer, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d, _ := pem.Decode(b)
	if d == nil {
		return nil, fmt.Errorf("no block found in %q", path)
	}
	switch d.Type {
	case rsaPrivateKey:
		return x509.ParsePKCS1PrivateKey(d.Bytes)
	case ecPrivateKey:
		return x509.ParseECPrivateKey(d.Bytes)
	default:
		return nil, fmt.Errorf("%q is unsupported", d.Type)
	}
}

// writeKey writes k to the specified path in PEM format.
// If file does not exists, it will be created with 0600 mod.
func writeKey(path string, k *ecdsa.PrivateKey) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	bytes, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return err
	}
	b := &pem.Block{Type: ecPrivateKey, Bytes: bytes}
	if err := pem.Encode(f, b); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// anyKey reads the key from file or generates a new one if gen == true.
// It returns an error if filename exists but cannot be read.
// A newly generated key is also stored to filename.
func anyKey(filename string, gen bool) (crypto.Signer, error) {
	k, err := readKey(filename)
	if err == nil {
		return k, nil
	}
	if !os.IsNotExist(err) || !gen {
		return nil, err
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return ecKey, writeKey(filename, ecKey)
}

// sameDir returns filename path placing it in the same dir as existing file.
func sameDir(existing, filename string) string {
	return filepath.Join(filepath.Dir(existing), filename)
}

// printAccount outputs account into into w using tabwriter.
func printAccount(w io.Writer, a *acme.Account, kp string) {
	tw := tabwriter.NewWriter(w, 0, 8, 0, '\t', 0)
	fmt.Fprintln(tw, "URI:\t", a.URI)
	fmt.Fprintln(tw, "Key:\t", kp)
	fmt.Fprintln(tw, "Contact:\t", strings.Join(a.Contact, ", "))
	fmt.Fprintln(tw, "Terms:\t", a.CurrentTerms)
	agreed := a.AgreedTerms
	if a.AgreedTerms == "" {
		agreed = "no"
	} else if a.AgreedTerms == a.CurrentTerms {
		agreed = "yes"
	}
	fmt.Fprintln(tw, "Accepted:\t", agreed)
	// TODO: print authorization and certificates
	tw.Flush()
}
