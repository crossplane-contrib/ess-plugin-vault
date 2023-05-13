package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/certificates"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane-contrib/ess-plugin-vault/apis/config/v1alpha1"
	"github.com/crossplane-contrib/ess-plugin-vault/pkg/plugin"
	proto "github.com/crossplane/crossplane-runtime/apis/proto/v1alpha1"
)

var cli struct {
	// Debug is the flag to run the plugin in debug mode.
	Debug bool `help:"Run the plugin in debug mode."`
	// Port is the port number that the plugin will listen on.
	Port int `default:"4040" help:"Port number that the plugin will listen on."`
	// CertsPath is the path to the directory where the certificates are stored.
	CertsPath string `default:"/certs" help:"Path to directory where the certificates are stored."`
}

func main() {
	ctx := kong.Parse(&cli)
	zl := zap.New(zap.UseDevMode(cli.Debug))
	logger := logging.NewLogrLogger(zl.WithName("ess-plugin-vault"))
	logger.Info("Starting Crossplane External Secrets Store Plugin for Vault")

	s := runtime.NewScheme()
	err := v1alpha1.AddToScheme(s)
	ctx.FatalIfErrorf(err, "cannot add apis to scheme")

	err = corev1.AddToScheme(s)
	ctx.FatalIfErrorf(err, "cannot add coreapis to scheme")

	cfg, err := ctrl.GetConfig()
	ctx.FatalIfErrorf(errors.Wrap(err, "cannot get config"))

	kube, err := client.New(cfg, client.Options{Scheme: s})
	ctx.FatalIfErrorf(err, "cannot create client")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cli.Port))
	ctx.FatalIfErrorf(err, "cannot listen on port %d", cli.Port)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	serverErrors := make(chan error, 1)

	tlsConfig, err := certificates.LoadMTLSConfig(filepath.Join(cli.CertsPath, "ca.crt"), filepath.Join(cli.CertsPath, "tls.crt"), filepath.Join(cli.CertsPath, "tls.key"), true)
	ctx.FatalIfErrorf(err, "cannot load certificates")

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	reflection.Register(grpcServer)

	essServer, err := plugin.NewESSVault(kube, listener, grpcServer, plugin.WithLogger(logger))
	ctx.FatalIfErrorf(err, "cannot create server")

	proto.RegisterExternalSecretStorePluginServiceServer(grpcServer, essServer)

	go func() {
		logger.Info("GRPC server listening on port", "port", cli.Port)
		serverErrors <- essServer.Serve()
	}()

	select {
	case err := <-serverErrors:
		ctx.FatalIfErrorf(err, "cannot start server")
	case <-shutdown:
		logger.Info("Shutting down the External Secrets Store Vault Plugin")
		essServer.GracefulStop()
	}
}
