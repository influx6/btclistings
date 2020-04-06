package pkg

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/influx6/btclists"
)

type CoinAPIRatingService struct {
	signal   context.Context
	conn     *TimeScaledDB
	workers  sync.WaitGroup
	exchange *CoinAPI
	doFn     chan func()
}

func NewCoinAPIRatingService(ctx context.Context, conn *TimeScaledDB, exchange *CoinAPI) *CoinAPIRatingService {
	var workers sync.WaitGroup
	txp := &CoinAPIRatingService{
		signal:   ctx,
		conn:     conn,
		workers:  workers,
		exchange: exchange,
		doFn:     make(chan func(), 0),
	}

	txp.manageRequests()
	return txp
}

func (t *CoinAPIRatingService) Wait() {
	t.workers.Wait()
}

func (t *CoinAPIRatingService) At(coin string, fiat string, ts time.Time) (btclists.Rate, error) {
	return btclists.Rate{}, errors.New("not ready")
}

func (t *CoinAPIRatingService) updateDBWithLatestRating() {

}

func (t *CoinAPIRatingService) manageRequests() {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

		var ticker = time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-t.signal.Done():
				// time to end this.
				return
			case <-ticker.C:
				// call db update method for latest ratings.
				t.updateDBWithLatestRating()
			case fn, ok := <-t.doFn:
				if !ok {
					return
				}

				// execute function
				fn()
			}
		}
	}()
}
