package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	apiHost                   = "http://127.0.0.1:8001"
	certificatesEndpoint      = "/apis/stable.hightower.com/v1/namespaces/default/certificates"
	certificatesWatchEndpoint = "/apis/stable.hightower.com/v1/namespaces/default/certificates?watch=true"
	secretsEndpoint           = "/api/v1/namespaces/default/secrets"
)

type CertificateEvent struct {
	Type   string      `json:"type"`
	Object Certificate `json:"object"`
}

type Certificate struct {
	ApiVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   map[string]string `json:"metadata"`
	Spec       CertificateSpec   `json:"spec"`
}

type CertificateSpec struct {
	Domain         string `json:"domain"`
	Email          string `json:"email"`
	Project        string `json:"project"`
	ServiceAccount string `json:"serviceAccount"`
}

type CertificateList struct {
	ApiVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   map[string]string `json:"metadata"`
	Items      []Certificate     `json:"items"`
}

type Secret struct {
	Kind       string            `json:"kind"`
	ApiVersion string            `json:"apiVersion"`
	Metadata   map[string]string `json:"metadata"`
	Data       map[string]string `json:"data"`
	Type       string            `json:"type"`
}

func getCertificates() ([]Certificate, error) {
	resp, err := http.Get(apiHost + certificatesEndpoint)
	if err != nil {
		return nil, err
	}

	var certList CertificateList
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&certList)
	if err != nil {
		return nil, err
	}

	return certList.Items, nil
}

func watchCertificateEvents() (<-chan CertificateEvent, <-chan error) {
	events := make(chan CertificateEvent)
	errc := make(chan error, 1)
	go func() {
		for {
			resp, err := http.Get(apiHost + certificatesWatchEndpoint)
			if err != nil {
				errc <- err
				time.Sleep(5 * time.Second)
				continue
			}
			if resp.StatusCode != 200 {
				errc <- errors.New("Invalid status code: " + resp.Status)
				time.Sleep(5 * time.Second)
				continue
			}

			decoder := json.NewDecoder(resp.Body)
			for {
				var event CertificateEvent
				err = decoder.Decode(&event)
				if err != nil {
					errc <- err
					break
				}
				events <- event
			}
		}
	}()

	return events, errc
}

func getServiceAccountFromSecret(name string) ([]byte, error) {
	resp, err := http.Get(apiHost + secretsEndpoint + "/" + name)
	if err != nil {
		return nil, err
	}
	var secret Secret
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&secret)
	if err != nil {
		return nil, err
	}

	data, ok := secret.Data["service-account.json"]
	if !ok {
		return nil, errors.New("Secret key service-account.json not found")
	}
	serviceAccount, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return serviceAccount, nil
}

func checkSecret(name string) (bool, error) {
	resp, err := http.Get(apiHost + secretsEndpoint + "/" + name)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != 200 {
		return false, nil
	}
	return true, nil
}

func syncKubernetesSecret(domain string, cert, key []byte) error {
	metadata := make(map[string]string)
	metadata["name"] = domain

	data := make(map[string]string)
	data["tls.crt"] = base64.StdEncoding.EncodeToString(cert)
	data["tls.key"] = base64.StdEncoding.EncodeToString(key)

	secret := &Secret{
		ApiVersion: "v1",
		Data:       data,
		Kind:       "Secret",
		Metadata:   metadata,
		Type:       "kubernetes.io/tls",
	}

	resp, err := http.Get(apiHost + secretsEndpoint + "/" + domain)
	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		// compare current cert
		var currentSecret Secret
		d, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		err = json.Unmarshal(d, &currentSecret)
		if err != nil {
			return err
		}
		if currentSecret.Data["tls.crt"] != secret.Data["tls.crt"] || currentSecret.Data["tls.key"] != secret.Data["tls.key"] {
			log.Printf("Secret [%s] out of sync.", domain)
			currentSecret.Data = secret.Data
			b := make([]byte, 0)
			body := bytes.NewBuffer(b)
			err := json.NewEncoder(body).Encode(currentSecret)
			if err != nil {
				return err
			}
			req, err := http.NewRequest("PUT", apiHost+secretsEndpoint+"/"+domain, body)
			if err != nil {
				return err
			}
			req.Header.Add("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return errors.New("Updating secret failed:" + resp.Status)
			}
			log.Println("Syncing secret [%s] complete.", domain)
		}
		return nil
	}

	if resp.StatusCode == 404 {
		log.Println("Secret [%s] not found. Creating...", domain)
		b := make([]byte, 0)
		body := bytes.NewBuffer(b)
		err := json.NewEncoder(body).Encode(secret)
		if err != nil {
			return err
		}

		resp, err := http.Post(apiHost+secretsEndpoint, "application/json", body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 201 {
			return errors.New("Secrets: Unexpected HTTP status code" + resp.Status)
		}
		return nil
	}
	return nil
}
