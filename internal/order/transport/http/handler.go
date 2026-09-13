package http

import (
	"encoding/json"
	"errors"
	orderdomain "gg-sell-like-core/internal/order"
	"gg-sell-like-core/pkg/httpx"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	uc  *orderdomain.Usecase
	log *slog.Logger
}

func NewHandler(
	uc *orderdomain.Usecase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		uc:  uc,
		log: log.With("component", "order.http"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.Create)
	mux.HandleFunc("GET /orders/{id}", h.GetByID)
}

type createRequest struct {
	SKUs []string `json:"skus"`
}

type createResponse struct {
	Status  string `json:"status"`
	OrderID int64  `json:"orderId"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
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

	o, err := h.uc.Create(
		r.Context(),
		orderdomain.CreateInput{
			SKUs: req.SKUs,
		},
		time.Now().UTC(),
	)
	if err != nil {
		switch {
		case errors.Is(err, orderdomain.ErrInvalidSKU):
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("invalid sku"),
			); err != nil {
				h.log.Error("write json on invalid sku", "err", err)
			}
			return
		case errors.Is(err, orderdomain.ErrAtLeastOneSKURequired):
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("at least one sku required"),
			); err != nil {
				h.log.Error("write json on at least one sku required", "err", err)
			}
			return
		default:
			if err := httpx.WriteJSON(
				w,
				http.StatusInternalServerError,
				httpx.NewErrResponse("internal error"),
			); err != nil {
				h.log.Error("write json on internal error", "err", err)
			}
			return
		}
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusOK,
		createResponse{
			Status:  "ok",
			OrderID: o.ID,
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}
}

type order struct {
	ID              int64              `json:"id"`
	Status          orderdomain.Status `json:"status"`
	Amount          int64              `json:"amount"`
	DeliveredAmount int64              `json:"deliveredAmount"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

func orderToHTTP(o orderdomain.Order) order {
	return order{
		ID:              o.ID,
		Status:          o.Status,
		Amount:          o.Amount,
		DeliveredAmount: o.DeliveredAmount,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

type getByIDResponse struct {
	Status string `json:"status"`
	Order  order  `json:"order"`
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		if err := httpx.WriteJSON(
			w,
			http.StatusBadRequest,
			httpx.NewErrResponse("invalid id"),
		); err != nil {
			h.log.Error("write json on invalid id", "err", err)
		}
		return
	}

	o, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, orderdomain.ErrNotFound):
			if err := httpx.WriteJSON(
				w,
				http.StatusNotFound,
				httpx.NewErrResponse("order not found"),
			); err != nil {
				h.log.Error("write json on order not found", "err", err)
			}
			return
		default:
			if err := httpx.WriteJSON(
				w,
				http.StatusInternalServerError,
				httpx.NewErrResponse("internal error"),
			); err != nil {
				h.log.Error("write json on internal error", "err", err)
			}
			return
		}
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusOK,
		getByIDResponse{
			Status: "ok",
			Order:  orderToHTTP(o),
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}
}
