package restapi

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	zkvote "github.com/unitychain/zkvote-node/zkvote/service"
)

// Server ...
type Server struct {
	*RESTAPI
	router http.Handler
	addr   string
}

// NewServer ...
func NewServer(node *zkvote.Node, serverAddr string) (*Server, error) {
	// get all HTTP REST API handlers available for controller API
	restService, err := NewRESTAPI(node)
	if err != nil {
		return nil, fmt.Errorf("failed to start server:  %w", err)
	}

	handlers := restService.GetHandlers()
	router := mux.NewRouter()

	for _, handler := range handlers {
		router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())
	}

	handler := cors.AllowAll().Handler(router)

	server := &Server{
		RESTAPI: restService,
		router:  handler,
		addr:    serverAddr,
	}

	return server, nil
}

// ListenAndServe starts the server using the standard Go HTTP server implementation.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.router)
}
