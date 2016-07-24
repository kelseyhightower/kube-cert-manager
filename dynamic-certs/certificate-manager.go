package main

import (
	"crypto/tls"
	"log"
	"sync"

	"golang.org/x/exp/inotify"
)

type CertificateManager struct {
	sync.RWMutex
	certFile    string
	keyFile     string
	certificate *tls.Certificate
	Error       chan error
	watcher     *inotify.Watcher
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
		case <-cm.watcher.Event:
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
		case err := <-cm.watcher.Error:
			cm.Error <- err
		}
	}
}

func (cm *CertificateManager) newWatcher() error {
	var err error
	cm.watcher, err = inotify.NewWatcher()
	if err != nil {
		return err
	}
	err = cm.watcher.AddWatch(cm.certFile, inotify.IN_IGNORED)
	if err != nil {
		return err
	}
	return cm.watcher.AddWatch(cm.keyFile, inotify.IN_IGNORED)
}

func (cm *CertificateManager) resetWatcher() error {
	err := cm.watcher.Close()
	if err != nil {
		return err
	}
	return cm.newWatcher()
}
