package pkg

import (
	"errors"
	"time"

	"github.com/influx6/btclists"
)

type TimeScaledDB struct{}

func (t *TimeScaledDB) Add(rate btclists.Rate) error {
	return errors.New("failed to add this")
}

func (t *TimeScaledDB) GetLatest(coin string, fiat string) error {
	return errors.New("failed to add this")
}

func (t *TimeScaledDB) GetRatingAt(coin string, fiat string, time time.Time) error {
	return errors.New("failed to add this")
}
