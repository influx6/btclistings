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
	Id   int             `json:"id" yaml:"id"`
	Date time.Time       `json:"date" yaml:"date"`
	Rate decimal.Decimal `json:"rate" yaml:"rate"`
	Coin string          `json:"coin" yaml:"coin"`
	Fiat string          `json:"fiat" yaml:"fiat"`
}

// Client is defined here as an interface for 2 specific reasons:
//
// 1. Easier to swap underline client handling request easily.
// 2. If we so wish to provide coverage tests for the code built
// 	  to handle responses from endpoint, this becomes easier.
//
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type RateService interface {
	// At returns giving crypto-currency pair exchange around provided time stamp.
	At(ctx context.Context, crypto string, fiat string, when time.Time) (Rate, error)

	// Latest returns latest crypto-currency and fiat-currency pair from underline provider.
	Latest(ctx context.Context, crypto string, fiat string) (Rate, error)

	// Range returns all known Rate for crypto-currency and fiat-currency pair within
	// time range (i.e from 'start' to 'end' time range)
	Range(ctx context.Context, crypto string, currency string, start time.Time, end time.Time) ([]Rate, error)
}

type RatingsAverageService interface {
	AverageForRange(ctx context.Context, crypto string, currency string, start time.Time, end time.Time) (decimal.Decimal, error)
}

// RatesDB defines expectation for minimum support required
// a db store for storing and retrieving Rates.
type RatesDB interface {
	RateService
	RatingsAverageService

	// Add adds giving rate into db
	Add(ctx context.Context, rate Rate) error

	// AddBatch adds provided batch into db.
	AddBatch(ctx context.Context, rate []Rate) error

	// Oldest returns oldest rate since time began.
	Oldest(ctx context.Context, coin string, fiat string) (Rate, error)

	// CountFor returns count for records between provided time ranges
	CountForRange(ctx context.Context, crypto string, currency string, start time.Time, end time.Time) (int, error)
}
