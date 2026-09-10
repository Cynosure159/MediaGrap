package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/mediagrap/mediagrap/internal/automation"
	"github.com/mediagrap/mediagrap/internal/tokens"
	"github.com/mediagrap/mediagrap/internal/webhooks"
)

func integrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, webhooks.ErrInvalid), errors.Is(err, tokens.ErrInvalid):
		writeError(w, 400, "invalid_configuration", "Check the name, sources, expiry and deployment URL policy")
	case errors.Is(err, webhooks.ErrUnavailable):
		writeError(w, 503, "signing_unavailable", "Configure the Webhook signing key file and restart the server")
	case errors.Is(err, webhooks.ErrConflict):
		writeError(w, 409, "state_conflict", "Refresh and check the current state before retrying")
	case errors.Is(err, sql.ErrNoRows):
		writeError(w, 404, "not_found", "Integration not found")
	default:
		writeError(w, 500, "integration_error", "Integration operation failed")
	}
}
func (s *server) integrationRoutes(mux *http.ServeMux) {
	if service := s.runtime.Automation; service != nil {
		mux.HandleFunc("GET /api/v1/automation/plans", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
			items, err := service.AdminPlans(r.Context())
			if err != nil {
				integrationError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"items": items})
		}))
		mux.HandleFunc("POST /api/v1/automation/plans/{id}/approve", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
			session, _ := sessionFromContext(r.Context())
			var in automation.ApplyInput
			if !decodeJSON(w, r, &in) {
				return
			}
			in.PlanID = r.PathValue("id")
			if err := service.Approve(r.Context(), session.User.ID, in); err != nil {
				writeError(w, 409, "approval_conflict", "Plan expired, changed, or no longer authorized")
				return
			}
			w.WriteHeader(204)
		}))
	}

	mux.HandleFunc("GET /api/v1/integrations/status", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		status, err := s.webhooks.Status(r.Context())
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 200, status)
	}))
	mux.HandleFunc("GET /api/v1/webhooks", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		items, err := s.webhooks.List(r.Context())
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
	mux.HandleFunc("POST /api/v1/webhooks", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		var in webhooks.Input
		if !decodeJSON(w, r, &in) {
			return
		}
		value, err := s.webhooks.Create(r.Context(), in)
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 201, value)
	}))
	mux.HandleFunc("GET /api/v1/webhooks/{id}", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		e, err := s.webhooks.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 200, e)
	}))
	mux.HandleFunc("PATCH /api/v1/webhooks/{id}", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		var in webhooks.Input
		if !decodeJSON(w, r, &in) {
			return
		}
		if err := s.webhooks.Update(r.Context(), r.PathValue("id"), in); err != nil {
			integrationError(w, err)
			return
		}
		w.WriteHeader(204)
	}))
	mux.HandleFunc("DELETE /api/v1/webhooks/{id}", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		if err := s.webhooks.Delete(r.Context(), r.PathValue("id")); err != nil {
			integrationError(w, err)
			return
		}
		w.WriteHeader(204)
	}))
	mux.HandleFunc("POST /api/v1/webhooks/{id}/rotate-secret", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.webhooks.Rotate(r.Context(), r.PathValue("id"))
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 201, value)
	}))
	mux.HandleFunc("POST /api/v1/webhooks/{id}/test", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		id, err := s.webhooks.Test(r.Context(), r.PathValue("id"))
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 202, map[string]string{"deliveryId": id})
	}))
	mux.HandleFunc("GET /api/v1/webhooks/{id}/deliveries", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		items, err := s.webhooks.Deliveries(r.Context(), r.PathValue("id"), before)
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
	mux.HandleFunc("POST /api/v1/webhooks/{id}/deliveries/{deliveryId}/retry", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		if err := s.webhooks.Retry(r.Context(), r.PathValue("id"), r.PathValue("deliveryId")); err != nil {
			integrationError(w, err)
			return
		}
		w.WriteHeader(204)
	}))
	mux.HandleFunc("GET /api/v1/api-tokens", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		items, err := s.tokens.List(r.Context(), session.User.ID)
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}))
	mux.HandleFunc("POST /api/v1/api-tokens", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		var in tokens.Input
		if !decodeJSON(w, r, &in) {
			return
		}
		value, err := s.tokens.Create(r.Context(), session.User.ID, in)
		if err != nil {
			integrationError(w, err)
			return
		}
		writeJSON(w, 201, value)
	}))
	mux.HandleFunc("DELETE /api/v1/api-tokens/{id}", s.requireAuthCSRF(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		if err := s.tokens.Revoke(r.Context(), session.User.ID, r.PathValue("id")); err != nil {
			integrationError(w, err)
			return
		}
		w.WriteHeader(204)
	}))
}
