package infrahttp

import "net/http"

func registerHealthcheck(router *Router) {
	router.Handle("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
}
