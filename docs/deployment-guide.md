# Deployment Guide

This guide will walk you through deploying the Kubernetes Certificate Manager.

By default `kube-cert-manager` obtains certificates from the Let's Encrypt staging environment. Set the `-acme-url` flag to `https://acme-v01.api.letsencrypt.org/directory` for production.

## High Level Tasks

* Create the Certificate Third Party Resource
* Create the Kubernetes Certificate Manager Deployment

## Deploying the Kubernetes Certificate Manager

### Create the Certificate Third Party Resource

The `kube-cert-manager` is driven by [Kubernetes Certificate Objects](certificate-objects.md). Certificates are not a core Kubernetes kind, but can be enabled with the [Certificate Third Party Resource](certificate-third-party-resource.md):

Create the Certificate Third Party Resource:

```
kubectl create -f extensions/certificate.yaml 
```

### Create the Kubernetes Certificate Manager Deployment

The `kube-cert-manager` requires persistent storage to hold the following data:

* Let's Encrypt user accounts, private keys, and registrations
* Let's Encrypt issued certificates

Create a persistent disk which will store the `kube-cert-manager` database.
> [boltdb](https://github.com/boltdb/bolt) is used to persistent data.

```
gcloud compute disks create kube-cert-manager --size 10GB
```

> 10GB is the minimal disk size for a Google Compute Engine persistent disk.

The `kube-cert-manager` requires access to the Kubernetes API to perform the following tasks:

* Read secrets that hold Google cloud service accounts.
* Create, update, and delete Kubernetes TLS secrets backed by Let's Encrypt Issued certificates.

The `kube-cert-manager` leverages `kubectl` running in proxy mode for API access and both containers should be deployed in the same pod.

Create the `kube-cert-manager` deployment:

```
kubectl create -f deployments/kube-cert-manager.yaml 
```
```
deployment "kube-cert-manager" created
```

Review the `kube-cert-manager` logs:

```
kubectl get pods
```
```
NAME                                 READY     STATUS    RESTARTS   AGE
kube-cert-manager-1999323568-op6nk   2/2       Running   0          25s
```

```
kubectl logs kube-cert-manager-1999323568-op6nk kube-cert-manager
```

```
2016/07/25 06:33:21 Starting Kubernetes Certificate Controller...
2016/07/25 06:33:22 Kubernetes Certificate Controller started successfully.
2016/07/25 06:33:27 Watching for certificate events.
2016/07/25 06:33:27 Starting reconciliation loop.
```
