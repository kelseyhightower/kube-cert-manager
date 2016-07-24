package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/certifi/gocertifi"
	"github.com/google/acme"
)

var (
	certExpiry = 365 * 12 * time.Hour
	certBundle = false
)

var (
	ErrNotFound = errors.New("account not found")
)

type Account struct {
	Account        *acme.Account
	AccountKey     *rsa.PrivateKey
	Email          string
	Certificate    []byte
	CertificateKey *rsa.PrivateKey
	CertificateURL string
	Domain         string
}

type ACMEClient struct {
	acme.Client
	endpoint *acme.Endpoint
}

func newACMEClient(discoveryURL string, key *rsa.PrivateKey) (*ACMEClient, error) {
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

	endpoint, err := getEndpoint(discoveryURL)
	if err != nil {
		return nil, err
	}

	acmeClient := acme.Client{
		Client: httpClient,
		Key:    key,
	}

	return &ACMEClient{acmeClient, &endpoint}, nil
}

func (c *ACMEClient) Register(account *acme.Account) error {
	return c.Client.Register(c.endpoint.RegURL, account)
}

func (c *ACMEClient) Authorize(url, domain string) (*acme.Authorization, *acme.Challenge, error) {
	authorization, err := c.Client.Authorize(url, domain)
	if err != nil {
		return nil, nil, err
	}

	var challenge *acme.Challenge
	for _, c := range authorization.Challenges {
		if c.Type == "dns-01" {
			challenge = &c
			break
		}
	}
	if challenge == nil {
		return nil, nil, errors.New("no supported challenge found")
	}
	return authorization, challenge, err
}

func (c *ACMEClient) Accept(authorization *acme.Authorization, challenge *acme.Challenge) error {
	if _, err := c.Client.Accept(challenge); err != nil {
		return err
	}

	for {
		authorization, err := c.GetAuthz(authorization.URI)
		if err != nil {
			return err
		}

		if authorization.Status == acme.StatusInvalid {
			return fmt.Errorf("could not authorize")
		}
		if authorization.Status != acme.StatusValid {
			time.Sleep(time.Duration(3) * time.Second)
			continue
		}
		break
	}
	return nil
}

func (c *ACMEClient) CreateCert(domain string, key *rsa.PrivateKey) ([]byte, string, error) {
	req := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: domain},
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, req, key)
	if err != nil {
		return nil, "", err
	}

	cert, certURL, err := c.Client.CreateCert(c.endpoint.CertURL, csr, certExpiry, certBundle)
	if err != nil {
		return nil, "", err
	}

	certPool, err := gocertifi.CACerts()
	if err != nil {
		return nil, "", err
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}

	if cert == nil {
		for {
			cert, err = acme.FetchCert(&httpClient, certURL, certBundle)
			if err == nil {
				break
			}
			d := 3 * time.Second
			if re, ok := err.(acme.RetryError); ok {
				d = time.Duration(re)
			}
			time.Sleep(d)
		}
	}

	var pemEncodedCert []byte
	for _, b := range cert {
		b = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b})
		pemEncodedCert = append(pemEncodedCert, b...)
	}

	return pemEncodedCert, certURL, nil
}

func (c *ACMEClient) RenewCert(certURL string) ([]byte, error) {
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

	cert, err := acme.FetchCert(&httpClient, certURL, certBundle)
	if err != nil {
		return nil, err
	}
	var pemEncodedCert []byte
	for _, b := range cert {
		b = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b})
		pemEncodedCert = append(pemEncodedCert, b...)
	}
	return pemEncodedCert, nil
}

func newAccount(email, domain string) (*Account, error) {
	var account *Account

	accountKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return account, err
	}

	certificateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return account, err
	}

	acmeAccount := &acme.Account{
		Contact: []string{fmt.Sprintf("%s:%s", "mailto", email)},
	}
	account = &Account{
		Account:        acmeAccount,
		AccountKey:     accountKey,
		Email:          email,
		CertificateKey: certificateKey,
		Domain:         domain,
	}
	return account, nil
}

func findAccount(domain string, db *bolt.DB) (*Account, error) {
	var account *Account
	err := db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte("Accounts")).Get([]byte(domain))
		if data == nil {
			return nil
		}
		decoder := gob.NewDecoder(bytes.NewReader(data))
		err := decoder.Decode(&account)
		if err != nil {
			return err
		}
		return nil
	})
	return account, err
}

func saveAccount(account *Account, db *bolt.DB) error {
	data := new(bytes.Buffer)
	enc := gob.NewEncoder(data)
	err := enc.Encode(account)
	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if err != nil {
			return err
		}
		bucket := tx.Bucket([]byte("Accounts"))
		err = bucket.Put([]byte(account.Domain), data.Bytes())
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func deleteAccount(domain string, db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("Accounts")).Delete([]byte(domain))
	})
	return err
}

func getEndpoint(url string) (acme.Endpoint, error) {
	certPool, err := gocertifi.CACerts()
	if err != nil {
		return acme.Endpoint{}, err
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}

	return acme.Discover(&httpClient, url)
}
