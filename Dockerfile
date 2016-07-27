FROM scratch
ADD kube-cert-manager /kube-cert-manager

# The Kubernetes Certificiate Manager does not support any
# DNS providers out of the box. Each dns provider plugin
# must be saved to the root directory named after the dns
# provider.
# See https://github.com/kelseyhightower/dns01-exec-plugins
ADD googledns /googledns

ENTRYPOINT ["/kube-cert-manager"]
