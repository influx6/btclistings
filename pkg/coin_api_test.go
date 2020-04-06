package pkg_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/influx6/btclists/pkg"

	"github.com/stretchr/testify/require"
)

/*
	Note To Reviewer:

	These can't be considered integration tests, as they exist pretty much to
	verify expected behaviour when valid response for success or failure is met.
*/

// MockClient will stand in as our http client, so we can verify
// certain expectations about our coin layer integration layer.
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do implements the our HttpClient interface.
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestCoinAPI_Range_ValidateURLWithoutToTime(t *testing.T) {
	var httpClient MockClient
	var coinLayer = pkg.CoinAPI{
		URL:      APIURI,
		APIToken: APIToken,
		Client:   &httpClient,
	}

	var formattedTime = url.QueryEscape(someTimeFormatted)
	var expectedURL = fmt.Sprintf("%s/v1/ohlcv/%s/%s/history?include_empty_items=false&limit=1&period_id=%s&time_start=%s", APIURI, COIN, FIAT, pkg.PeriodInterval, formattedTime)
	httpClient.DoFunc = func(req *http.Request) (response *http.Response, err error) {
		require.Equal(t, expectedURL, req.URL.String())
		return nil, errors.New("not concerned")
	}

	_, _ = coinLayer.Range(context.Background(), COIN, FIAT, someTime, time.Time{}, 1)
}

func TestCoinAPI_Range_ValidateURLWithToTime(t *testing.T) {
	var httpClient MockClient
	var coinLayer = pkg.CoinAPI{
		URL:      APIURI,
		APIToken: APIToken,
		Client:   &httpClient,
	}

	var formattedTime = url.QueryEscape(someTimeFormatted)
	var formattedOtherTime = url.QueryEscape(someOtherTimeFormatted)
	var expectedURL = fmt.Sprintf(
		"%s/v1/ohlcv/%s/%s/history?include_empty_items=false&limit=1&period_id=%s&time_end=%s&time_start=%s",
		APIURI,
		COIN,
		FIAT,
		pkg.PeriodInterval,
		formattedOtherTime,
		formattedTime,
	)

	httpClient.DoFunc = func(req *http.Request) (response *http.Response, err error) {
		require.Equal(t, expectedURL, req.URL.String())
		return nil, errors.New("not concerned")
	}

	_, _ = coinLayer.Range(context.Background(), COIN, FIAT, someTime, someTimeLater, 1)
}

func TestCoinAPI_Rate_ValidateURLWithoutTime(t *testing.T) {
	var httpClient MockClient
	var coinLayer = pkg.CoinAPI{
		URL:      APIURI,
		APIToken: APIToken,
		Client:   &httpClient,
	}

	var expectedURL = fmt.Sprintf("%s/v1/exchangerate/%s/%s?", APIURI, COIN, FIAT)
	httpClient.DoFunc = func(req *http.Request) (response *http.Response, err error) {
		require.Equal(t, expectedURL, req.URL.String())
		return nil, errors.New("not concerned")
	}

	coinLayer.Rate(context.Background(), COIN, FIAT, time.Time{})
}

func TestCoinAPI_Rate_ValidateURLWithTime(t *testing.T) {
	var httpClient MockClient
	var coinLayer = pkg.CoinAPI{
		URL:      APIURI,
		APIToken: APIToken,
		Client:   &httpClient,
	}

	var formattedTime = url.QueryEscape(someTimeFormatted)
	var expectedURL = fmt.Sprintf("%s/v1/exchangerate/%s/%s?time=%s", APIURI, COIN, FIAT, formattedTime)
	httpClient.DoFunc = func(req *http.Request) (response *http.Response, err error) {
		require.Equal(t, expectedURL, req.URL.String())
		return nil, errors.New("not concerned")
	}

	coinLayer.Rate(context.Background(), COIN, FIAT, someTime)
}
