// Copyright 2016 Google Inc. All Rights Reserved.
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
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/certifi/gocertifi"
)

var httpClient http.Client

func init() {
	// Use the Root Certificates bundle from the Certifi project so we don't
	// rely on the host OS or container base images for a CA Bundle.
	// See https://certifi.io for more details.
	certPool, err := gocertifi.CACerts()
	if err != nil {
		log.Fatal(err)
	}
	httpClient = http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}
}
