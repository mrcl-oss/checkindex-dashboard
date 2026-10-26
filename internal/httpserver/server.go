package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const shutdownTimeout = 10 * time.Second

type managedServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

// Serve runs the HTTP server until it exits or ctx is canceled, then shuts it down gracefully.
func Serve(ctx context.Context, server managedServer) error {
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.ListenAndServe()
	}()

	select {
	case err := <-serveResult:
		return normalizeServeError(err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return normalizeServeError(<-serveResult)
	}
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
