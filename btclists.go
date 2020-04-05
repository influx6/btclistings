package btclists

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

const (
	DateFormat     = "2006-01-02"
	DateTimeFormat = time.RFC3339
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrLimitReached = errors.New("limited reached")
	ErrUnauthorized = errors.New("unauthorized request")
	ErrRateNotFound = errors.New("unable to retrieve or find rate")
)

type Rate struct {
	Time time.Time       `json:"time"`
	Rate decimal.Decimal `json:"rate"`
	Coin string          `json:"coin"`
	Fiat string          `json:"fiat"`
}

// Client is defined here as an interface for 2 specific reasons:
//
// 1. Easier to swap underline client handling request easily.
// 2. If we so wish to provide coverage tests for the code built
// 	  to handle responses from endpoint becomes easier.
//
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimeRateService exposes a method to retrieve giving
// rate for coin to fiat within a giving duration.
type TimeRateService interface {
	Rate(ctx context.Context, coin string, fiat string, duration time.Duration) (Rate, error)
}

type RateServer interface {
	// At returns giving crypto-currency pair exchange around provided time stamp.
	At(ctx context.Context, crypto string, fiat string, when time.Time) (Rate, error)

	// Latest returns latest crypto-currency and fiat-currency pair from underline provider.
	Latest(ctx context.Context, crypto string, fiat string) (Rate, error)

	// Range returns all known Rate for crypto-currency and fiat-currency pair within
	// time range (i.e from 'start' to 'end' time range)
	Range(ctx context.Context, crypto string, currency string, start time.Time, end time.Time) ([]Rate, error)
}

type RateStore interface {
	// Add adds Rate into underline store.
	Add(ctx context.Context, data Rate) error
}
