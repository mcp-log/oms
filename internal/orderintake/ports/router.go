package ports

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a Chi router with all order intake routes.
func NewRouter(h *HTTPHandler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/v1", func(r chi.Router) {
		r.Post("/orders", h.HandleCreateOrder)
		r.Get("/orders", h.HandleListOrders)
		r.Get("/orders/{orderId}", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleGetOrder(w, rq, chi.URLParam(rq, "orderId"))
		})
		r.Get("/orders/{orderId}/lines", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleListOrderLines(w, rq, chi.URLParam(rq, "orderId"))
		})
		r.Post("/orders/{orderId}/confirm", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleConfirmOrder(w, rq, chi.URLParam(rq, "orderId"))
		})
		r.Post("/orders/{orderId}/cancel", func(w http.ResponseWriter, rq *http.Request) {
			h.HandleCancelOrder(w, rq, chi.URLParam(rq, "orderId"))
		})
	})

	return r
}
