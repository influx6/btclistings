package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/influx6/btclists"
)

var (
	ErrNoTimestamp = errors.New("no timestamp provided, use t query")
)

type RateResponse struct {
	Data float64 `json:"rate"`
}

type RateError struct {
	Error string `json:"error"`
}

// NOTE: All http API handlers are written to support specified crypto-currency to
// specific fiat currency. We have no need for supporting multiple coins and fiats,
// as this is to be a simple implementation.

// GetLatest uses provided RateServer for specific fiat and coin.
//
// Route: /{version}/{route} e.g /v1/latest
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error: { error: {error text} } with status code in range 400-500.
//
func GetLatest(rates btclists.RateServer, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		panic("not implemented")
	}
}

// GetLatestAt uses provided RateServer returning price of specific fiat and crypto-coin
// at provided timestamp. Timestamp is expected to be ISO 8601 format.
//
// Route: /{version}/{route}?t={timestamp} e.g /v1/latest_at?t={timestamp}
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error Response: { error: {error text} } with status code in range 400-500.
//
func GetLatestAt(rates btclists.RateServer, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		panic("not implemented")
	}
}

// validateAndRetrieveAtTimestamp embodies validation logic necessary
// to verify expected timestamp for giving request.
//
// We could move this into a middleware phase, but for now, I want
// it here as it make sense to have it as part of the handler layer
// for easier reading.
func validateAndRetrieveAtTimestamp(r *http.Request) (time.Time, error) {
	var t = r.URL.Query().Get("t")
	if t == "" {
		return time.Time{}, ErrNoTimestamp
	}
	return time.Parse("", t)
}

// GetAverageFor uses provided RateServer returning price of for specific time range.
// Timestamps are expected to be ISO 8601 format strings.
//
// Route: /{version}/{route}?from={timestamp}&to={timestamp} e.g /v1/average?from={timestamp}&to={timestamp}
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error Response: { error: {error text} } with status code in range 400-500.
//
func GetAverageFor(server btclists.RateServer, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		panic("not implemented")
	}
}

func respondWithRate(writer http.ResponseWriter, rate float64) error {
	return json.NewEncoder(writer).Encode(RateResponse{rate})
}
