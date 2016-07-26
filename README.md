# Kubernetes Certificate Manager

This is not an official Google Project.

## Features

* Manage Kubernetes TLS secrets backed by Let's Encrypt issued certificates.
* Manage [Let's Encrypt](https://letsencrypt.org) issued certificates based on Kubernetes ThirdParty Resources.
* Domain validation using ACME [dns-01 challenges](https://letsencrypt.github.io/acme-spec/#rfc.section.7.4).

## Project Goals

* Demonstrate how to build custom Kubernetes controllers.
* Demonstrate how to use Kubernetes [Third Party Resources](https://github.com/kubernetes/kubernetes/blob/release-1.3/docs/design/extending-api.md).
* Demonstrate how to interact with the Kubernetes API (watches, reconciliation, etc).
* Demonstrate how to write great documentation for Kubernetes add-ons and extensions.
* Promote the usage of Let's Encrypt for securing web applications running on Kubernetes.

## Requirements

* A registered DNS domain managed by [Google Cloud DNS](https://cloud.google.com/dns).
* A [Google service account](https://cloud.google.com/storage/docs/authentication#service_accounts) with permissions to manage DNS records for your domains.

## Usage

* [Deployment Guide](docs/deployment-guide.md)
* [Creating a Certificate](docs/create-a-certificate.md)
* [Deleting a Certificate](docs/delete-a-certificate.md)
* [Consuming Certificates](docs/consume-certificates.md)

## Documentation

* [Certificate Third Party Resources](docs/certificate-third-party-resource.md)
* [Certificate Objects](docs/certificate-objects.md)
