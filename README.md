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

### Create a `certificate` object

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
kubectl create -f kubernetes/certificates/hightowerlabs-com.yaml
```

### Create A Google Cloud Service Account Secret

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
2016/07/24 03:56:12 Starting Kubernetes Certificate Controller...
2016/07/24 03:56:12 Kubernetes Certificate Controller started successfully.
2016/07/24 03:56:12 Processing all certificates...
2016/07/24 03:56:12 Get http://127.0.0.1:8001/apis/stable.hightower.com/v1/namespaces/default/certificates: dial tcp 127.0.0.1:8001: getsockopt: connection refused
2016/07/24 03:56:17 Processing certificate: hightowerlabs-dot-com
2016/07/24 03:56:17 Looking up ACME account using: kelsey.hightower@gmail.com
2016/07/24 03:56:17 Renewing certificate for hightowerlabs.com...
2016/07/24 03:56:17 Secret [%s] not found. Creating... hightowerlabs.com
2016/07/24 03:56:17 Watching for certificate changes...
2016/07/24 03:56:18 Processing certificate event for hightowerlabs-dot-com
2016/07/24 03:56:18 Looking up ACME account using: kelsey.hightower@gmail.com
2016/07/24 03:56:18 Renewing certificate for hightowerlabs.com...
2016/07/24 03:56:47 Starting reconciliation loop...
2016/07/24 03:56:47 Processing certificate: hightowerlabs-dot-com
2016/07/24 03:56:47 Looking up ACME account using: kelsey.hightower@gmail.com
2016/07/24 03:56:48 Renewing certificate for hightowerlabs.com...
2016/07/24 03:56:48 Reconciliation loop complete.
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
