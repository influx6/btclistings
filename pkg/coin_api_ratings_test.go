package pkg_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/influx6/btclists/pkg"
	"github.com/shopspring/decimal"

	"github.com/influx6/btclists"
	"github.com/stretchr/testify/mock"
)

const (
	defaultRating = 34.531
)

type MockRateDB struct {
	mock.Mock
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

func createResponse() (*http.Response, error) {
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
		StatusCode: http.StatusOK,
		Body:       &ClosingBuffer{bytes.NewBuffer(data)},
	}, nil
}

//func TestNewCoinAPIRatingService_InitialBehaviour(t *testing.T) {
//	var db = new(MockRateDB)
//	var res, err = createResponse()
//	require.Nil(t, err)
//
//	var calledAPI = false
//	var client = new(MockClient)
//	client.DoFunc = func(req *http.Request) (response *http.Response, err error) {
//		calledAPI = true
//		return res, nil
//	}
//
//	var ctx, canceler = context.WithCancel(context.Background())
//	var api = pkg.NewCoinAPI(APIURI, APIToken, client)
//	var service = pkg.NewCoinAPIRatingService(ctx, db, api, COIN, FIAT)
//
//	db.On("Latest", COIN, FIAT, time.Time{}).Return(btclists.Rate{}, nil)
//
//	require.True(t, calledAPI)
//
//	canceler()
//	service.Wait()
//}
