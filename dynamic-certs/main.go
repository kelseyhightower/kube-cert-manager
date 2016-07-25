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
	"flag"
	"io"
	"log"
	"net/http"
)

var (
	httpAddr string
	tlsCert  string
	tlsKey   string
)

func main() {
	flag.StringVar(&httpAddr, "http", ":443", "HTTP Listen address.")
	flag.StringVar(&tlsCert, "tls-cert", "/etc/tls/server.pem", "TLS certificate path")
	flag.StringVar(&tlsKey, "tls-key", "/etc/tls/server.key", "TLS private key path")
	flag.Parse()

	log.Println("Initializing application...")

	cm, err := NewCertificateManager(tlsCert, tlsKey)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "Hello, World!\n")
	})

	server := http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: cm.GetCertificate,
		},
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServeTLS("", "")
	}()

	log.Printf("HTTPS listener on %s...", httpAddr)

	for {
		select {
		case err := <-errChan:
			log.Fatal(err)
		case err := <-cm.Error:
			log.Println(err)
		}
	}
}
