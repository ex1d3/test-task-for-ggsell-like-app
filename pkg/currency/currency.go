package currency

type Currency string

const (
	Unknown Currency = ""
	RUB     Currency = "rub"
)

func (c Currency) IsValid() bool {
	switch c {
	case RUB:
		return true
	default:
		return false
	}
}
