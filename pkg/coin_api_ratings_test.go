package pkg_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/influx6/btclists"
	"github.com/stretchr/testify/mock"
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

func (m *MockRateDB) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (btclists.Rate, error) {
	var result = m.Called(coin, fiat, from, to)
	return result.Get(0).(btclists.Rate), result.Error(1)
}

func TestNewCoinAPIRatingService(t *testing.T) {
	var db = new(MockRateDB)
	var client = MockClient{
		DoFunc: func(req *http.Request) (response *http.Response, err error) {

		},
	}
}
