package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"time"
)

var html = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Kubernetes Pod</title>
</head>
<body>
  <h3>Pod Info</h3>
  <ul>
    <li>Hostname: %s</li>
  </ul>
  <h3>Certificate Details</h3>
  <ul>
    <li>Issuer: %s</li>
    <li>Serial: %s</li>
    <li>NotBefore: %s</li>
    <li>NotAfter: %s</li>
  </ul>
</body>
</html>
`

func httpHandler(w http.ResponseWriter, req *http.Request) {
	cert, err := x509.ParseCertificate(cm.certificate.Certificate[0])
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(w, html, hostname, cert.Issuer.CommonName, cert.SerialNumber,
		cert.NotBefore.Format(time.RFC822Z), cert.NotAfter.Format(time.RFC822Z))
}
