# ess-plugin-vault

> **⚠️ DEPRECATION WARNING**  
> External Secret Store (ESS) support has been removed in Crossplane v2. This plugin will no longer be compatible with Crossplane v2 and later versions. For more information, see [this comment](https://github.com/crossplane/crossplane/issues/5522#issuecomment-2948012043).


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

Below is an example of configuring the plugin to use the AppRole authentication:

```yaml
apiVersion: secrets.crossplane.io/v1alpha1
kind: VaultConfig
metadata:
  name: vault-approle
spec:
  server: <server_address>
  mountPath: <mount_path>
  namespace: <namespace>
  version: v2
  auth:
    method: AppRole
    app_role:
      source: Secret
      secretRef:
        namespace: <namespace>
        name: <secret_name>
        key: <key_name>
```

where the secret referenced in `secretRef` looks like below:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <secret_name>
  namespace: <namespace>
type: Opaque
stringData:
  credentials: |
    {
    "role_id": "roleid",
    "secret_id": "secretid",
    ## specify if the mountPath for approle auth is different from "approle"
    "mountPath": ""
    }
```

Note: Above examples shows setting `app_role` source to `Secret` but other sources like 
`Environment`, `InjectedIdentity` and `Filesystem` can also be used.

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
