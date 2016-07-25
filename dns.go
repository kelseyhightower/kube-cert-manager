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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googledns "google.golang.org/api/dns/v1"
)

type GoogleDNSClient struct {
	domain  string
	project string
	*googledns.Service
}

func NewGoogleDNSClient(serviceAccount []byte, project, domain string) (*GoogleDNSClient, error) {
	jwtConfig, err := google.JWTConfigFromJSON(
		serviceAccount,
		googledns.NdevClouddnsReadwriteScope,
	)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &httpClient)

	jwtHTTPClient := jwtConfig.Client(ctx)
	service, err := googledns.New(jwtHTTPClient)
	if err != nil {
		return nil, err
	}

	return &GoogleDNSClient{domain, project, service}, nil
}

func (c *GoogleDNSClient) CreateDNSRecord(fqdn, value string, ttl int) error {
	zones, err := c.ManagedZones.List(c.project).Do()
	if err != nil {
		return err
	}

	zoneName := ""
	for _, zone := range zones.ManagedZones {
		if strings.HasSuffix(c.domain+".", zone.DnsName) {
			zoneName = zone.Name
		}
	}
	if zoneName == "" {
		return errors.New("Zone not found")
	}

	record := &googledns.ResourceRecordSet{
		Name:    fqdn,
		Rrdatas: []string{value},
		Ttl:     int64(ttl),
		Type:    "TXT",
	}

	change := &googledns.Change{
		Additions: []*googledns.ResourceRecordSet{record},
	}

	changesCreateCall, err := c.Changes.Create(c.project, zoneName, change).Do()
	if err != nil {
		return err
	}

	for changesCreateCall.Status == "pending" {
		time.Sleep(time.Second)
		changesCreateCall, err = c.Changes.Get(c.project, zoneName, changesCreateCall.Id).Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GoogleDNSClient) DeleteDNSRecord(fqdn string) error {
	zones, err := c.ManagedZones.List(c.project).Do()
	if err != nil {
		return err
	}

	zoneName := ""
	for _, zone := range zones.ManagedZones {
		if strings.HasSuffix(c.domain+".", zone.DnsName) {
			zoneName = zone.Name
		}
	}
	if zoneName == "" {
		return errors.New("Zone not found")
	}

	records, err := c.ResourceRecordSets.List(c.project, zoneName).Do()
	if err != nil {
		return err
	}

	matchingRecords := []*googledns.ResourceRecordSet{}
	for _, record := range records.Rrsets {
		if record.Type == "TXT" && record.Name == fqdn {
			matchingRecords = append(matchingRecords, record)
		}
	}

	for _, record := range matchingRecords {
		change := &googledns.Change{
			Deletions: []*googledns.ResourceRecordSet{record},
		}
		_, err = c.Changes.Create(c.project, zoneName, change).Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GoogleDNSClient) monitorDNSPropagation(fqdn, value string, ttl int) error {
	dnsClient := new(dns.Client)
	dnsClient.Net = "tcp"
	dnsClient.Timeout = time.Second * 10

	ns, err := net.LookupNS(c.domain)
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
