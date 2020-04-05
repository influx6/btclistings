package exchanges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/shopspring/decimal"

	"github.com/influx6/btclists"
)

const (
	MaxLimit          = 5000
	PeriodInterval    = "5MIN"
	CoinApiProdURL    = "https://rest.coinapi.io/"
	CoinApiSandboxURL = "https://rest-sandbox.coinapi.io/"
)

var (
	ErrBadRequest = errors.New("bad request")
)

type ExchangeRate struct {
	Time         time.Time       `json:"time"`
	AssetIdBase  string          `json:"asset_id_base"`
	AssetIdQuote string          `json:"asset_id_quote"`
	Rate         decimal.Decimal `json:"rate"`
}

func (e *ExchangeRate) Valid() error {
	if e.Time.IsZero() {
		return errors.New("exchange data can't have zero time")
	}
	if e.AssetIdQuote == "" || e.AssetIdBase == "" {
		return errors.New("exchange data base or quote value can't be empty")
	}
	return nil
}

type CandleSticks struct {
	Start        time.Time       `json:"time_period_start"`
	End          time.Time       `json:"time_period_end"`
	TimeOpen     time.Time       `json:"time_open"`
	TimeClose    time.Time       `json:"time_close"`
	PriceOpen    decimal.Decimal `json:"price_open"`
	PriceHigh    decimal.Decimal `json:"price_high"`
	PriceLow     decimal.Decimal `json:"price_low"`
	PriceClose   decimal.Decimal `json:"price_close"`
	VolumeTraded decimal.Decimal `json:"volume_traded"`
	TradesCount  uint32          `json:"trades_count"`
}

// CoinAPI wraps out necessary decorate to make http requests to the
// CoinAPI API for retrieving crypto-currency rates.
//
// CoinAPI does not support precision second, minute, hour retrieval but
// does provide a simple API, with quick start, suitable enough for this need.
//
// We could consider CoinAPI or other providers for more precision data.
// For example, CoinAPI provides endpoints for retrieving on a per minute, hour or more
// base for candle-sticks data values on the rate changes for a giving coin.
type CoinAPI struct {
	URL          string
	APIToken     string
	limitReached bool
	Client       btclists.Client
}

// Rate retrieves rate for giving coin based on fiat currency for specific
// time.
func (c *CoinAPI) Rate(ctx context.Context, coin string, fiat string, time time.Time) (btclists.Rate, error) {
	var rate btclists.Rate
	if c.limitReached {
		return rate, btclists.ErrLimitReached
	}

	var path = fmt.Sprintf("%s/v1/exchangerate/%s/%s", c.URL, coin, fiat)

	var query = url.Values{}
	if !time.IsZero() {
		query.Set("time", time.Format(btclists.DateTimeFormat))
	}

	var req, err = buildRequest(ctx, c.APIToken, "GET", path, query, nil)
	if err != nil {
		return rate, err
	}

	var res, resErr = c.Client.Do(req)
	if resErr != nil {
		return rate, err
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return rate, ErrBadRequest
	case 429:
		c.limitReached = true
		return rate, btclists.ErrLimitReached
	case http.StatusUnauthorized:
		return rate, btclists.ErrInvalidToken
	case http.StatusForbidden:
		return rate, btclists.ErrUnauthorized
	case 550:
		return rate, btclists.ErrRateNotFound
	default:
		// nothing to do here
	}

	var exchange ExchangeRate
	if err = json.NewDecoder(res.Body).Decode(&exchange); err != nil {
		return rate, err
	}

	if validErr := exchange.Valid(); validErr != nil {
		return rate, err
	}

	rate.Rate = exchange.Rate
	rate.Fiat = exchange.AssetIdQuote
	rate.Coin = exchange.AssetIdBase
	rate.Time = exchange.Time
	return rate, nil
}

// RangeFrom returns all results from giving time till provided limit.
func (c *CoinAPI) RangeFrom(ctx context.Context, coin string, fiat string, from time.Time, limit int) ([]btclists.Rate, error) {
	return c.Range(ctx, coin, fiat, from, time.Time{}, limit)
}

// Range retrieves all rates for giving coin for giving fiat and crypto-coin pair from provided
// time range (if to is not provided, then till limit requested). Note CoinAPI has a 100,000 record
// limit.
func (c *CoinAPI) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time, limit int) ([]btclists.Rate, error) {
	if c.limitReached {
		return nil, btclists.ErrLimitReached
	}

	if from.IsZero() {
		return nil, errors.New("invalid 'from' time range provided")
	}

	var query = url.Values{}
	query.Set("include_empty_items", "false")
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("time_start", from.Format(btclists.DateTimeFormat))

	if !to.IsZero() {
		query.Set("time_end", to.Format(btclists.DateTimeFormat))
	}

	var path = fmt.Sprintf("%s/v1/ohlcv/%s/%s/history", c.URL, coin, fiat)
	var req, err = buildRequest(ctx, c.APIToken, "GET", path, query, nil)
	if err != nil {
		return nil, err
	}

	var res, resErr = c.Client.Do(req)
	if resErr != nil {
		return nil, err
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, errors.New("bad request")
	case 429:
		c.limitReached = true
		return nil, btclists.ErrLimitReached
	case http.StatusUnauthorized:
		return nil, btclists.ErrInvalidToken
	case http.StatusForbidden:
		return nil, btclists.ErrUnauthorized
	case 550:
		return nil, btclists.ErrRateNotFound
	default:
		// nothing to do here
	}

	var sticks []CandleSticks
	if err = json.NewDecoder(res.Body).Decode(&sticks); err != nil {
		return nil, err
	}

	var rates = make([]btclists.Rate, 0, len(sticks))
	for _, candle := range sticks {
		var rate btclists.Rate
		rate.Rate = candle.PriceClose
		rate.Time = candle.End
		rate.Fiat = fiat
		rate.Coin = coin
		rates = append(rates, rate)
	}

	return rates, nil
}

func buildRequest(ctx context.Context, token string, method string, path string, queries url.Values, body io.Reader) (*http.Request, error) {
	var targetURL = fmt.Sprintf("%s?%s", path, queries.Encode())
	var req, err = http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-CoinAPI-Key", token)
	return req, nil
}
