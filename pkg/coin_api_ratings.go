package pkg

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/influx6/btclists"
)

var (
	zeroTime = time.Time{}
)

type CoinAPIRatingService struct {
	fiat     string
	coin     string
	tdb      *TimeScaledDB
	ctx      context.Context
	workers  sync.WaitGroup
	exchange *CoinAPI
	doFn     chan func()
}

func NewCoinAPIRatingService(ctx context.Context, db *TimeScaledDB, exchange *CoinAPI, coin string, fiat string, workers int) *CoinAPIRatingService {
	var waiter sync.WaitGroup
	txp := &CoinAPIRatingService{
		fiat:     fiat,
		coin:     coin,
		ctx:      ctx,
		tdb:      db,
		workers:  waiter,
		exchange: exchange,
		doFn:     make(chan func(), 0),
	}

	// spawn single worker for latest
	txp.manageLatestWorker()

	// spawn necessary workers for other requests
	//
	// A better approach is to use a WorkerGroup built specifically
	// to expand and decrease worker count based on load. But for now
	// let's keep it simple
	for i := 0; i < workers; i++ {
		txp.manageFunctionPools()
	}

	return txp
}

func (t *CoinAPIRatingService) Wait() {
	t.workers.Wait()
}

func (t *CoinAPIRatingService) At(coin string, fiat string, ts time.Time) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("not ready")
}

func (t *CoinAPIRatingService) updateDBWithLatestRating() error {
	// retrieve latest ratings pair for current time.
	var latestRating, err = t.exchange.Rate(t.ctx, t.coin, t.fiat, zeroTime)
	if err != nil {
		return err
	}

	// send latest ratings into db.
	return t.tdb.Add(latestRating)
}

func (t *CoinAPIRatingService) manageFunctionPools() {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

		var ticker = time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				// time to end this.
				return
			case fn, ok := <-t.doFn:
				if !ok {
					// should not happen but if it does, end this.
					log.Println("[ALERT] | Unexpected end for CoinAPIRatingService | use the context next time")
					return
				}

				// execute function
				fn()
			}
		}
	}()
}

func (t *CoinAPIRatingService) manageLatestWorker() {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

		var ticker = time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				// time to end this.
				return
			case <-ticker.C:
				// call db update method for latest ratings.
				if err := t.updateDBWithLatestRating(); err != nil {
					log.Printf("[ERROR] | Failed to update latest rating  | %s\n", err)
				}
			}
		}
	}()
}
