package infrahttp

import "net/http"

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux *http.ServeMux
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) Handle(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	var wrapped http.Handler = handler
	for i := len(middleware) - 1; i >= 0; i-- {
		wrapped = middleware[i](wrapped)
	}

	r.mux.Handle(pattern, wrapped)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	r.mux.ServeHTTP(w, request)
}
