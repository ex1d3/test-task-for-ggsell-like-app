package http

import (
	"errors"
	productdomain "gg-sell-like-core/internal/product"
	"gg-sell-like-core/pkg/currency"
	"gg-sell-like-core/pkg/xhttp"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	uc  *productdomain.Usecase
	log *slog.Logger
}

func NewHandler(
	uc *productdomain.Usecase,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		uc:  uc,
		log: logger.With("component", "product.http"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /products", h.List)
}

type product struct {
	SKU      string             `json:"sku"`
	Name     string             `json:"name"`
	Type     productdomain.Type `json:"type"`
	Price    int64              `json:"price"`
	Currency currency.Currency  `json:"currency"`
	Image    string             `json:"image"`
}

func productToHTTP(p productdomain.Product) product {
	return product{
		SKU:      p.SKU,
		Name:     p.Name,
		Type:     p.Type,
		Price:    p.Price,
		Currency: p.Currency,
		Image:    p.Image,
	}
}

type listResponse struct {
	Status   string    `json:"status"`
	Products []product `json:"products"`
	Total    int64     `json:"total"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var skip int64
	if v := q.Get("skip"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid skip"),
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
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid limit"),
			); err != nil {
				h.log.Error("write json on limit parse", "err", err)
			}
			return
		}
		limit = n
	}

	output, err := h.uc.List(
		r.Context(),
		productdomain.ListInput{
			Pagination: productdomain.ListPagination{
				Skip:  skip,
				Limit: limit,
			},
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, productdomain.ErrInvalidSkip):
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid skip"),
			); err != nil {
				h.log.Error("write json on invalid skip", "err", err)
			}
			return
		case errors.Is(err, productdomain.ErrInvalidLimit):
			if err := xhttp.WriteJSON(
				w,
				http.StatusBadRequest,
				xhttp.NewErrResponse("invalid limit"),
			); err != nil {
				h.log.Error("write json on invalid limit", "err", err)
			}
			return
		default:
			if err := xhttp.WriteJSON(
				w,
				http.StatusInternalServerError,
				xhttp.NewErrResponse("internal error"),
			); err != nil {
				h.log.Error("write json on internal error", "err", err)
			}
			return
		}
	}

	products := make([]product, 0, len(output.Products))
	for _, p := range output.Products {
		products = append(products, productToHTTP(p))
	}

	if err := xhttp.WriteJSON(
		w,
		http.StatusOK,
		listResponse{
			Status:   "ok",
			Products: products,
			Total:    output.Total,
		},
	); err != nil {
		h.log.Error("write json on ok", "err", err)
	}

}
