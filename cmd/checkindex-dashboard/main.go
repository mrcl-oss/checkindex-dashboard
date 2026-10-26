package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrcl-oss/checkindex-dashboard/internal/config"
	"github.com/mrcl-oss/checkindex-dashboard/internal/dashboard"
	"github.com/mrcl-oss/checkindex-dashboard/internal/httpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	service := dashboard.NewService(nil, cfg.RefreshInterval)
	serviceDone := make(chan struct{})
	go func() {
		defer close(serviceDone)
		service.Run(ctx)
	}()

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           dashboard.NewHandlerWithRefreshInterval(service, cfg.RefreshInterval),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := httpserver.Serve(ctx, server); err != nil {
		stop()
		<-serviceDone
		log.Printf("Dashboard server stopped unexpectedly: %v", err)
		os.Exit(1)
	}
	stop()
	<-serviceDone
}
