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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	apiHost                   = "http://127.0.0.1:8001"
	certificatesEndpoint      = "/apis/stable.hightower.com/v1/certificates"
	certificatesWatchEndpoint = "/apis/stable.hightower.com/v1/certificates?watch=true"
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
	Domain    string `json:"domain"`
	Email     string `json:"email"`
	Provider  string `json:"provider"`
	Secret    string `json:"secret"`
	SecretKey string `json:"secretKey"`
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
	var resp *http.Response
	var err error
	for {
		resp, err = http.Get(apiHost + certificatesEndpoint)
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	var certList CertificateList
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&certList)
	if err != nil {
		return nil, err
	}

	return certList.Items, nil
}

func monitorCertificateEvents() (<-chan CertificateEvent, <-chan error) {
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

func getDNSConfigFromSecret(name, namespace, key string) ([]byte, error) {
	resp, err := http.Get(certificateEndpoint(namespace, name))
	if err != nil {
		return nil, err
	}
	var secret Secret
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&secret)
	if err != nil {
		return nil, err
	}

	data, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("Secret key %s not found", key)
	}
	config, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func deleteKubernetesSecret(c Certificate) error {

	req, err := http.NewRequest("DELETE", certificateEndpoint(c.Metadata["namespace"], c.Spec.Domain), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Deleting %s secret failed: %s", c.Spec.Domain, resp.Status)
	}
	return nil
}

func certificateEndpoint(namespace string, name string) string {
	return apiHost + "/api/v1/namespaces/" + namespace + "/secrets/" + name
}

func syncKubernetesSecret(requested Certificate, cert, key []byte) error {
	metadata := make(map[string]string)
	metadata["name"] = requested.Spec.Domain

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
	endPoint := certificateEndpoint(requested.Metadata["namespace"], requested.Spec.Domain)
	fmt.Println("Secret endpoint is: " + endPoint)
	resp, err := http.Get(endPoint)
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
			log.Printf("%s secret out of sync.", requested.Spec.Domain)
			currentSecret.Data = secret.Data
			b := make([]byte, 0)
			body := bytes.NewBuffer(b)
			err := json.NewEncoder(body).Encode(currentSecret)
			if err != nil {
				return err
			}
			req, err := http.NewRequest("PUT", endPoint, body)
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
			log.Printf("Syncing %s secret complete.", requested.Spec.Domain)
		}
		return nil
	}

	if resp.StatusCode == 404 {
		log.Printf("%s secret missing.", requested.Spec.Domain)
		var b []byte
		body := bytes.NewBuffer(b)
		err := json.NewEncoder(body).Encode(secret)
		if err != nil {
			return err
		}

		resp, err := http.Post(apiHost+"/api/v1/namespaces/"+requested.Metadata["namespace"]+"/secrets", "application/json", body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 201 {
			return errors.New("Secrets: Unexpected HTTP status code" + resp.Status)
		}
		log.Printf("%s secret created.", requested.Spec.Domain)
		return nil
	}
	return nil
}
