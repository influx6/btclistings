package pkg

import (
	"errors"

	"github.com/influx6/btclists"
)

type TimeScaledDB struct{}

func (t *TimeScaledDB) Add(rate btclists.Rate) error {
	return errors.New("failed to add this")
}
