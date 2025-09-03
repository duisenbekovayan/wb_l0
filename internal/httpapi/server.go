package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/duisenbekovayan/wb_l0/internal/cache"
	"github.com/duisenbekovayan/wb_l0/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	srv   *http.Server
	cache *cache.Store
	pg    *storage.PG
}

func New(addr string, c *cache.Store, pg *storage.PG) *Server {
	r := chi.NewRouter()

	s := &Server{
		srv:   &http.Server{Addr: addr, Handler: r},
		cache: c,
		pg:    pg,
	}

	// CORS/статик по-простому
	r.Handle("/*", http.FileServer(http.Dir("./web")))

	r.Get("/order/{uid}", s.getOrder)
	return s
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")

	if o, ok := s.cache.Get(uid); ok {
		_ = json.NewEncoder(w).Encode(o)
		return
	}

	o, err := s.pg.GetOrder(r.Context(), uid)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.cache.Set(o)
	_ = json.NewEncoder(w).Encode(o)
}

func (s *Server) Start() error {
	log.Printf("HTTP on %s", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error { return s.srv.Shutdown(ctx) }
