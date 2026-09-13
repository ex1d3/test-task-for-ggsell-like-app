package http

import (
	"encoding/json"
	"errors"
	providerdomain "gg-sell-like-core/internal/provider"
	"gg-sell-like-core/pkg/httpx"
	"log/slog"
	"net/http"
	"time"
)

type Handler struct {
	uc  *providerdomain.Provider
	log *slog.Logger
}

func NewHandler(
	uc *providerdomain.Provider,
	log *slog.Logger,
) *Handler {
	return &Handler{
		uc:  uc,
		log: log.With("component", "provider.http"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /issue", h.Issue)
}

type issueRequest struct {
	RequestID string `json:"request_id"`
	SKU       string `json:"sku"`
	OrderID   string `json:"order_id"`
}

type issueResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
}

func (h *Handler) Issue(w http.ResponseWriter, r *http.Request) {
	var req issueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err := httpx.WriteJSON(
			w,
			http.StatusBadRequest,
			httpx.NewErrResponse("invalid body"),
		); err != nil {
			h.log.Error("write json on invalid body", "err", err)
		}
		return
	}

	key, err := h.uc.Issue(
		r.Context(),
		providerdomain.IssueInput{
			RequestID: req.RequestID,
			SKU:       req.SKU,
		},
		time.Now().UTC(),
	)
	if err != nil {
		switch {
		case errors.Is(err, providerdomain.ErrOutOfStock):
			if err := httpx.WriteJSON(
				w,
				http.StatusConflict,
				httpx.NewErrResponse("out of stock"),
			); err != nil {
				h.log.Error("write json on out of stock", "err", err)
			}
		case errors.Is(err, providerdomain.ErrIssueFailed):
			if err := httpx.WriteJSON(
				w,
				http.StatusInternalServerError,
				httpx.NewErrResponse("issue failed"),
			); err != nil {
				h.log.Error("write json on issue failed", "err", err)
			}
		case errors.Is(err, providerdomain.ErrTimedOut):
			if err := httpx.WriteJSON(
				w,
				http.StatusRequestTimeout,
				httpx.NewErrResponse("timed out"),
			); err != nil {
				h.log.Error("write json on timed out", "err", err)
			}
		default:
			if err := httpx.WriteJSON(
				w,
				http.StatusInternalServerError,
				httpx.NewErrResponse("internal error"),
			); err != nil {
				h.log.Error("write json on internal error", "err", err)
			}
		}
		return
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusOK,
		issueResponse{
			Status:    "ok",
			RequestID: req.RequestID,
			Code:      key,
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}
}
