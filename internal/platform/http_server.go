package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type HttpServer struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

func NewHttpServer(addr string, handler http.Handler) *HttpServer {
	return &HttpServer{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		shutdownTimeout: 10 * time.Second,
	}
}

func (s *HttpServer) Run(ctx context.Context, log *slog.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	httpLogger := log.With("component", "http")
	httpLogger.Info("listening", "addr", s.server.Addr)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("listen and serve: %w", err)

	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		s.shutdownTimeout,
	)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	err := <-errCh
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
