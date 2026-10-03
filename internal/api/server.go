package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server runs the operator's HTTP API as a controller-runtime manager runnable.
type Server struct {
	addr    string
	handler http.Handler
	log     logr.Logger
}

// NewServer creates an HTTP API server for the supplied Kubernetes client.
func NewServer(addr string, kubeClient client.Client, reader client.Reader, allowedNamespaces string, logger logr.Logger) *Server {
	return &Server{
		addr:    addr,
		handler: NewPodChaosHandler(kubeClient, reader, strings.Split(allowedNamespaces, ",")),
		log:     logger,
	}
}

// Start serves requests until the manager is shutting down.
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.addr,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		s.log.Error(err, "HTTP API server stopped unexpectedly")
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}

// NeedLeaderElection lets the API accept requests on every manager replica.
func (*Server) NeedLeaderElection() bool {
	return false
}
