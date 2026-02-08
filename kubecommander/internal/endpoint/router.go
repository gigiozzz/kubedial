package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router defines the interface for HTTP routing
type Router interface {
	Route(pattern string, fn func(r Router))
	Get(pattern string, handler http.HandlerFunc)
	Post(pattern string, handler http.HandlerFunc)
	Put(pattern string, handler http.HandlerFunc)
	Delete(pattern string, handler http.HandlerFunc)
	Use(middlewares ...func(http.Handler) http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// ChiRouter implements Router using chi
type ChiRouter struct {
	mux chi.Router
}

// NewChiRouter creates a new ChiRouter
func NewChiRouter() *ChiRouter {
	return &ChiRouter{mux: chi.NewRouter()}
}

// Route creates a sub-router
func (r *ChiRouter) Route(pattern string, fn func(Router)) {
	r.mux.Route(pattern, func(cr chi.Router) {
		fn(&ChiRouter{mux: cr})
	})
}

// Get registers a GET handler
func (r *ChiRouter) Get(pattern string, handler http.HandlerFunc) {
	r.mux.Get(pattern, handler)
}

// Post registers a POST handler
func (r *ChiRouter) Post(pattern string, handler http.HandlerFunc) {
	r.mux.Post(pattern, handler)
}

// Put registers a PUT handler
func (r *ChiRouter) Put(pattern string, handler http.HandlerFunc) {
	r.mux.Put(pattern, handler)
}

// Delete registers a DELETE handler
func (r *ChiRouter) Delete(pattern string, handler http.HandlerFunc) {
	r.mux.Delete(pattern, handler)
}

// Use adds middleware
func (r *ChiRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.mux.Use(middlewares...)
}

// ServeHTTP implements http.Handler
func (r *ChiRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// URLParam extracts a URL parameter from the request
func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
