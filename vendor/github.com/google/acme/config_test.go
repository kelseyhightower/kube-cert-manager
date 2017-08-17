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
	"io/ioutil"
	"reflect"
	"testing"

	"golang.org/x/crypto/acme"
)

func TestConfigReadWrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "acme-config")
	if err != nil {
		t.Fatal(err)
	}
	configDir = dir
	write := &userConfig{
		Account: acme.Account{
			URI:            "https://example.com/acme/reg/123",
			Contact:        []string{"mailto:dude@example.com"},
			AgreedTerms:    "http://agreed",
			CurrentTerms:   "https://terms",
			Authz:          "https://authz",
			Authorizations: "https://authorizations",
			Certificates:   "https://certificates",
		},
	}
	if err := writeConfig(write); err != nil {
		t.Fatal(err)
	}
	read, err := readConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, write) {
		t.Errorf("read: %+v\nwant: %+v", read, write)
	}
}
