# Kubernetes Certificate Manager

Status: Almost working prototype

This is not an official Google Project.

`kube-cert-manager` is currently a prototype with the following features:

* Manage Lets Encrypt certificates based on a ThirdParty `certificate` resource.
* Will only ever support the dns-01 challenge for Google Cloud DNS. (For now)
* Saves Lets Encrypt certificates as Kubernetes secrets.

This repository will also include a end-to-end tutorial on how to dynamically load TLS certificates.

## Requirements

The `kube-cert-manager` requires a [Google Cloud DNS](https://cloud.google.com/dns) account and a [service account](https://cloud.google.com/storage/docs/authentication#service_accounts) JSON file.

## Usage

### Add the Certificate ThirdParty resource

```
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "A specification of a Let's Encrypt Certificate to manage."
metadata:
  name: "certificate.stable.hightower.com"
versions:
  - name: v1
```

```
kubectl create -f extensions/certificate.yaml 
```

### Run kube-cert-manager

```
gcloud compute disks create kube-cert-manager
```

```
kubectl create -f deployments/kube-cert-manager.yaml 
deployment "kube-cert-manager" created
```

```
kubectl logs kube-cert-manager-2924908400-ua73z kube-cert-manager -f
```

```
2016/07/24 14:31:04 Starting Kubernetes Certificate Controller...
2016/07/24 14:31:04 Kubernetes Certificate Controller started successfully.
2016/07/24 14:31:04 Processing all certificates...
2016/07/24 14:31:09 Watching for certificate changes...
2016/07/24 14:31:09 Starting reconciliation loop...
```

### Create a Certificate

#### Create A Google Cloud Service Account Secret

```
kubectl create secret generic hightowerlabs \
  --from-file=/Users/khightower/Desktop/service-account.json
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

#### Create a Certificate Object

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

```
kubectl create -f certificates/hightowerlabs-com.yaml
```

```
certificate "hightowerlabs-dot-com" created
```

After submitting the certificate configuration to the Kubernetes API it will be processed by the `kube-cert-manager`:

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
