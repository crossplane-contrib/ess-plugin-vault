# ess-plugin-vault

Crossplane External Secret Store plugin for Hashicorp Vault.

## Installation

Having a Crossplane installation where External Secret Stores alpha feature enabled, install the plugin with:

```
CROSSPLANE_NAMESPACE=crossplane-system
helm upgrade --install ess-plugin-vault oci://xpkg.upbound.io/crossplane-contrib/ess-plugin-vault --namespace $CROSSPLANE_NAMESPACE
```

## Configuration

Create a `VaultConfig` resource to configure the plugin with the Vault server
address, authentication method and token. You would then reference this config
in the `StoreConfig` resources for Crossplane and Providers.

See the following example which configures the plugin to connect to a local
Vault instance running in the `vault-system` namespace with a token injected to
`/vault/secrets/token` by the Vault Agent Injector:

```yaml
apiVersion: secrets.crossplane.io/v1alpha1
kind: VaultConfig
metadata:
  name: local
spec:
  server: http://vault.vault-system:8200
  mountPath: secret/
  version: v2
  auth:
    method: Token
    token:
      source: Filesystem
      fs:
        path: /vault/secrets/token
```

And then reference this config in the `StoreConfig` resources for Crossplane and
Provider GCP:

```yaml
apiVersion: secrets.crossplane.io/v1alpha1
kind: StoreConfig
metadata:
  name: vault
spec:
  type: Plugin
  defaultScope: crossplane-system
  plugin:
    endpoint: ess-plugin-vault.crossplane-system:4040
    configRef:
      apiVersion: secrets.crossplane.io/v1alpha1
      kind: VaultConfig
      name: local
```

```yaml
apiVersion: gcp.crossplane.io/v1alpha1
kind: StoreConfig
metadata:
  name: vault
spec:
  type: Plugin
  defaultScope: crossplane-system
  plugin:
    endpoint: ess-plugin-vault.crossplane-system:4040
    configRef:
      apiVersion: secrets.crossplane.io/v1alpha1
      kind: VaultConfig
      name: local
```

## Developing locally

Start a local development environment with Kind with the plugin installed:

```
make build local-dev
```

Follow this guide to get a local Vault instance running: https://docs.crossplane.io/v1.9/guides/vault-as-secret-store
