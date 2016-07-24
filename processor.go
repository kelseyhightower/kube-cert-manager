package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/acme"
)

func syncCertificates(db *bolt.DB) <-chan error {
	errc := make(chan error, 1)
	go func() {
		for {
			var err error
			time.Sleep(30 * time.Second)
			log.Println("Starting reconciliation loop...")
			var certificates []Certificate
			for {
				certificates, err = getCertificates()
				if err != nil {
					errc <- err
					time.Sleep(5 * time.Second)
					continue
				}
				break
			}
			for _, cert := range certificates {
				log.Printf("Processing certificate: %s", cert.Metadata["name"])
				err := processCertificate(cert, db)
				if err != nil {
					errc <- err
					continue
				}
			}
			log.Println("Reconciliation loop complete.")
		}
	}()
	return errc
}

func processCertificateEvent(c CertificateEvent, db *bolt.DB) error {
	switch {
	case c.Type == "ADDED":
		return processCertificate(c.Object, db)
	case c.Type == "DELETED":
		log.Println("Deleting certificate...")
	}
	return nil
}

func processCertificate(c Certificate, db *bolt.DB) error {
	log.Println("Looking up ACME account using:", c.Spec.Email)
	account, err := findAccount(c.Spec.Email, db)
	if err != nil {
		return err
	}

	if account == nil {
		log.Printf("ACME account for %s not found. Creating new account...", c.Spec.Email)
		account, err = newAccount(c.Spec.Email)
		if err != nil {
			return err
		}
	}

	acmeClient, err := newACMEClient(discoveryURL, account.AccountKey)
	if err != nil {
		return errors.New("Error creating ACME client:" + err.Error())
	}

	if account.Account.URI == "" {
		err = acmeClient.Register(account.Account)
		if err != nil {
			return errors.New("Error registering account" + err.Error())
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
		log.Printf("Renewing certificate for %s...", c.Spec.Domain)
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
		err = syncKubernetesSecret(c.Spec.Domain, account.Certificate, key)
		if err != nil {
			return errors.New("Error creating Kubernetes secret " + err.Error())
		}
		return nil
	}

	authorization, challenge, err := acmeClient.Authorize(account.Account.Authz, c.Spec.Domain)
	if err != nil {
		return errors.New("Error authorizing account: " + err.Error())
	}

	jwkThumbprint := acme.JWKThumbprint(&account.AccountKey.PublicKey)
	fqdn, value, ttl := DNSChallengeRecord(c.Spec.Domain, challenge.Token, jwkThumbprint)

	serviceAccount, err := getServiceAccountFromSecret(c.Spec.ServiceAccount)
	if err != nil {
		return errors.New("Error getting service account from secret" + err.Error())
	}

	googleDNSClient, err := NewGoogleDNSClient(serviceAccount, c.Spec.Project, c.Spec.Domain)
	if err != nil {
		return errors.New("Error creating google DNS client" + err.Error())
	}

	// Cleaning up the DNS challenge here creates a race between two processes
	// managing DNS challenge records.
	googleDNSClient.DeleteDNSRecord(fqdn)

	err = googleDNSClient.CreateDNSRecord(fqdn, value, ttl)
	if err != nil {
		return err
	}

	// We need to make sure the DNS challenge record has propagated across the
	// authoritative nameservers for the fqdn before accepting the ACME challenge.
	if err := waitDNS(fqdn, value, ttl); err != nil {
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
	err = syncKubernetesSecret(c.Spec.Domain, account.Certificate, key)
	if err != nil {
		return errors.New("Error creating Kubernetes secret " + err.Error())
	}

	err = googleDNSClient.DeleteDNSRecord(fqdn)
	if err != nil {
		return err
	}
	return nil
}
