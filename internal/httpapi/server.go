package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/duisenbekovayan/wb_l0/internal/cache"
	"github.com/duisenbekovayan/wb_l0/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	srv   *http.Server
	cache *cache.Store
	pg    *storage.PG
}

func New(addr string, c *cache.Store, pg *storage.PG) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	s := &Server{
		srv:   &http.Server{Addr: addr, Handler: r},
		cache: c,
		pg:    pg,
	}

	// 1) Сначала API (иначе его "съест" catch-all для статики)
	r.Get("/order/{uid}", s.getOrder)

	// 2) Статика. Лучше отдать только под корнем.
	fs := http.FileServer(http.Dir("./web"))
	r.Handle("/*", fs) // теперь /order не перехватывается, т.к. объявлен выше

	return s
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	uid := chi.URLParam(r, "uid")

	// cache first
	if o, ok := s.cache.Get(uid); ok {
		_ = json.NewEncoder(w).Encode(o)
		return
	}

	// then DB
	o, err := s.pg.GetOrder(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
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
