# Deleting a Certificate

Deleting a certificate object will cause the `kube-cert-manager` to delete the Kubernetes TLS secret holding the Let's Encrypt certificate and private key.

## Delete a Certificate

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

## Recreating a Certificate

Submitting a previously deleted Certificate configuration to the Kubernetes API server will cause the `kube-cert-manager` to reuse the existing Let's Encrypt account associated with the email address defined for the certificate. If a valid Let's Encrypt issued certificate is available it will be downloaded and used when recreating the Kubernetes TLS secret.

```
kubectl create -f kubernetes/certificates/hightowerlabs-com.yaml
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