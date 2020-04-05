package exchanges

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/influx6/btclists"
)

var (
	coinLayerURL = "http://api.coinlayer.com/api"
)

// CoinLayer wraps out necessary decorate to make http requests to the
// CoinLayer API for retrieving crypto-currency rates.
//
// CoinLayer does not support precision second, minute, hour retrieval but
// does provide a simple API, with quick start, suitable enough for this need.
//
// We could consider CoinAPI or other providers for more precision data.
// For example, CoinAPI provides endpoints for retrieving on a per minute, hour or more
// base for candle-sticks data values on the rate changes for a giving coin.
type CoinLayer struct {
	APIToken string
	Client   btclists.Client
}

// Rate retrieves rate for giving coin based on fiat currency for specific
// time.
func (c *CoinLayer) Rate(ctx context.Context, coin string, fiat string, time time.Time) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("not found")
}

// Range retrieves all rates for giving coin for giving fiat from time-range.
func (c *CoinLayer) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	return nil, errors.New("not found")
}

func buildRequest(ctx context.Context, token string, method string, path string, queries url.Values, body io.Reader) (*http.Request, error) {
	var targetURL = fmt.Sprintf("%s/%s?access_key=%s&%s", coinLayerURL, path, token, queries.Encode())
	return http.NewRequestWithContext(ctx, method, targetURL, body)
}
