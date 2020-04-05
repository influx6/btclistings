package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/influx6/btclists"
	"github.com/influx6/btclists/pkg/db"
)

type TimeScalePeriodicRatings struct {
	signal   context.Context
	conn     *db.TimeScaledDB
	workers  sync.WaitGroup
	exchange btclists.RateServer
}

func NewTimeScalePeriodicRatings(ctx context.Context, conn *db.TimeScaledDB, exchange btclists.RateServer) *TimeScalePeriodicRatings {
	var workers sync.WaitGroup
	txp := &TimeScalePeriodicRatings{
		signal:   ctx,
		conn:     conn,
		workers:  workers,
		exchange: exchange,
	}

	// boot-up go-routines to manage periodic updates
	// and
	txp.manageUpdates()
	txp.manageRequests()
	return txp
}

func (t *TimeScalePeriodicRatings) GetTime(coin string, fiat string, ts time.Time) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("not ready")
}

// manages requests to be handled for requesting rates from exchange into
// storage.
func (t *TimeScalePeriodicRatings) manageRequests() {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

	}()
}

// manages
func (t *TimeScalePeriodicRatings) manageUpdates() {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

	}()
}
