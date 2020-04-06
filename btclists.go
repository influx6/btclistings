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

type RateService interface {
	// At returns giving crypto-currency pair exchange around provided time stamp.
	At(ctx context.Context, crypto string, fiat string, when time.Time) (Rate, error)

	// Latest returns latest crypto-currency and fiat-currency pair from underline provider.
	Latest(ctx context.Context, crypto string, fiat string) (Rate, error)

	// Range returns all known Rate for crypto-currency and fiat-currency pair within
	// time range (i.e from 'start' to 'end' time range)
	Range(ctx context.Context, crypto string, currency string, start time.Time, end time.Time) ([]Rate, error)
}

// RateDB defines expectation for minimum support required
// a db store for storing and retrieving Rates.
//
// NOTE To Reviewer: There are times when being specific and tying implementation
// details together is far better than an interface, say a higher level struct which
// may need deeper level access to another implementation details (usually internal to a pkg).
//
// The level for this need is high enough for me to abstract necessary methods
// as a contract, but this may not always be the case. It is tempting to define
// everything with interfaces but this can lead to Interface poisoning. Also
// interfaces do come with allocation cost.
//
type RateDB interface {
	RateService

	Add(ctx context.Context, rate Rate) error
	AddBatch(ctx context.Context, rate []Rate) error
	Oldest(ctx context.Context, coin string, fiat string) (Rate, error)
}
