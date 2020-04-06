package pkg

import (
	"context"
	"errors"
	"time"

	"github.com/influx6/btclists"
)

type TimeScaledDB struct{}

func (t *TimeScaledDB) Add(rate btclists.Rate) error {
	return errors.New("failed to add this")
}

func (t *TimeScaledDB) AddBatch(rate []btclists.Rate) error {
	if len(rate) == 0 {
		return nil
	}

	return errors.New("failed to add this")
}

func (t *TimeScaledDB) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("failed to add this")
}

func (t *TimeScaledDB) Oldest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("failed to add this")
}

func (t *TimeScaledDB) At(ctx context.Context, coin string, fiat string, time time.Time) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("failed to add this")
}

func (t *TimeScaledDB) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	return nil, errors.New("failed to add this")
}
