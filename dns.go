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
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/miekg/dns"
)

type dnsClient struct {
	domain    string
	provider  string
	secret    string
	secretKey string
}

func newDNSClient(provider, domain, secret, secretKey string) (*dnsClient, error) {
	return &dnsClient{domain, provider, secret, secretKey}, nil
}

func envVar(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func (c *dnsClient) createRecord(fqdn, value string, ttl int) error {
	providerConfig, err := getDNSConfigFromSecret(c.secret, c.secretKey)
	if err != nil {
		return errors.New("Error getting dns config from secret" + err.Error())
	}
	env := []string{
		envVar("APIVERSION", "v1"),
		envVar("COMMAND", "CREATE"),
		envVar("DOMAIN", c.domain),
		envVar("FQDN", fqdn),
		envVar("TOKEN", value),
	}

	cmd := &exec.Cmd{
		Path:  filepath.Join("/", c.provider),
		Env:   env,
		Stdin: bytes.NewReader(providerConfig),
	}
	_, err = cmd.Output()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return errors.New(string(exitError.Stderr))
		}
		return err
	}
	return nil
}

func (c *dnsClient) deleteRecord(fqdn, value string, ttl int) error {
	providerConfig, err := getDNSConfigFromSecret(c.secret, c.secretKey)
	if err != nil {
		return errors.New("Error getting dns config from secret" + err.Error())
	}
	env := []string{
		envVar("APIVERSION", "v1"),
		envVar("COMMAND", "DELETE"),
		envVar("DOMAIN", c.domain),
		envVar("FQDN", fqdn),
		envVar("TOKEN", value),
	}

	cmd := &exec.Cmd{
		Path:  filepath.Join("/", c.provider),
		Env:   env,
		Stdin: bytes.NewReader(providerConfig),
	}
	_, err = cmd.Output()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return errors.New(string(exitError.Stderr))
		}
		return err
	}
	return nil
}

func (c *dnsClient) monitorDNSPropagation(fqdn, value string, ttl int) error {
	dnsClient := new(dns.Client)
	dnsClient.Net = "tcp"
	dnsClient.Timeout = time.Second * 10

	suffix, err := publicsuffix.EffectiveTLDPlusOne(strings.TrimSuffix(fqdn, "."))
	if err != nil {
		return err
	}
	ns, err := net.LookupNS(dns.Fqdn(suffix))
	if err != nil {
		return err
	}
	nameservers := make([]string, 0)
	for _, s := range ns {
		nameservers = append(nameservers, net.JoinHostPort(s.Host, "53"))
	}

	log.Printf("Monitoring %s DNS propagation: %s", fqdn, strings.Join(nameservers, " "))

	dnsMsg := new(dns.Msg)
	dnsMsg.SetQuestion(fqdn, dns.TypeTXT)
	dnsMsg.SetEdns0(4096, false)
	dnsMsg.RecursionDesired = false

	var wg sync.WaitGroup
	for _, ns := range nameservers {
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()
			for {
				in, _, err := dnsClient.Exchange(dnsMsg, ns)
				if err != nil {
					log.Println(err)
					time.Sleep(1 * time.Second)
					continue
				}

				if len(in.Answer) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				for _, rr := range in.Answer {
					if txt, ok := rr.(*dns.TXT); ok {
						if strings.Join(txt.Txt, "") == value {
							log.Printf("%s DNS-01 challenge complete on %s", c.domain, ns)
							return
						}
					}
				}
			}
		}(ns)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Wait until the TTL expires to be sure Let's Encrypt picks up the
		// right TXT record.
		time.Sleep(time.Duration(ttl) * time.Second)
		log.Printf("%s DNS propagation complete.", fqdn)
		return nil
	case <-time.After(300 * time.Second):
		return fmt.Errorf("Timeout waiting for %s DNS propagation", fqdn)
	}
}

func DNSChallengeRecord(domain, token, jwkThumbprint string) (string, string, int) {
	fqdn := fmt.Sprintf("_acme-challenge.%s.", domain)
	keyAuthorization := fmt.Sprintf("%s.%s", token, jwkThumbprint)
	keyAuthorizationShaBytes := sha256.Sum256([]byte(keyAuthorization))
	value := base64.URLEncoding.EncodeToString(keyAuthorizationShaBytes[:sha256.Size])
	value = strings.TrimRight(value, "=")
	ttl := 30
	return fqdn, value, ttl
}
