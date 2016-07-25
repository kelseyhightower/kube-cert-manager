# Consuming Certificates

Once you have the Kubernetes Certificate Manager up and running [create one or more certificates](create-a-certificate.md), which will give you a set of Kubernetes TLS secrets that you can consume in your applications.

This tutorial will walk you through creating a Pod manifest that consumes the certificates created by the Kubernetes Certificate Manager.

## Create an Application

First you'll need an application that serves up HTTPS traffic to clients. The application should have the following features:

* Support loading TLS certificates from a filesystem.
* Support reloading certificates at runtime.

The [dynamic-certs](https://github.com/kelseyhightower/kube-cert-manager/tree/master/dynamic-certs) example application meets the above requirements and will be used for this tutorial. The dynamic-certs application leverages [inotify](http://man7.org/linux/man-pages/man7/inotify.7.html) to monitor a filesystem for TLS certificate changes and reloads them without requiring a restart.

## Create a Deployment

The complete `dynamic-certs` deployment config can be found [here](https://github.com/kelseyhightower/kube-cert-manager/blob/master/dynamic-certs/deployments/dynamic-certs.yaml), for now lets focus on the important parts.

```
spec:
  containers:
  - name: dynamic-certs
    image: kelseyhightower/dynamic-certs:1.0.0
    args:
      - "-tls-cert=/etc/tls/tls.crt"
      - "-tls-key=/etc/tls/tls.key"
    volumeMounts:
      - name: dynamic-certs-tls
        mountPath: /etc/tls
  volumes:
    - name: dynamic-certs-tls
      secret:
        secretName: hightowerlabs.com
```

The key to consuming Kubernetes TLS secrets is to use a secret volume. Study the snippet above and notice how the `hightowerlabs.com` secret is being mounted under the `/etc/tls` directory. By default the Kubernetes Certificate Manager will store all certificates and privates key using the `tls.crt` and `tls.key` key names. This will result in two files under the `/etc/tls` directory at runtime.

Use kubectl to create the `dynamic-certs` deployment:

```
kubectl create -f dynamic-certs/deployments/dynamic-certs.yaml
```

```
deployment "dynamic-certs" created
```

Review the logs:

```
kubectl logs dynamic-certs-1623907102-wg95k
```
```
2016/07/25 14:15:53 Initializing application...
2016/07/25 14:15:53 Loading TLS certificates...
2016/07/25 14:15:53 HTTPS listener on :443...
2016/07/25 14:15:53 Watching for TLS certificate changes...
```

### Getting a New Certificate

An easy way to force the Kubernetes Certificate Manager to generate a new Let's Encrypt issued certificate is to delete the `hightowerlabs-dot-com` certificate object:

```
kubectl delete certificates hightowerlabs-dot-com
```
```
certificate "hightowerlabs-dot-com" deleted
```

```
kubectl logs kube-cert-manager-1999323568-npjf5 kube-cert-manager -f
```

```
2016/07/25 14:17:33 Deleting Let's Encrypt account: hightowerlabs.com
2016/07/25 14:17:33 Deleting Kubernetes TLS secret: hightowerlabs.com
```

Now recreate the hightowerlabs-dot-com certificate:

```
kubectl create -f kubernetes/certificates/hightowerlabs-com.yaml
```
``` 
certificate "hightowerlabs-dot-com" created
```

This will cause the `kube-cert-manager` to create a new Let's Encrypt user account and aquire a new certificate.

```
kubectl logs kube-cert-manager-1999323568-npjf5 kube-cert-manager -f
```

```
2016/07/25 14:19:35 Creating new Let's Encrypt account: hightowerlabs.com
2016/07/25 14:19:38 Monitoring _acme-challenge.hightowerlabs.com. DNS propagation: ns-cloud-c1.googledomains.com.:53 ns-cloud-c2.googledomains.com.:53 ns-cloud-c3.googledomains.com.:53 ns-cloud-c4.googledomains.com.:53
2016/07/25 14:19:39 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c4.googledomains.com.:53
2016/07/25 14:19:43 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c1.googledomains.com.:53
2016/07/25 14:19:46 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c3.googledomains.com.:53
2016/07/25 14:20:10 hightowerlabs.com DNS-01 challenge complete on ns-cloud-c2.googledomains.com.:53
2016/07/25 14:20:40 _acme-challenge.hightowerlabs.com. DNS propagation complete.
2016/07/25 14:20:44 hightowerlabs.com secret missing.
2016/07/25 14:20:45 hightowerlabs.com secret created.
```

After a few minutes the `dynamic-certs` application will pickup and reload the new TLS certificates.

```
kubectl logs dynamic-certs-1623907102-wg95k -f
```

```
2016/07/25 14:15:53 Initializing application...
2016/07/25 14:15:53 Loading TLS certificates...
2016/07/25 14:15:53 HTTPS listener on :443...
2016/07/25 14:15:53 Watching for TLS certificate changes...
2016/07/25 14:22:30 Reloading TLS certificates...
2016/07/25 14:22:30 Loading TLS certificates...
2016/07/25 14:22:30 Reloading TLS certificates complete.
```