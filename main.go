package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	"github.com/NYTimes/gcs-helper/v3/handlers"
	"github.com/google/gops/agent"
	"google.golang.org/api/option"
)

var version = "SNAPSHOT"

const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 120 * time.Second
	// shutdownTimeout stays under Kubernetes' default 30s termination grace
	// period.
	shutdownTimeout = 25 * time.Second
)

func main() {
	handleFlags()
	// gops is only for diagnostics, so don't refuse to serve without it.
	err := agent.Listen(agent.Options{})
	if err != nil {
		log.Printf("could not start gops agent: %v", err)
	} else {
		defer agent.Close()
	}
	config, err := handlers.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	logger := config.Logger()
	hc, err := config.Client.HTTPClient()
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize http client")
	}
	client, err := storage.NewClient(context.Background(), option.WithHTTPClient(hc))
	if err != nil {
		logger.WithError(err).Fatal("failed to create storage client instance")
	}
	handler := getHandler(config, client, hc)
	listener, err := net.Listen("tcp", config.Listen)
	if err != nil {
		logger.WithField("listenAddr", config.Listen).WithError(err).Fatal("failed to start listener")
	}

	// No WriteTimeout: proxied downloads are limited by GCS_HELPER_PROXY_TIMEOUT.
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		logger.Infof("Listening on %s...", listener.Addr())
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("failed to start server")
		}
	}()

	<-ctx.Done()
	stop()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		logger.WithError(err).Error("failed to finish in-flight requests")
	}
}

func handleFlags() {
	printVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *printVersion {
		fmt.Printf("gcs-proxy %s\n", version)
		os.Exit(0)
	}
}
