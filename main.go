package main

import (
	"flag"
	"fmt"
	"log"
	"path"
	"time"

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
	log.Println("Processing all certificates...")
	var certificates []Certificate
	for {
		certificates, err = getCertificates()
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	for _, cert := range certificates {
		log.Printf("Processing certificate: %s", cert.Metadata["name"])
		err := processCertificate(cert, db)
		if err != nil {
			log.Println(err)
			continue
		}
	}

	// Watch for events that add, modify, or delete Certificate definitions and
	// process them asynchronously.
	log.Println("Watching for certificate events...")
	events, errs := watchCertificateEvents()

	// Start the certificate reconciler that will ensure all Certificate
	// definitions are backed by a LetsEncrypt certificate and a Kubernetes
	// TLS secret.
	log.Println("Starting reconciliation loop...")
	syncErrs := syncCertificates(syncInterval, db)

	for {
		select {
		case event := <-events:
			log.Printf("Processing certificate event: %s", event.Object.Metadata["name"])
			err := processCertificateEvent(event, db)
			if err != nil {
				log.Println(err)
			}
		case err := <-errs:
			log.Println(err)
		case err := <-syncErrs:
			log.Println(err)
		}
	}
}
