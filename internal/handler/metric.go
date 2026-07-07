package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ezhigval/go-toolkit/httputil"
	"github.com/ezhigval/metrics-collector/internal/model"
	promreg "github.com/ezhigval/metrics-collector/internal/prom"
	"github.com/ezhigval/metrics-collector/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc  *service.MetricService
	prom *promreg.Registry
}

func New(svc *service.MetricService, prom *promreg.Registry) *Handler {
	return &Handler{svc: svc, prom: prom}
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req model.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.recordHTTP(r, http.StatusBadRequest, start)
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "invalid json", err))
		return
	}

	n, err := h.svc.Ingest(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrEmptyBatch) || errors.Is(err, service.ErrInvalidName) {
			status = http.StatusBadRequest
		}
		h.recordHTTP(r, status, start)
		httputil.WriteError(w, httputil.NewAppError(status, "INGEST_FAILED", err.Error(), err))
		return
	}

	h.recordHTTP(r, http.StatusAccepted, start)
	httputil.WriteJSON(w, http.StatusAccepted, map[string]any{"accepted": n})
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	name := chi.URLParam(r, "name")
	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	series, err := h.svc.Query(r.Context(), name, from, to, limit)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidName) {
			status = http.StatusBadRequest
		}
		h.recordHTTP(r, status, start)
		httputil.WriteError(w, httputil.NewAppError(status, "QUERY_FAILED", err.Error(), err))
		return
	}

	h.recordHTTP(r, http.StatusOK, start)
	httputil.WriteJSON(w, http.StatusOK, series)
}

func (h *Handler) ListNames(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	names, err := h.svc.ListNames(r.Context())
	if err != nil {
		h.recordHTTP(r, http.StatusInternalServerError, start)
		httputil.WriteError(w, httputil.NewAppError(http.StatusInternalServerError, "INTERNAL", "list failed", err))
		return
	}
	h.recordHTTP(r, http.StatusOK, start)
	httputil.WriteJSON(w, http.StatusOK, names)
}

func (h *Handler) Prometheus(w http.ResponseWriter, r *http.Request) {
	h.prom.Handler().ServeHTTP(w, r)
}

func (h *Handler) recordHTTP(r *http.Request, status int, start time.Time) {
	path := r.URL.Path
	if route := chi.RouteContext(r.Context()); route != nil && route.RoutePattern() != "" {
		path = route.RoutePattern()
	}
	h.prom.HTTPRequests.WithLabelValues(r.Method, path, strconv.Itoa(status)).Inc()
	_ = start
}
