# Certificate Third Party Resource

## Create the Certificate Third Party Resource

Save the following contents to `certificate.yaml`:

```
apiVersion: extensions/v1beta1
kind: ThirdPartyResource
description: "A specification of a Let's Encrypt Certificate to manage."
metadata:
  name: "certificate.stable.hightower.com"
versions:
  - name: v1
```

Submit the Third Party Resource configuration to the Kubernetes API server:

```
kubectl create -f certificate.yaml 
```

At this point you can now create [Certificate Objects](certificate-objects.md).
