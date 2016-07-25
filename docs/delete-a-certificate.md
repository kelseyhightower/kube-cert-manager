# Deleting a Certificate

Deleting a Kubernetes Certificate object will cause the `kube-cert-manager` to delete the following items:

* The Kubernetes TLS secret holding the Let's Encrypt certificate and private key.
* The Let's Encrypt user account registered for the domain.

## Delete a Certificate

```
kubectl delete certificates hightowerlabs-dot-com
```
```
certificate "hightowerlabs-dot-com" deleted
```

Logs from the `kube-cert-manager`:

```
2016/07/25 06:42:03 Processing certificate event: hightowerlabs-dot-com
2016/07/25 06:42:03 Deleting Let's Encrypt account: hightowerlabs.com
2016/07/25 06:42:03 Deleting Kubernetes TLS secret: hightowerlabs.com
```