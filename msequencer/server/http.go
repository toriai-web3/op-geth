package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/rs/cors"
)

var (
	once   sync.Once
	server *Server
)

type Server struct {
	port         int
	httpListener net.Listener
	handler      *HttpHandler
}

func NewServer(port int) *Server {
	once.Do(func() {
		server = &Server{
			port: port,
		}
	})
	return server
}

func (server *Server) Start() error {
	var (
		listener net.Listener
		srv      *http.Server
		err      error
	)

	handler, err := NewHttpHandler()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	allowOrigins := make([]string, 1)
	allowOrigins[0] = "*"
	mux.Handle("/", newCorsHandler(handler, allowOrigins))

	// start http listener with http/1.1
	listener, err = net.Listen("tcp", fmt.Sprintf(":%d", server.port))
	if err != nil {
		return err
	}
	srv = newHTTPServer(mux, nil)

	// for-loop here
	srv.Serve(listener)

	return nil
}

func (server *Server) Stop() error {
	//TODO: server stop
	return nil
}

func newCorsHandler(srv *HttpHandler, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(srv)
}

func newHTTPServer(mux *http.ServeMux, tlsConfig *tls.Config) *http.Server {
	return &http.Server{
		Handler:     mux,
		ReadTimeout: readTimeout,
		TLSConfig:   tlsConfig,
	}
}
