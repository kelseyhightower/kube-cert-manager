# DNS Provider Plugins

The Kubernetes Certificate Manager does not have support for any DNS providers built in. Support for DNS providers is done using [dns-01 exec plugins](https://github.com/kelseyhightower/dns01-exec-plugins). To ease initial deployments the `kelseyhightower/kube-cert-manager` image ships with the `googledns` dns01 exec plugin baked in. See the [Dockerfile](https://github.com/kelseyhightower/kube-cert-manager/blob/master/Dockerfile) for more info.

## Why Exec Based Plugins?

The plugin model was chosen because the API between the Kubernetes Certificate Manager is rather simple. dns-01 exec plugins only need to create or delete a single DNS TXT record.

Exec based plugins also make it easy for people to extend the Kubernetes Certificate Manager without recompiling the `kube-cert-manager` binary. Exec based plugins also let people build plugins in their language of choice. This is a huge win because not everyone uses Go for everything.

## Creating DNS-01 Exec Plugins

See the [DNS-01 exec plugins](https://github.com/kelseyhightower/dns01-exec-plugins) github repo for more details and example implementations.

## Shipping DNS-01 Exec Plugins

The `kube-cert-manager` is [deployed](deployment-guide.md) using a Kubernetes deployment, which requires a container image. By default the `kube-cert-manager` deployment utilizes the following Docker image:

```
kelseyhightower/kube-cert-manager:0.2.0
```

To add additional DNS-01 exec plugins create a Dockerfile that adds each plugin to the root directory of the container image. Example:

```
FROM kelseyhightower/kube-cert-manager:0.2.0
ADD googledns /googledns
ADD cloudflare /cloudflare
ENTRYPOINT ["/kube-cert-manager"]
```

Ideally each plugin should be self-contained and compiled for Linux. For Go programs that means building your binaries like this:

```
cd $PLUGINDIR/cloudflare
```

```
GOOS=linux go build \
  -a --ldflags '-extldflags "-static"' \
  -tags netgo \
  -installsuffix netgo \
  -o cloudflare .
```

See the [googledns](https://github.com/kelseyhightower/dns01-exec-plugins/tree/master/googledns) plugin for a working example.

## Non Go Plugins

dns01 exec plugins can be written in any language, but you must be sure to build a container image with all the necessary runtimes and libraries to make them work.
