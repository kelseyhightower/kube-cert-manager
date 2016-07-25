package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"path"

	"github.com/boltdb/bolt"
)

var (
	dataDir      = "/var/lib/cert-manager"
	discoveryURL = "https://acme-staging.api.letsencrypt.org/directory"
	syncInterval = 120
)

func main() {
	flag.StringVar(&dataDir, "data-dir", dataDir, "Data directory path.")
	flag.StringVar(&discoveryURL, "amce-url", discoveryURL, "AMCE endpoint URL.")
	flag.IntVar(&syncInterval, "sync-interval", syncInterval, "Sync interval in seconds.")
	flag.Parse()

	log.Println("Starting Kubernetes Certificate Controller...")

	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	db, err := bolt.Open(path.Join(dataDir, "data.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("Accounts"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Kubernetes Certificate Controller started successfully.")

	// Process all Certificates definitions during the startup process.
	err = syncCertificates(db)
	if err != nil {
		log.Println(err)
	}

	// Watch for events that add, modify, or delete Certificate definitions and
	// process them asynchronously.
	log.Println("Watching for certificate events.")
	watchErrs := watchCertificateEvents(db)

	// Start the certificate reconciler that will ensure all Certificate
	// definitions are backed by a LetsEncrypt certificate and a Kubernetes
	// TLS secret.
	log.Println("Starting reconciliation loop.")
	reconcileErrs := reconcileCertificates(syncInterval, db)

	for {
		select {
		case err := <-watchErrs:
			log.Println(err)
		case err := <-reconcileErrs:
			log.Println(err)
		}
	}
}
