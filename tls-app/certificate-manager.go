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
	"crypto/tls"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type CertificateManager struct {
	sync.RWMutex
	certFile    string
	keyFile     string
	certificate *tls.Certificate
	Error       chan error
	watcher     *fsnotify.Watcher
}

func NewCertificateManager(certFile, keyFile string) (*CertificateManager, error) {
	cm := &CertificateManager{
		certFile: certFile,
		keyFile:  keyFile,
		Error:    make(chan error, 10),
	}
	err := cm.setCertificate()
	if err != nil {
		return nil, err
	}

	go cm.watchCertificate()

	return cm, nil
}

func (cm *CertificateManager) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.RLock()
	defer cm.RUnlock()
	return cm.certificate, nil
}

func (cm *CertificateManager) setCertificate() error {
	log.Println("Loading TLS certificates...")
	c, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
	if err != nil {
		return err
	}
	cm.Lock()
	cm.certificate = &c
	cm.Unlock()
	return nil
}

func (cm *CertificateManager) watchCertificate() error {
	log.Println("Watching for TLS certificate changes...")
	err := cm.newWatcher()
	if err != nil {
		return err
	}

	for {
		select {
		case <-cm.watcher.Events:
			log.Println("Reloading TLS certificates...")
			err := cm.setCertificate()
			if err != nil {
				cm.Error <- err
			}
			log.Println("Reloading TLS certificates complete.")
			err = cm.resetWatcher()
			if err != nil {
				cm.Error <- err
			}
		case err := <-cm.watcher.Errors:
			cm.Error <- err
		}
	}
}

func (cm *CertificateManager) newWatcher() error {
	var err error
	cm.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = cm.watcher.Add(cm.certFile)
	if err != nil {
		return err
	}
	return cm.watcher.Add(cm.keyFile)
}

func (cm *CertificateManager) resetWatcher() error {
	err := cm.watcher.Close()
	if err != nil {
		return err
	}
	return cm.newWatcher()
}
