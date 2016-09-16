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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/acme"
)

// processorLock ensures that Certificate reconciliation and Certificate
// event processing does not happen at the same time.
var processorLock = &sync.Mutex{}

func reconcileCertificates(interval int, db *bolt.DB, done chan struct{}, wg *sync.WaitGroup) {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(interval) * time.Second):
				err := syncCertificates(db)
				if err != nil {
					log.Println(err)
				}
			case <-done:
				wg.Done()
				log.Println("Stopped reconciliation loop.")
				return
			}
		}
	}()
}

func watchCertificateEvents(db *bolt.DB, done chan struct{}, wg *sync.WaitGroup) {
	events, watchErrs := monitorCertificateEvents()
	go func() {
		for {
			select {
			case event := <-events:
				err := processCertificateEvent(event, db)
				if err != nil {
					log.Println(err)
				}
			case err := <-watchErrs:
				log.Println(err)
			case <-done:
				wg.Done()
				log.Println("Stopped certificate event watcher.")
				return
			}
		}
	}()
}

func syncCertificates(db *bolt.DB) error {
	processorLock.Lock()
	defer processorLock.Unlock()

	certificates, err := getCertificates()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, cert := range certificates {
		wg.Add(1)
		go func(cert Certificate) {
			defer wg.Done()
			err := processCertificate(cert, db)
			if err != nil {
				log.Println(err)
			}
		}(cert)
	}
	wg.Wait()
	return nil
}

func processCertificateEvent(c CertificateEvent, db *bolt.DB) error {
	processorLock.Lock()
	defer processorLock.Unlock()
	switch {
	case c.Type == "ADDED":
		return processCertificate(c.Object, db)
	case c.Type == "DELETED":
		return deleteCertificate(c.Object, db)
	}
	return nil
}

func deleteCertificate(c Certificate, db *bolt.DB) error {
	log.Printf("Deleting Let's Encrypt account: %s", c.Spec.Domain)
	err := deleteAccount(c.Spec.Domain, db)
	if err != nil {
		return errors.New("Error deleting the Let's Encrypt account " + err.Error())
	}
	log.Printf("Deleting Kubernetes TLS secret: %s", c.Spec.Domain)
	return deleteKubernetesSecret(c)
}

func processCertificate(c Certificate, db *bolt.DB) error {
	account, err := findAccount(c.Spec.Domain, db)
	if err != nil {
		return err
	}

	if account == nil {
		log.Printf("Creating new Let's Encrypt account: %s", c.Spec.Domain)
		account, err = newAccount(c.Spec.Email, c.Spec.Domain)
		if err != nil {
			return err
		}
	}

	acmeClient, err := newACMEClient(discoveryURL, account.AccountKey)
	if err != nil {
		return errors.New("Error creating ACME client: " + err.Error())
	}

	if account.Account.URI == "" {
		err = acmeClient.Register(account.Account)
		if err != nil {
			return errors.New("Error registering account: " + err.Error())
		}
		account.Account.AgreedTerms = account.Account.CurrentTerms
		err = acmeClient.UpdateReg(account.Account.URI, account.Account)
		if err != nil {
			return errors.New("Error agreeing to terms" + err.Error())
		}

		err = saveAccount(account, db)
		if err != nil {
			return errors.New("Error saving account" + err.Error())
		}
	}

	if account.CertificateURL != "" {
		cert, err := acmeClient.RenewCert(account.CertificateURL)
		if err != nil {
			return errors.New("Error renewing certificate" + err.Error())
		}
		account.Certificate = cert
		key := pem.EncodeToMemory(&pem.Block{
			Type:    "RSA PRIVATE KEY",
			Headers: nil,
			Bytes:   x509.MarshalPKCS1PrivateKey(account.CertificateKey),
		})
		err = syncKubernetesSecret(c, account.Certificate, key)
		if err != nil {
			return errors.New("Error creating Kubernetes secret: " + err.Error())
		}
		return nil
	}

	authorization, challenge, err := acmeClient.Authorize(account.Account.Authz, c.Spec.Domain)
	if err != nil {
		return errors.New("Error authorizing account: " + err.Error())
	}

	jwkThumbprint := acme.JWKThumbprint(&account.AccountKey.PublicKey)
	fqdn, value, ttl := DNSChallengeRecord(c.Spec.Domain, challenge.Token, jwkThumbprint)

	dnsExecClient := &dnsClient{
		c.Spec.Domain,
		c.Spec.Provider,
		c.Spec.Secret,
		c.Spec.SecretKey,
		c.Metadata["namespace"],
	}

	// Cleaning up the DNS challenge here creates a race between two processes
	// managing DNS challenge records.
	dnsExecClient.deleteRecord(fqdn, value, ttl)

	err = dnsExecClient.createRecord(fqdn, value, ttl)
	if err != nil {
		return err
	}

	// We need to make sure the DNS challenge record has propagated across the
	// authoritative nameservers for the fqdn before accepting the ACME challenge.
	if err := dnsExecClient.monitorDNSPropagation(fqdn, value, ttl); err != nil {
		return err
	}

	if err := acmeClient.Accept(authorization, challenge); err != nil {
		return err
	}

	cert, certURL, err := acmeClient.CreateCert(c.Spec.Domain, account.CertificateKey)
	if err != nil {
		return err
	}
	account.Certificate = cert
	account.CertificateURL = certURL

	err = saveAccount(account, db)
	if err != nil {
		return err
	}

	key := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(account.CertificateKey),
	})
	err = syncKubernetesSecret(c, account.Certificate, key)
	if err != nil {
		return errors.New("Error creating Kubernetes secret: " + err.Error())
	}

	err = dnsExecClient.deleteRecord(fqdn, value, ttl)
	if err != nil {
		return err
	}
	return nil
}
