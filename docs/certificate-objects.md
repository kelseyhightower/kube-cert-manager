# Certificate Objects

Certificate objects are used to declare one or more Let's Encrypt issued TLS certificates. Cetificate objects are consumed by the [Kubernetes Certificate Manager](https://github.com/kelseyhightower/kube-cert-manager).

Before you can create a Certificate object you must create the [Certificate Third Party Resource](certificate-third-party-resource.md) in your Kubernetes cluster.

## Required Fields

* apiVersion - The Kubernetes API version. See Certificate Third Party Resource.
* kind - The Kubernetes object type.
* metadata.name - The name of the Certificate object.
* spec.domain - The DNS domain to obtain a Let's Encrypt certificate for.
* spec.email - The email address used for a Let's Encrypt registration.
* spec.provider - The name of the dns provider plugin. See https://github.com/kelseyhightower/dns01-exec-plugins
* spec.secret - The Kubernetes secret that holds dns provider configuration.
* spec.secretKey - The Kubernetes secret key that holds the dns provider configuration data.

### Example

The following Kubernetes Certificate configuration assume the following:

* The `googledns` dns01 exec plugin is available from the `kube-cert-manager` container.
* The `hightowerlabs.com` domain is registered.
* The `hightowerlabs.com` domain is managed by [Google Cloud DNS](https://cloud.google.com/dns) under the `hightowerlabs` Google Cloud project.
* A Kubernetes Secret named `hightowerlabs` exists with a key named `service-account.json` which holds a Google service account with permissions to manage DNS records for the `hightowerlabs.com` domain.

Example Certificate Object

```
apiVersion: "stable.hightower.com/v1"
kind: "Certificate"
metadata:
  name: "hightowerlabs-dot-com"
spec:
  domain: "hightowerlabs.com"
  email: "kelsey.hightower@gmail.com"
  provider: "googledns"
  secret: "hightowerlabs"
  secretKey: "service-account.json"
```
