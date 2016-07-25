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
