package plugin

import (
	"context"
	"net"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	ess "github.com/crossplane/crossplane-runtime/apis/proto/v1alpha1"
	constore "github.com/crossplane/crossplane-runtime/pkg/connection/store"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane-contrib/ess-plugin-vault/apis/config/v1alpha1"
	vault "github.com/crossplane-contrib/ess-plugin-vault/pkg/vault"
)

const (
	errGetConfig  = "could not get config"
	errVaultStore = "could not create new Vault Store"
)

// Server defines the available operations for gRPC ESSVault.
type Server interface {
	// Serve is called for serving requests.
	Serve() error
	// GracefulStop is called for stopping the ESSVault.
	GracefulStop()
	// GetSecret returns the secret.
	GetSecret(ctx context.Context, in *ess.GetSecretRequest) (*ess.GetSecretResponse, error)
}

// ESSVault implements Server.
type ESSVault struct {
	listener   net.Listener
	grpcServer *grpc.Server
	kube       client.Client
	logger     logging.Logger

	ess.UnimplementedExternalSecretStorePluginServiceServer
}

type ESSVaultOption func(*ESSVault)

func WithLogger(logger logging.Logger) ESSVaultOption {
	return func(e *ESSVault) {
		e.logger = logger
	}
}

// NewESSVault creates a new gRPC ESSVault and registers it.
func NewESSVault(kube client.Client, listener net.Listener, gs *grpc.Server, opts ...ESSVaultOption) (*ESSVault, error) {
	s := &ESSVault{
		listener:   listener,
		kube:       kube,
		grpcServer: gs,

		logger: logging.NewNopLogger(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *ESSVault) Serve() error {
	return s.grpcServer.Serve(s.listener)
}

func (s *ESSVault) GracefulStop() {
	s.grpcServer.GracefulStop()
}

func getConfig(ctx context.Context, kube client.Client, ref *ess.ConfigReference) (*v1alpha1.VaultConfig, error) {
	if ref == nil {
		return nil, errors.New("config reference is nil")
	}

	sc := &v1alpha1.VaultConfig{}
	if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name}, sc); err != nil {
		return nil, errors.Wrap(err, "could not get config reference")
	}

	return sc, nil
}

func (s *ESSVault) GetSecret(ctx context.Context, in *ess.GetSecretRequest) (*ess.GetSecretResponse, error) {
	s.logger.Debug("Getting secret", "name", in.Secret.ScopedName)

	secret := new(constore.Secret)
	sn := new(constore.ScopedName)
	sn.Name = in.Secret.ScopedName

	cfg, err := getConfig(ctx, s.kube, in.Config)
	if err != nil {
		return nil, errors.Wrap(err, errGetConfig)
	}

	store, err := vault.NewVaultStore(ctx, s.kube, cfg)
	if err != nil {
		return nil, errors.Wrap(err, errVaultStore)
	}

	err = store.ReadKeyValues(ctx, *sn, secret)
	if err != nil {
		return nil, errors.Wrap(err, "could not read key values")
	}

	essSecret := new(ess.Secret)
	essSecret.Data = make(map[string][]byte, len(secret.Data))
	for k, v := range secret.Data {
		essSecret.Data[k] = v
	}

	if secret.Metadata != nil && len(secret.Metadata.Labels) != 0 {
		essSecret.Metadata = make(map[string]string, len(secret.Metadata.Labels))
		for k, v := range secret.Metadata.Labels {
			essSecret.Metadata[k] = v
		}
	}

	essSecret.ScopedName = in.Secret.ScopedName

	resp := new(ess.GetSecretResponse)
	resp.Secret = essSecret

	return resp, nil
}

func (s *ESSVault) ApplySecret(ctx context.Context, in *ess.ApplySecretRequest) (*ess.ApplySecretResponse, error) {
	s.logger.Debug("Applying secret", "name", in.Secret.ScopedName)

	secret := new(constore.Secret)
	if in.Secret != nil && len(in.Secret.Data) != 0 {
		secret.Data = make(map[string][]byte, len(in.Secret.Data))
		for k, v := range in.Secret.Data {
			secret.Data[k] = v
		}
	}

	if in.Secret != nil && len(in.Secret.Metadata) != 0 {
		secret.Metadata = new(v1.ConnectionSecretMetadata)
		secret.Metadata.Labels = make(map[string]string, len(in.Secret.Metadata))
		for k, v := range in.Secret.Metadata {
			secret.Metadata.Labels[k] = v
		}
	}

	secret.ScopedName.Name = in.Secret.ScopedName

	cfg, err := getConfig(ctx, s.kube, in.Config)
	if err != nil {
		return nil, errors.Wrap(err, errGetConfig)
	}

	store, err := vault.NewVaultStore(ctx, s.kube, cfg)
	if err != nil {
		return nil, errors.Wrap(err, errVaultStore)
	}

	isChanged, err := store.WriteKeyValues(ctx, secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write key values")
	}

	resp := new(ess.ApplySecretResponse)
	resp.Changed = isChanged

	return resp, nil
}

func (s *ESSVault) DeleteKeys(ctx context.Context, in *ess.DeleteKeysRequest) (*ess.DeleteKeysResponse, error) {
	s.logger.Debug("Deleting keys from secret", "name", in.Secret.ScopedName)

	cfg, err := getConfig(ctx, s.kube, in.Config)
	if err != nil {
		return nil, errors.Wrap(err, errGetConfig)
	}

	store, err := vault.NewVaultStore(ctx, s.kube, cfg)
	if err != nil {
		return nil, errors.Wrap(err, errVaultStore)
	}

	secret := new(constore.Secret)
	secret.ScopedName.Name = in.Secret.ScopedName

	return &ess.DeleteKeysResponse{}, store.DeleteKeyValues(ctx, secret)
}
