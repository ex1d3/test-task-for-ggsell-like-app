package http

import (
	"encoding/json"
	"errors"
	"fmt"
	webhookdomain "gg-sell-like-core/internal/webhook"
	"gg-sell-like-core/pkg/currency"
	"gg-sell-like-core/pkg/xhttp"
	"log/slog"
	"net/http"
	"time"
)

type Handler struct {
	uc  *webhookdomain.Usecase
	log *slog.Logger
}

func NewHandler(
	uc *webhookdomain.Usecase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		uc:  uc,
		log: log.With("component", "webhook.http"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks", h.Handle)
}

type handleRequest struct {
	EventID  string               `json:"event_id"`
	OrderID  int64                `json:"order_id"`
	Status   webhookdomain.Status `json:"status"`
	Amount   int64                `json:"amount"`
	Currency currency.Currency    `json:"currency"`
}

type handleResponse struct {
	Status string `json:"status"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req handleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err := xhttp.WriteJSON(
			w,
			http.StatusBadRequest,
			xhttp.NewErrResponse("invalid body"),
		); err != nil {
			h.log.Error("write json on invalid body", "err", err)
		}
		return
	}

	if err := h.uc.Handle(
		r.Context(),
		webhookdomain.HandleInput{
			EventID:  req.EventID,
			OrderID:  req.OrderID,
			Status:   req.Status,
			Amount:   req.Amount,
			Currency: req.Currency,
		},
		time.Now().UTC(),
	); err != nil {
		switch {
		case errors.Is(err, webhookdomain.ErrInvalidStatus):
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid status"),
			); err != nil {
				h.log.Error("write json on invalid status", "err", err)
			}
		case errors.Is(err, webhookdomain.ErrInvalidCurrency):
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid currency"),
			); err != nil {
				h.log.Error("write json on invalid currency", "err", err)
			}
		case errors.Is(err, webhookdomain.ErrOrderNotFound):
			if err := xhttp.WriteJSON(
				w,
				http.StatusInternalServerError,
				xhttp.NewErrResponse("order not found"),
			); err != nil {
				h.log.Error("write json on invalid currency", "err", err)
			}
		default:
			if err := xhttp.WriteJSON(
				w,
				http.StatusInternalServerError,
				xhttp.NewErrResponse(fmt.Sprintf("internal server err: %v", err)),
			); err != nil {
				h.log.Error("write json on internal error", "err", err)
			}
		}
		return
	}

	if err := xhttp.WriteJSON(
		w,
		http.StatusOK,
		handleResponse{
			Status: "ok",
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}
}
