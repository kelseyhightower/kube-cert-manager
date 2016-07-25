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
cat kubernetes/certificates/hightowerlabs-com.yaml
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
kubectl create -f kubernetes/certificates/hightowerlabs-com.yaml
```

```
certificate "hightowerlabs-dot-com" created
```

After submitting the Certificate configuration to the Kubernetes API it will be processed by the `kube-cert-manager`:

Logs from the `kube-cert-manager`:

```
2016/07/25 06:37:58 Processing certificate event: hightowerlabs-dot-com
2016/07/25 06:37:58 Creating new Let's Encrypt account: hightowerlabs.com
2016/07/25 06:38:02 Monitoring _acme-challenge.hightowerlabs.com. DNS propagation: ns-cloud-c1.googledomains.com.:53 ns-cloud-c2.googledomains.com.:53 ns-cloud-c3.googledomains.com.:53 ns-cloud-c4.googledomains.com.:53
2016/07/25 06:38:20 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c1.googledomains.com.:53
2016/07/25 06:38:25 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c3.googledomains.com.:53
2016/07/25 06:38:25 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c4.googledomains.com.:53
2016/07/25 06:38:49 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c2.googledomains.com.:53
2016/07/25 06:39:19 _acme-challenge.hightowerlabs.com. DNS propagation complete.
2016/07/25 06:39:22 hightowerlabs.com secret missing.
2016/07/25 06:39:22 hightowerlabs.com secret created.
```

## Results

```
kubectl get secrets hightowerlabs.com
```
```
NAME                TYPE                DATA      AGE
hightowerlabs.com   kubernetes.io/tls   2         1m
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