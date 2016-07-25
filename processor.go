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

func syncCertificates(interval int, db *bolt.DB) <-chan error {
	errc := make(chan error, 1)
	go func() {
		for {
			var err error
			time.Sleep(time.Duration(interval) * time.Second)
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
				err := processCertificate(cert, db)
				if err != nil {
					errc <- err
					continue
				}
			}
		}
	}()
	return errc
}

func processCertificateEvent(c CertificateEvent, db *bolt.DB) error {
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
	return deleteKubernetesSecret(c.Spec.Domain)
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
		log.Printf("Syncing Kubernetes secret: %s", c.Spec.Domain)
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
	if err := googleDNSClient.monitorDNSPropagation(fqdn, value, ttl); err != nil {
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
		return errors.New("Error creating Kubernetes secret: " + err.Error())
	}

	err = googleDNSClient.DeleteDNSRecord(fqdn)
	if err != nil {
		return err
	}
	return nil
}
