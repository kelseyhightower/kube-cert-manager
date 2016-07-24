# Kubernetes Certificate Manager

This is not an official Google Project.

## Features

* Manage [Let's Encrypt](https://letsencrypt.org) issued certificates based on Kubernetes ThirdParty Resources.
* Domain validation using ACME [dns-01 challenges](https://letsencrypt.github.io/acme-spec/#rfc.section.7.4).
* Saves Let's Encrypt issued certificates as Kubernetes TLS secrets.

## Project Goals

* Demonstrate how to build custom Kubernetes controllers.
* Demonstrate how to use Kubernetes [Third Party Resources](https://github.com/kubernetes/kubernetes/blob/release-1.3/docs/design/extending-api.md).
* Demonstrate how interact with the Kubernetes API (watches, reconciliation, etc).
* Demonstrate how to write great documentation for Kubernetes add-ons and extensions.
* Promote the usage of Let's Encrypt for securing web application running on Kubernetes.

## Requirements

The `kube-cert-manager` requires a [Google Cloud DNS](https://cloud.google.com/dns) account and a [service account](https://cloud.google.com/storage/docs/authentication#service_accounts) JSON file.

## Usage

### Create the Certificate ThirdParty Resource

The `kube-cert-manager` is driven by Kubernetes Certificate objects. Certificates are not a core Kubernetes kind, but can be defined using the following `ThirdPartyResource`:

```
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "A specification of a Let's Encrypt Certificate to manage."
metadata:
  name: "certificate.stable.hightower.com"
versions:
  - name: v1
```

Create the Certificate Third Party Resource:

```
kubectl create -f extensions/certificate.yaml 
```

### Create the Kubernetes Certificate Manager Deployment

Create a persistent disk which will store the `kube-cert-manager` database.
> [boltdb](https://github.com/boltdb/bolt) is used to persistent data.

```
gcloud compute disks create kube-cert-manager --size 10GB
```

> 10GB is the minimal disk size for a Google Compute Engine persistent disk.

Create the `kube-cert-manager` deployment:

```
kubectl create -f deployments/kube-cert-manager.yaml 
```
```
deployment "kube-cert-manager" created
```

Review the `kube-cert-manager` logs:

```
kubectl logs kube-cert-manager-2924908400-ua73z kube-cert-manager
```

```
2016/07/24 14:31:04 Starting Kubernetes Certificate Controller...
2016/07/24 14:31:04 Kubernetes Certificate Controller started successfully.
2016/07/24 14:31:04 Processing all certificates...
2016/07/24 14:31:09 Watching for certificate changes...
2016/07/24 14:31:09 Starting reconciliation loop...
```

### Create a Certificate

Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object.

#### Create A Google Cloud Service Account Secret

The `kube-cert-manager` requires a service account with access to the Google DNS API. The service account must be stored in a Kubernetes secret that can be retrieved by the `kube-cert-manager` at runtime.

```
kubectl create secret generic hightowerlabs \
  --from-file=service-account.json
```

> The secret key must be named `service-account.json`

```
kubectl describe secret hightowerlabs
```
```
Name:        hightowerlabs
Namespace:   default
Labels:      <none>
Annotations: <none>

Type:        Opaque

Data
====
service-account.json:   3915 bytes
```

#### Create a Kubernetes Certificate Object

```
cat certificates/hightowerlabs-com.yaml
```

```
apiVersion: "stable.hightower.com/v1"
kind: "Certificate"
metadata:
  name: "hightowerlabs-dot-com"
spec:
  domain: "hightowerlabs.com"
  email: "kelsey.hightower@gmail.com"
  project: "hightowerlabs"
  serviceAccount: "hightowerlabs"
```

> The `spec.serviceAccount` value must match the name of a Kubernetes secret that holds a Google service account.

```
kubectl create -f certificates/hightowerlabs-com.yaml
```

```
certificate "hightowerlabs-dot-com" created
```

After submitting the Certificate configuration to the Kubernetes API it will be processed by the `kube-cert-manager`:

Logs from the `kube-cert-manager`:

```
2016/07/24 14:32:12 Processing certificate event for hightowerlabs-dot-com
2016/07/24 14:32:12 ACME account for kelsey.hightower@gmail.com not found. Creating new account.
2016/07/24 14:32:17 matching TXT record found [ns-cloud-c1.googledomains.com:53]
2016/07/24 14:32:40 matching TXT record found [ns-cloud-c2.googledomains.com:53]
2016/07/24 14:32:41 matching TXT record found [ns-cloud-c3.googledomains.com:53]
2016/07/24 14:32:41 matching TXT record found [ns-cloud-c4.googledomains.com:53]
2016/07/24 14:33:15 Secret [hightowerlabs.com] not found. Creating...
```

#### Results

```
kubectl get secrets hightowerlabs.com
```
```
NAME                TYPE                DATA      AGE
hightowerlabs.com   kubernetes.io/tls   2         10m
```

```
kubectl describe secrets hightowerlabs.com
```
```
Name:        hightowerlabs.com
Namespace:   default
Labels:      <none>
Annotations: <none>

Type:        kubernetes.io/tls

Data
====
tls.crt:     1761 bytes
tls.key:     1679 bytes
```

### Deleting a Certificate

Deleting a certificate object will cause the `kube-cert-manager` to delete the Kubernetes TLS secret holding the Let's Encrypt certificate and private key.

```
kubectl delete certificates hightowerlabs-dot-com
```
```
certificate "hightowerlabs-dot-com" deleted
```

Logs from the `kube-cert-manager`:

```
2016/07/24 14:54:26 Processing certificate event for hightowerlabs-dot-com
2016/07/24 14:54:26 Deleting hightowerlabs.com certificate...
```

> Note: The Let's Encrypt user account is not deleted from the `kube-cert-manager` internal database to prevent hitting rate limits on account registrations. User accounts can also be used by multiple certificates so we keep them around even if the account is not in use.

### Recreating a Certificate

Submitting a previously deleted Certificate configuration to the Kubernetes API server will cause the `kube-cert-manager` to reuse the existing Let's Encrypt account associated with the email address defined for the certificate. If a valid Let's Encrypt issued certificate is available it will be downloaded and used when recreating the Kubernetes TLS secret.

```
kubectl create -f certificates/hightowerlabs-com.yaml
```
```
certificate "hightowerlabs-dot-com" created
```

Logs from the `kube-cert-manager`:

```
2016/07/24 14:58:06 Processing certificate event for hightowerlabs-dot-com
2016/07/24 14:58:07 Fetching existing certificate for hightowerlabs.com.
2016/07/24 14:58:07 Secret [hightowerlabs.com] not found. Creating...
```

```
kubectl get secrets
```
```
NAME                  TYPE                                  DATA      AGE
default-token-c5vn8   kubernetes.io/service-account-token   3         1d
hightowerlabs         Opaque                                1         1d
hightowerlabs.com     kubernetes.io/tls                     2         2m
```
