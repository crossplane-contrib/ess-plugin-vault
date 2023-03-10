package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/crossplane/crossplane-runtime/pkg/errors"

	"github.com/crossplane-contrib/ess-plugin-vault/pkg/server"
)

func run(log *log.Logger) error {
	port := 4040
	log.Println("main: Initializing GRPC server")
	defer log.Println("main: Completed")

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	serverErrors := make(chan error, 1)

	gRPCServer, err := server.NewServer(port)
	if err != nil {
		return errors.Wrap(err, "running server")
	}

	go func() {
		log.Printf("main: GRPC server listening on port %d", port)
		serverErrors <- gRPCServer.Serve()
	}()

	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Printf("main: %v: Start shutdown", sig)
		gRPCServer.GracefulStop()
	}

	return nil
}

func main() {
	log := log.New(os.Stdout, "GRPC SERVER : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	if err := run(log); err != nil {
		log.Println("main: error:", err)
		os.Exit(1)
	}
}
