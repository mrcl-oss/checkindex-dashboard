package httpserver

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestServeShutsDownAfterContextCancellation(t *testing.T) {
	server := newFakeServer(nil)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- Serve(ctx, server)
	}()

	select {
	case <-server.started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("server did not begin serving")
	}
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not finish after cancellation")
	}
	select {
	case <-server.shutdownCalled:
	default:
		t.Fatal("Shutdown() was not called")
	}
}

func TestServeReturnsUnexpectedListenerError(t *testing.T) {
	wantErr := errors.New("listener failed")
	server := newFakeServer(wantErr)
	if err := Serve(context.Background(), server); !errors.Is(err, wantErr) {
		t.Fatalf("Serve() error = %v, want %v", err, wantErr)
	}
}

type fakeServer struct {
	started        chan struct{}
	stop           chan struct{}
	shutdownCalled chan struct{}
	listenErr      error
	stopOnce       sync.Once
}

func newFakeServer(listenErr error) *fakeServer {
	return &fakeServer{
		started:        make(chan struct{}),
		stop:           make(chan struct{}),
		shutdownCalled: make(chan struct{}),
		listenErr:      listenErr,
	}
}

func (s *fakeServer) ListenAndServe() error {
	close(s.started)
	if s.listenErr != nil {
		return s.listenErr
	}
	<-s.stop
	return http.ErrServerClosed
}

func (s *fakeServer) Shutdown(context.Context) error {
	close(s.shutdownCalled)
	s.stopOnce.Do(func() { close(s.stop) })
	return nil
}

func (s *fakeServer) Close() error {
	s.stopOnce.Do(func() { close(s.stop) })
	return nil
}
