package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	phttp "github.com/influx6/btclists/pkg/http"

	"github.com/influx6/btclists"
)

const (
	fiat = "USD"
	coin = "BTC"
)

var _ btclists.RateServer = (*RateServerMock)(nil)

var someTime = time.Now()
var someTimeLater = someTime.Add(time.Hour * 3600)
var someRate = btclists.Rate{
	Time: someTime,
	Rate: 43.322,
	Coin: coin,
	Fiat: fiat,
}

// RateServerMock implements btclists.RateServer.
type RateServerMock struct {
	LatestFunc func(ctx context.Context, coin string, fiat string) (btclists.Rate, error)
	AtFunc     func(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error)
	RangeFunc  func(ctx context.Context, coin string, fiat string, from, to time.Time) ([]btclists.Rate, error)
}

// Range implements btclists.RateServer interface.
//
// We test the functions using this implementation and not this
// implementation.
func (rs RateServerMock) Range(ctx context.Context, coin string, fiat string, from, to time.Time) ([]btclists.Rate, error) {
	return rs.RangeFunc(ctx, coin, fiat, from, to)
}

// At implements btclists.RateServer interface.
//
// We test the functions using this implementation and not this
// implementation.
func (rs *RateServerMock) At(ctx context.Context, coin string, fiat string, ts time.Time) (btclists.Rate, error) {
	return rs.AtFunc(ctx, coin, fiat, ts)
}

// Latest implements btclists.RateServer interface.
//
// We test the functions using this implementation and not this
// implementation.
func (rs *RateServerMock) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	return rs.LatestFunc(ctx, coin, fiat)
}

func TestLatestHandlerFailure(t *testing.T) {
	var rates = new(RateServerMock)
	rates.LatestFunc = func(ctx context.Context, cn string, ft string) (rate btclists.Rate, err error) {
		require.Equal(t, coin, cn)
		require.Equal(t, fiat, ft)
		err = btclists.ErrRateNotFound
		return
	}

	var httpFunc = phttp.GetLatest(rates, fiat, coin)

	var response = httptest.NewRecorder()
	var request = httptest.NewRequest("GET", "/latest", nil)

	httpFunc(response, request)

	require.NotEqual(t, 0, response.Body.Len())
	require.Equal(t, http.StatusNotFound, response.Code)

	var rateError phttp.RateError
	var err = json.NewDecoder(response.Body).Decode(&rateError)
	require.Nil(t, err)
	require.Equal(t, btclists.ErrRateNotFound.Error(), rateError.Error)
}

func TestLatestHandlerSuccess(t *testing.T) {
	var rates = new(RateServerMock)
	rates.LatestFunc = func(ctx context.Context, cn string, ft string) (rate btclists.Rate, err error) {
		require.Equal(t, coin, cn)
		require.Equal(t, fiat, ft)

		rate = someRate
		return
	}

	var httpFunc = phttp.GetLatest(rates, fiat, coin)

	var response = httptest.NewRecorder()
	var request = httptest.NewRequest("GET", "/latest", nil)

	httpFunc(response, request)

	require.NotEqual(t, 0, response.Body.Len())
	require.Equal(t, http.StatusOK, response.Code)

	var rateResponse phttp.RateResponse
	var err = json.NewDecoder(response.Body).Decode(&rateResponse)
	require.Nil(t, err)
	require.Equal(t, someRate.Rate, rateResponse.Data)
}

func TestAtHandlerSuccess(t *testing.T) {
	var rates = new(RateServerMock)
	rates.AtFunc = func(ctx context.Context, cn string, ft string, when time.Time) (rate btclists.Rate, err error) {
		require.Equal(t, coin, cn)
		require.Equal(t, fiat, ft)
		require.Equal(t, when.Format(btclists.DateTimeFormat), someTime.Format(btclists.DateTimeFormat))

		rate = someRate
		err = nil
		return
	}

	var httpFunc = phttp.GetLatestAt(rates, fiat, coin)

	var response = httptest.NewRecorder()

	var values = url.Values{}
	values.Add("t", someTime.Format(btclists.DateTimeFormat))

	var request = httptest.NewRequest(
		"GET",
		fmt.Sprintf("/at?%s", values.Encode()),
		nil,
	)

	httpFunc(response, request)

	require.NotEqual(t, 0, response.Body.Len())
	require.Equal(t, http.StatusOK, response.Code)

	var rateResponse phttp.RateResponse
	var err = json.NewDecoder(response.Body).Decode(&rateResponse)
	require.Nil(t, err)
	require.Equal(t, someRate.Rate, rateResponse.Data)
}

func TestAtHandlerValidation(t *testing.T) {
	var rates = new(RateServerMock)
	rates.AtFunc = func(ctx context.Context, cn string, ft string, when time.Time) (rate btclists.Rate, err error) {
		rate = someRate
		return
	}
	var httpFunc = phttp.GetLatestAt(rates, fiat, coin)

	var table = []struct {
		t      string
		status int
	}{
		{
			t:      time.Now().Format(btclists.DateTimeFormat),
			status: http.StatusOK,
		},
		{
			t:      time.Now().Format(time.RFC3339),
			status: http.StatusOK,
		},
		{
			t:      "2013-43-232T34:32",
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.ANSIC),
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.Stamp),
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.RubyDate),
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.RFC3339Nano),
			status: http.StatusOK,
		},
		{
			t:      time.Now().Format(time.RFC822Z),
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.RFC822),
			status: http.StatusBadRequest,
		},
		{
			t:      time.Now().Format(time.Kitchen),
			status: http.StatusBadRequest,
		},
	}

	var values = url.Values{}

	for index, test := range table {
		values.Set("t", test.t)

		var response = httptest.NewRecorder()
		var request = httptest.NewRequest(
			"GET",
			fmt.Sprintf("/at?%s", values.Encode()),
			nil,
		)

		httpFunc(response, request)
		require.Equal(t, test.status, response.Code, "Failed at %d", index)
	}
}

func TestAverageHandlerValidation(t *testing.T) {
	var rates = new(RateServerMock)
	rates.RangeFunc = func(ctx context.Context, cn string, ft string, from time.Time, to time.Time) (rates []btclists.Rate, err error) {
		return []btclists.Rate{someRate}, nil
	}

	var httpFunc = phttp.GetAverageFor(rates, fiat, coin)

	var now = time.Now()
	var table = []struct {
		status int
		from   string
		to     string
	}{
		{
			status: http.StatusOK,
			from:   now.Format(btclists.DateTimeFormat),
			to:     now.Add(time.Minute * 1).Format(btclists.DateTimeFormat),
		},
		{
			status: http.StatusOK,
			from:   now.Format(btclists.DateTimeFormat),
			to:     now.Format(btclists.DateTimeFormat),
		},
		{
			status: http.StatusBadRequest,
			from:   now.Format(btclists.DateTimeFormat),
			to:     now.Add(time.Minute * 1).Format(time.ANSIC),
		},
		{
			status: http.StatusBadRequest,
			from:   now.Format(time.RFC822Z),
			to:     now.Add(time.Minute * 1).Format(time.RFC3339),
		},
		{
			status: http.StatusBadRequest,
			from:   now.Format(btclists.DateTimeFormat),
			to:     now.Add(time.Minute * 1).Format(time.ANSIC),
		},
	}

	var values = url.Values{}
	for index, test := range table {
		values.Set("to", test.to)
		values.Set("from", test.from)

		var response = httptest.NewRecorder()
		var request = httptest.NewRequest(
			"GET",
			fmt.Sprintf("/range?%s", values.Encode()),
			nil,
		)

		httpFunc(response, request)

		require.Equal(t, test.status, response.Code, "Failed for test %d", index)
	}
}
