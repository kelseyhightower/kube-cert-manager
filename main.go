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
)

func main() {
	flag.StringVar(&dataDir, "data-dir", dataDir, "Data directory path")
	flag.StringVar(&discoveryURL, "amce-url", discoveryURL, "AMCE endpoint")
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

	// Process all certificates.
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

	// Watch for certificate events.
	log.Println("Watching for certificate changes...")
	events, errs := watchCertificateEvents()
	syncErrs := syncCertificates(db)
	for {
		select {
		case event := <-events:
			log.Printf("Processing certificate event for %s", event.Object.Metadata["name"])
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
