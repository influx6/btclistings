package btclists

import "time"

const (
	TimeFormat = time.RFC3339
)

type Rate struct {
	Time time.Time `json:"time"`
	Rate float64   `json:"rate"`
	Coin string    `json:"coin"`
	Fiat string    `json:"fiat"`
}

type RateServer interface {
	// At returns giving crypto-currency pair exchange around provided time stamp.
	At(crypto string, fiat string, when time.Time) (Rate, error)

	// Latest returns latest crypto-currency and fiat-currency pair from underline provider.
	Latest(crypto string, fiat string) (Rate, error)

	// Range returns all known Rate for crypto-currency and fiat-currency pair within
	// time range (i.e from 'start' to 'end' time range)
	Range(crypto string, currency string, start time.Time, end time.Time) ([]Rate, error)
}

type RateStore interface {
	// Add adds Rate into underline store.
	Add(data Rate) error
}

// RateProvisionService provides both storage and retrieval of rates.
type RateProvisionService interface {
	RateServer
	RateStore
}
