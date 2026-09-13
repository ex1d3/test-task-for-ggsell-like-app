package http

import (
	"errors"
	itemdomain "gg-sell-like-core/internal/order/item"
	"gg-sell-like-core/pkg/httpx"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	uc  *itemdomain.Usecase
	log *slog.Logger
}

func NewHandler(
	uc *itemdomain.Usecase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		uc:  uc,
		log: log.With("component", "order.item.http"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /orders/{id}/items", h.listByOrder)
}

type item struct {
	ID        int64             `json:"id"`
	OrderID   int64             `json:"orderId"`
	SKU       string            `json:"sku"`
	Amount    int64             `json:"amount"`
	Code      string            `json:"code,omitempty"`
	Status    itemdomain.Status `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func itemToHTTP(i itemdomain.Item) item {
	return item{
		ID:        i.ID,
		OrderID:   i.OrderID,
		SKU:       i.SKU,
		Amount:    i.Amount,
		Code:      i.Code,
		Status:    i.Status,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
	}
}

type listByOrderResponse struct {
	Status string `json:"status"`
	Items  []item `json:"items"`
	Total  int64  `json:"total"`
}

func (h *Handler) listByOrder(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()

	var skip int64
	if v := q.Get("skip"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("invalid skip"),
			); err != nil {
				h.log.Error("write json on skip parse", "err", err)
			}
			return
		}
		skip = n
	}

	var limit int64
	if v := q.Get("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("invalid limit"),
			); err != nil {
				h.log.Error("write json on limit parse", "err", err)
			}
			return
		}
		limit = n
	}

	output, err := h.uc.ListByOrder(
		r.Context(),
		itemdomain.ListByOrderInput{
			OrderID: id,
			Pagination: itemdomain.ListByOrderPagination{
				Skip:  skip,
				Limit: limit,
			},
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, itemdomain.ErrInvalidSkip):
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("invalid skip"),
			); err != nil {
				h.log.Error("write json on invalid skip", "err", err)
			}
			return
		case errors.Is(err, itemdomain.ErrInvalidLimit):
			if err := httpx.WriteJSON(
				w,
				http.StatusBadRequest,
				httpx.NewErrResponse("invalid limit"),
			); err != nil {
				h.log.Error("write json on invalid limit", "err", err)
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

	items := make([]item, 0, len(output.Items))
	for _, i := range output.Items {
		items = append(items, itemToHTTP(i))
	}

	if err := httpx.WriteJSON(
		w,
		http.StatusOK,
		listByOrderResponse{
			Status: "ok",
			Items:  items,
			Total:  output.Total,
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}
}
