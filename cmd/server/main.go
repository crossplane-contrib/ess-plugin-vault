package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane-contrib/ess-plugin-vault/pkg/server"
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
	logger.Info("Starting the External Secrets Store Vault Plugin")

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	serverErrors := make(chan error, 1)

	gRPCServer, err := server.NewServer(cli.Port, cli.CertsPath)
	ctx.FatalIfErrorf(err, "cannot create server")

	go func() {
		logger.Info("GRPC server listening on port", "port", cli.Port)
		serverErrors <- gRPCServer.Serve()
	}()

	select {
	case err := <-serverErrors:
		ctx.FatalIfErrorf(err, "cannot start server")
	case <-shutdown:
		logger.Info("Shutting down the External Secrets Store Vault Plugin")
		gRPCServer.GracefulStop()
	}
}
