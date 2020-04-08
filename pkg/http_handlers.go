package pkg

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/influx6/btclists"
)

var (
	ErrInvalidTimestamp = errors.New("timestamp is not valid")
	ErrNoTimestamp      = errors.New("no timestamp provided, use t query")
	ErrUnableToService  = errors.New("unable to service request at the moment")
)

type RateResponse struct {
	Data string `json:"data"`
}

type RateError struct {
	Error string `json:"error"`
}

// NOTE: All http API handlers are written to support specified crypto-currency to
// specific fiat currency. We have no need for supporting multiple coins and fiats,
// as this is to be a simple implementation.

// GetLatest uses provided RateService for specific fiat and coin to return last known
// and available rate for giving pair from provided RateService.
//
// Route: /{version}/{route} e.g /v1/latest
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error: { error: {error text} } with status code in range 400-500.
//
func GetLatest(rates btclists.RateService, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var latest, err = rates.Latest(request.Context(), coin, fiat)
		if err != nil {
			if err == btclists.ErrRateNotFound {
				writer.WriteHeader(http.StatusNotFound)
				respondWithError(writer, err)
				return
			}

			writer.WriteHeader(http.StatusInternalServerError)
			respondWithError(writer, ErrUnableToService)
			return
		}

		writer.WriteHeader(http.StatusOK)
		respondWithRate(writer, latest.Rate)
	}
}

// GetLatestAt uses provided RateService returning price of specific fiat and crypto-coin
// at provided timestamp.
//
// Timestamps are expected to be ISO 8601 format strings encoded properly (URL Encoded).
//
// Route: /{version}/{route}?t={timestamp} e.g /v1/latest_at?t={timestamp}
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error Response: { error: {error text} } with status code in range 400-500.
//
func GetLatestAt(rates btclists.RateService, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var timestamp, err = validateAndRetrieveAtTimestamp(request)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			respondWithError(writer, err)
			return
		}

		var result, rateErr = rates.At(request.Context(), coin, fiat, timestamp)
		if rateErr != nil {
			if rateErr == btclists.ErrRateNotFound {
				writer.WriteHeader(http.StatusNotFound)
				respondWithError(writer, rateErr)
				return
			}

			writer.WriteHeader(http.StatusInternalServerError)
			respondWithError(writer, ErrUnableToService)
			return
		}

		writer.WriteHeader(http.StatusOK)
		respondWithRate(writer, result.Rate)
	}
}

// validateAndRetrieveAtTimestamp embodies validation logic necessary
// to verify expected timestamp for giving request.
func validateAndRetrieveAtTimestamp(r *http.Request) (time.Time, error) {
	var t = r.URL.Query().Get("t")
	return validateTimestampString(t)
}

func validateTimestampString(t string) (time.Time, error) {
	if t == "" {
		return time.Time{}, ErrNoTimestamp
	}

	// is this a RFC3339 or ISO 8601 date timestamp?
	if dateTime, err := time.Parse(btclists.DateTimeFormat, t); err == nil {
		return dateTime.UTC(), nil
	}

	// is this just a simple date format YYYY-MM-DD ?
	if date, err := time.Parse(btclists.DateFormat, t); err == nil {
		return date.UTC(), nil
	}

	return time.Time{}, ErrInvalidTimestamp
}

// GetAverageFor uses provided RateService returning price of for specific time range.
// Timestamps are expected to be ISO 8601 format strings encoded properly (URL Encoded).
//
// Route: /{version}/{route}?from={timestamp}&to={timestamp} e.g /v1/average?from={timestamp}&to={timestamp}
// Response Format: application/json
// Response: { data: {price} } where 'price' is a float64 type.
// Error Response: { error: {error text} } with status code in range 400-500.
//
func GetAverageFor(averageService btclists.RatingsAverageService, ratingService btclists.RateService, fiat string, coin string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var from, to, err = validateAndRetrieveStartAndEndTimestamps(request)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			respondWithError(writer, err)
			return
		}

		// if we are giving same time, just divert to at call
		if from.Equal(to) {
			var atRating, atErr = ratingService.At(request.Context(), coin, fiat, from)
			if atErr != nil {
				if atErr == btclists.ErrRateNotFound {
					writer.WriteHeader(http.StatusNotFound)
					respondWithError(writer, atErr)
					return
				}

				writer.WriteHeader(http.StatusInternalServerError)
				respondWithError(writer, ErrUnableToService)
				return
			}

			respondWithRate(writer, atRating.Rate)
			return
		}

		var average, avgErr = averageService.AverageForRange(request.Context(), coin, fiat, from, to)
		if avgErr != nil {
			if avgErr == btclists.ErrRateNotFound {
				writer.WriteHeader(http.StatusNotFound)
				respondWithError(writer, avgErr)
				return
			}

			writer.WriteHeader(http.StatusInternalServerError)
			respondWithError(writer, avgErr)
			return
		}

		respondWithRate(writer, average)
	}
}

// validateAndRetrieveStartAndEndTimestamp embodies validation logic necessary
// to verify expected timestamps for giving request.
func validateAndRetrieveStartAndEndTimestamps(r *http.Request) (time.Time, time.Time, error) {
	var fromTs = r.URL.Query().Get("from")
	var from, fromErr = validateTimestampString(fromTs)
	if fromErr != nil {
		return time.Time{}, time.Time{}, errors.New("from timestamp value is invalid")
	}

	var toTs = r.URL.Query().Get("to")
	var to, toErr = validateTimestampString(toTs)
	if toErr != nil {
		return time.Time{}, time.Time{}, errors.New("to timestamp value is invalid")
	}

	return from, to, nil
}

func respondWithRate(writer http.ResponseWriter, rate decimal.Decimal) {
	if err := json.NewEncoder(writer).Encode(RateResponse{Data: rate.String()}); err != nil {
		log.Printf("[ALERT] JSON encoding just exploded, that is bad: %+s", err)
	}
}

func respondWithError(writer http.ResponseWriter, err error) {
	if err := json.NewEncoder(writer).Encode(RateError{Error: err.Error()}); err != nil {
		log.Printf("[ALERT] JSON encoding just exploded, that is bad: %+s", err)
	}
}
