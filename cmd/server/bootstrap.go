package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/elcomware/edupilot/internal/api"
	"github.com/elcomware/edupilot/internal/app"
	"github.com/elcomware/edupilot/internal/kernel/apperr"
)

// newHTTPTransport is the Mode B adapter. It exposes the same application
// services the Wails adapter exposes, over HTTP, and returns the same Result
// envelope, so one frontend gateway serves both modes.
func newHTTPTransport(logger *slog.Logger, edupilot *app.Application) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// The operation names match the desktop gateway exactly:
	// platform.organisation.App.GetCurrent, platform.campus.App.List.
	mux.HandleFunc("POST /api/platform.organisation.App.GetCurrent",
		handle(logger, func(ctx context.Context) api.Result[api.OrganisationView] {
			organisation, err := edupilot.Organisations.GetCurrentOrganisation(ctx)
			if err != nil {
				return api.Fail[api.OrganisationView](err)
			}
			return api.OK(api.NewOrganisationView(organisation))
		}))

	mux.HandleFunc("POST /api/platform.campus.App.List",
		handlePage(logger, func(ctx context.Context, request api.PageRequest) api.Result[api.Page[api.CampusView]] {
			organisation, err := edupilot.Organisations.GetCurrentOrganisation(ctx)
			if err != nil {
				return api.Fail[api.Page[api.CampusView]](err)
			}

			campuses, err := edupilot.Campuses.ListCampuses(ctx, organisation.ID, true)
			if err != nil {
				return api.Fail[api.Page[api.CampusView]](err)
			}
			return api.OK(api.NewPage(api.NewCampusViews(campuses), request))
		}))

	return withLogging(logger, mux)
}

// handler is a typed operation that needs no request payload.
type handler[T any] func(ctx context.Context) api.Result[T]

// pageHandler is a typed operation that reads a page request from the body.
type pageHandler[T any] func(ctx context.Context, request api.PageRequest) api.Result[T]

func handle[T any](logger *slog.Logger, operation handler[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := operation(r.Context())
		writeResult(logger, w, result)
	}
}

func handlePage[T any](logger *slog.Logger, operation pageHandler[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request api.PageRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				writeJSON(w, http.StatusBadRequest,
					api.Fail[any](apperr.ValidationFailed("the request body is not valid JSON")))
				return
			}
		}

		writeResult(logger, w, operation(r.Context(), request.Normalize()))
	}
}

// writeResult renders a Result with the HTTP status that matches its meaning:
// a business rejection is a 4xx the client can act on, an unexpected failure is
// a 500 that carries no internal detail.
func writeResult[T any](logger *slog.Logger, w http.ResponseWriter, result api.Result[T]) {
	if result.OK {
		writeJSON(w, http.StatusOK, result)
		return
	}

	logger.Warn("operation rejected", "code", result.Error.Code, "message", result.Error.Message)
	writeJSON(w, statusForCode(result.Error.Code), result)
}

func statusForCode(code apperr.Code) int {
	switch code {
	case apperr.CodeValidationFailed:
		return http.StatusBadRequest
	case apperr.CodeUnauthenticated:
		return http.StatusUnauthorized
	case apperr.CodeInsufficientPermission:
		return http.StatusForbidden
	case apperr.CodeNotFound:
		return http.StatusNotFound
	case apperr.CodeAlreadyExists, apperr.CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return
	}
}

// statusRecorder captures the response status so the access log can report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"took", time.Since(started))
	})
}
