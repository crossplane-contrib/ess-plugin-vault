package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	ess "github.com/crossplane/crossplane-runtime/apis/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/certificates"
	constore "github.com/crossplane/crossplane-runtime/pkg/connection/store"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/crossplane-contrib/ess-plugin-vault/apis/config/v1alpha1"
	vault "github.com/crossplane-contrib/ess-plugin-vault/pkg/vault"
)

var (
	netListen = net.Listen
)

const (
	errGetConfig  = "could not get config"
	errVaultStore = "could not create new Vault Store"
)

// Server defines the available operations for gRPC server.
type Server interface {
	// Serve is called for serving requests.
	Serve() error
	// GracefulStop is called for stopping the server.
	GracefulStop()
	// GetSecret returns the secret.
	GetSecret(ctx context.Context, in *ess.GetSecretRequest) (*ess.GetSecretResponse, error)
}

// server implements Server.
type server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	kube       client.Client
	ess.UnimplementedExternalSecretStoreServiceServer
}

func (s *server) Serve() error {
	return s.grpcServer.Serve(s.listener)
}

func (s *server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

// NewServer creates a new gRPC server and registers it.
func NewServer(port int) (Server, error) {
	sc := runtime.NewScheme()
	err := v1alpha1.AddToScheme(sc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add scheme")
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: sc})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	listener, err := netListen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, errors.Wrap(err, "tcp listening")
	}

	s := new(server)

	s.listener = listener
	s.kube = cl
	tlsConfig, err := certificates.Load("/certs", true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load certificates")
	}

	creds := credentials.NewTLS(tlsConfig)
	s.grpcServer = grpc.NewServer(grpc.Creds(creds))
	//s.grpcServer = grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(middleFunc))

	ess.RegisterExternalSecretStoreServiceServer(s.grpcServer, s)
	reflection.Register(s.grpcServer)

	return s, nil
}

func middleFunc(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// get client tls info
	if p, ok := peer.FromContext(ctx); ok {
		if mTLS, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			for _, item := range mTLS.State.PeerCertificates {
				fmt.Println(item.Issuer)
				log.Println("request certificate subject:", item.Subject)
			}
		}
	}
	return handler(ctx, req)
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

func (s *server) GetSecret(ctx context.Context, in *ess.GetSecretRequest) (*ess.GetSecretResponse, error) {
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

func (s *server) ApplySecret(ctx context.Context, in *ess.ApplySecretRequest) (*ess.ApplySecretResponse, error) {
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

func (s *server) DeleteKeys(ctx context.Context, in *ess.DeleteKeysRequest) (*ess.DeleteKeysResponse, error) {
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
