# Stackryze cert-manager Webhook (DNS-01)

An [ACME DNS-01 webhook solver](https://cert-manager.io/docs/configuration/acme/dns01/webhook/)
for [cert-manager](https://cert-manager.io) that issues certificates (including
**wildcards**) by creating `_acme-challenge` TXT records on
[Stackryze DNS](https://dns.stackryze.com). It uses the Stackryze REST API with a
Bearer token — no impact on the DNS serving path.

```
cert-manager  ──(DNS-01 challenge)──►  stackryze-webhook  ──(Bearer token)──►  api.stackryze.com
                                              │
                                      creates/deletes TXT _acme-challenge
```

## Prerequisites

- Kubernetes with [cert-manager](https://cert-manager.io/docs/installation/) installed.
- A Stackryze API token with **write** scope (Settings → API tokens).
- Your zone(s) already created on Stackryze.

## Install

```bash
# 1. Create the API token secret in the cert-manager namespace
kubectl -n cert-manager create secret generic stackryze-api-token \
  --from-literal=token=sk_dns_xxxxxxxx

# 2. Install the webhook (into the cert-manager namespace)
helm install stackryze-webhook ./deploy/stackryze-webhook \
  --namespace cert-manager \
  --set groupName=acme.stackryze.com \
  --set image.repository=ghcr.io/stackryze/stackryze-cert-manager-webhook \
  --set image.tag=latest
```

`groupName` must be identical in the chart, the built image (`GROUP_NAME`), and
your Issuer solver config.

## Configure an Issuer

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-stackryze
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: you@example.com
    privateKeySecretRef:
      name: letsencrypt-stackryze-account
    solvers:
      - dns01:
          webhook:
            groupName: acme.stackryze.com
            solverName: stackryze
            config:
              apiUrl: https://api.stackryze.com/api
              apiTokenSecretRef:
                name: stackryze-api-token
                key: token
```

Then request a wildcard certificate — see [examples/clusterissuer.yaml](examples/clusterissuer.yaml).

## Config reference

| Field | Required | Description |
|-------|----------|-------------|
| `apiUrl` | no | API base (default `https://api.stackryze.com/api`) |
| `apiTokenSecretRef.name` | yes | Secret holding the API token |
| `apiTokenSecretRef.key` | yes | Key in the secret (e.g. `token`) |

The token secret must live in the same namespace as the Issuer (for a
`ClusterIssuer`, that's the cert-manager namespace).

## Build locally

Requires Go 1.23+.

```bash
go mod tidy
go build -o stackryze-cert-manager-webhook .

# Docker
docker build -t ghcr.io/stackryze/stackryze-cert-manager-webhook:latest .
```

## How it works

- `Present` finds the zone for the challenge, then creates a TXT record at
  `_acme-challenge…` with the ACME key (TTL 3600s). Repeated presents are
  idempotent.
- `CleanUp` deletes that TXT record after validation.
