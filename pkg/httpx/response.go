package httpx

type Response struct {
	Status string `json:"status"`
}

type ErrResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func NewErrResponse(reason string) ErrResponse {
	return ErrResponse{
		Status: "error",
		Reason: reason,
	}
}
