package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/certifi/gocertifi"
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

	certPool, err := gocertifi.CACerts()
	if err != nil {
		return nil, err
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
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

func waitDNS(fqdn, value string, ttl int) error {
	// Add a timeout so we don't block forever.
	dnsClient := new(dns.Client)
	dnsClient.Net = "tcp"
	dnsClient.Timeout = time.Second * 10

	for {
		m := new(dns.Msg)
		m.SetQuestion(fqdn, dns.TypeTXT)
		m.SetEdns0(4096, false)
		m.RecursionDesired = false

		nameservers := []string{
			"8.8.8.8:53",
			"8.8.4.4:53",
		}

		for _, ns := range nameservers {
			for {
				var found bool
				in, _, err := dnsClient.Exchange(m, ns)
				if err != nil {
					log.Println(err)
					time.Sleep(5 * time.Second)
					continue
				}

				if len(in.Answer) == 0 {
					time.Sleep(5 * time.Second)
					continue
				}

				for _, rr := range in.Answer {
					if txt, ok := rr.(*dns.TXT); ok {
						if strings.Join(txt.Txt, "") == value {
							log.Printf("matching TXT record found [%s]", ns)
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
		}
		break
	}

	time.Sleep(30 * time.Second)
	return nil
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
