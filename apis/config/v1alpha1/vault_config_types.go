/*
Copyright 2023 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// +kubebuilder:object:root=true

// VaultConfig is the CRD type for External Vault Config.
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,categories={crossplane,pkg}
type VaultConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec *VaultConfigSpec `json:"spec,omitempty"`
}

type VaultConfigSpec struct {
	// Server is the url of the Vault server, e.g. "https://vault.acme.org"
	Server string `json:"server"`

	// MountPath is the mount path of the KV secrets engine.
	MountPath string `json:"mountPath"`

	// Version of the KV Secrets engine of Vault.
	// https://www.vaultproject.io/docs/secrets/kv
	// +optional
	// +kubebuilder:default=v2
	Version *VaultKVVersion `json:"version,omitempty"`

	// CABundle configures CA bundle for Vault Server.
	// +optional
	CABundle *VaultCABundleConfig `json:"caBundle,omitempty"`

	// Auth configures an authentication method for Vault.
	Auth VaultAuthConfig `json:"auth"`
}

// VaultAuthMethod represent a Vault authentication method.
// https://www.vaultproject.io/docs/auth
type VaultAuthMethod string

const (
	// VaultAuthToken indicates that "Token Auth" will be used to
	// authenticate to Vault.
	// https://www.vaultproject.io/docs/auth/token
	VaultAuthToken VaultAuthMethod = "Token"
)

// VaultAuthTokenConfig represents configuration for Vault Token Auth Method.
// https://www.vaultproject.io/docs/auth/token
type VaultAuthTokenConfig struct {
	// Source of the credentials.
	// +kubebuilder:validation:Enum=None;Secret;Environment;Filesystem
	Source v1.CredentialsSource `json:"source"`

	// CommonCredentialSelectors provides common selectors for extracting
	// credentials.
	v1.CommonCredentialSelectors `json:",inline"`
}

// VaultAuthConfig required to authenticate to a Vault API.
type VaultAuthConfig struct {
	// Method configures which auth method will be used.
	Method VaultAuthMethod `json:"method"`
	// Token configures Token Auth for Vault.
	// +optional
	Token *VaultAuthTokenConfig `json:"token,omitempty"`
}

// VaultCABundleConfig represents configuration for configuring a CA bundle.
type VaultCABundleConfig struct {
	// Source of the credentials.
	// +kubebuilder:validation:Enum=None;Secret;Environment;Filesystem
	Source v1.CredentialsSource `json:"source"`

	// CommonCredentialSelectors provides common selectors for extracting
	// credentials.
	v1.CommonCredentialSelectors `json:",inline"`
}

// VaultKVVersion represent API version of the Vault KV engine
// https://www.vaultproject.io/docs/secrets/kv
type VaultKVVersion string

const (
	// VaultKVVersionV1 indicates that Secret API is KV Secrets Engine Version 1
	// https://www.vaultproject.io/docs/secrets/kv/kv-v1
	VaultKVVersionV1 VaultKVVersion = "v1"

	// VaultKVVersionV2 indicates that Secret API is KV Secrets Engine Version 2
	// https://www.vaultproject.io/docs/secrets/kv/kv-v2
	VaultKVVersionV2 VaultKVVersion = "v2"
)

// +kubebuilder:object:root=true

// VaultConfigList contains a list of VaultConfig
type VaultConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultConfig `json:"items"`
}
