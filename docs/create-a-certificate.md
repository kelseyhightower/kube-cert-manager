# Creating a Certificate

Let's Encrypt issued certificates are automatically created for each Kubernetes Certificate object.

## Create A Google Cloud Service Account Secret

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

## Create a Kubernetes Certificate Object

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

## Results

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