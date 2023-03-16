# ess-plugin-vault

Crossplane External Secret Store plugin for Hashicorp Vault.

## Developing locally

Start a local development environment with Kind with the plugin installed:

```
make build local-dev
```

Follow this guide to get a local Vault instance running: https://docs.crossplane.io/v1.9/guides/vault-as-secret-store

Create the following manifests to configure Crossplane, Provider GCP and the Plugin:

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