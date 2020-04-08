package pkg_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influx6/btclists/pkg"
	"github.com/shopspring/decimal"

	"github.com/influx6/btclists"
	"github.com/stretchr/testify/mock"
)

const (
	defaultRating = 34.531
)

// AverageFor returns
var (
	_ btclists.RatesDB = (*MockRateDB)(nil)
)

type MockRateDB struct {
	mock.Mock
}

func (m *MockRateDB) CountForRange(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (int, error) {
	var result = m.Called(coin, fiat, from, to)
	return result.Get(0).(int), result.Error(1)
}

func (m *MockRateDB) Add(ctx context.Context, rating btclists.Rate) error {
	var result = m.Called(rating)
	return result.Error(0)
}

func (m *MockRateDB) AddBatch(ctx context.Context, ratings []btclists.Rate) error {
	var result = m.Called(ratings)
	return result.Error(0)
}

func (m *MockRateDB) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var result = m.Called(coin, fiat)
	return result.Get(0).(btclists.Rate), result.Error(1)
}

func (m *MockRateDB) Oldest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var result = m.Called(coin, fiat)
	return result.Get(0).(btclists.Rate), result.Error(1)
}

func (m *MockRateDB) At(ctx context.Context, coin string, fiat string, from time.Time) (btclists.Rate, error) {
	var result = m.Called(coin, fiat, from)
	return result.Get(0).(btclists.Rate), result.Error(1)
}

func (m *MockRateDB) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	var result = m.Called(coin, fiat, from, to)
	return result.Get(0).([]btclists.Rate), result.Error(1)
}

func (m *MockRateDB) AverageForRange(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (decimal.Decimal, error) {
	var result = m.Called(coin, fiat, from, to)
	return result.Get(0).(decimal.Decimal), result.Error(1)
}

type MockCoinMarket struct {
	RateFunc      func(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error)
	RangeFromFunc func(ctx context.Context, coin string, fiat string, from time.Time, limit int) ([]btclists.Rate, error)
	RangeFunc     func(ctx context.Context, coin string, fiat string, from time.Time, to time.Time, limit int) ([]btclists.Rate, error)
}

func (c *MockCoinMarket) Rate(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error) {
	return c.RateFunc(ctx, coin, fiat, at)
}

func (c *MockCoinMarket) RangeFrom(ctx context.Context, coin string, fiat string, from time.Time, limit int) ([]btclists.Rate, error) {
	return c.RangeFromFunc(ctx, coin, fiat, from, limit)
}

func (c *MockCoinMarket) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time, limit int) ([]btclists.Rate, error) {
	return c.RangeFunc(ctx, coin, fiat, from, to, limit)
}

func createResponse(statusCode int) (*http.Response, error) {
	var data, err = json.Marshal(pkg.ExchangeRate{
		Time:         someTime,
		AssetIdBase:  COIN,
		AssetIdQuote: FIAT,
		Rate:         decimal.NewFromFloat(10.343),
	})
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       &ClosingBuffer{bytes.NewBuffer(data)},
	}, nil
}

func TestNewCoinRatingService_Latest_ToAPI(t *testing.T) {
	var db = new(MockRateDB)
	var market = new(MockCoinMarket)

	var calledAPI = false
	market.RateFunc = func(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error) {
		calledAPI = true
		return someRate, nil
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var service = pkg.NewCoinRatingService(ctx, db, market, COIN, FIAT)

	db.On("Add", someRate).Return(nil)
	db.On("Latest", COIN, FIAT).Return(btclists.Rate{}, errors.New("not in db"))

	var result, resErr = service.Latest(context.Background(), COIN, FIAT)
	require.NoError(t, resErr)
	require.Equal(t, someRate, result)

	require.True(t, calledAPI)
	db.AssertExpectations(t)

	canceler()
	service.Wait()
}

func TestNewCoinRatingService_Latest_ToDB(t *testing.T) {
	var db = new(MockRateDB)
	var market = new(MockCoinMarket)

	var calledAPI = false
	market.RateFunc = func(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error) {
		calledAPI = true
		return someRate, nil
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var service = pkg.NewCoinRatingService(ctx, db, market, COIN, FIAT)

	db.On("Latest", COIN, FIAT).Return(someRate, nil)

	var result, resErr = service.Latest(context.Background(), COIN, FIAT)
	require.NoError(t, resErr)
	require.Equal(t, someRate, result)

	require.False(t, calledAPI)
	db.AssertExpectations(t)

	canceler()
	service.Wait()
}
