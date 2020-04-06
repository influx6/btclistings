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
	zeroTime   = time.Time{}
	ErrDBError = errors.New("db error occurred")
)

// CoinAPIRatingService implements the btclists.RateService, and periodically
// pulls latest exchange rate from db per minute.
type CoinAPIRatingService struct {
	fiat     string
	coin     string
	tdb      btclists.RateDB
	ctx      context.Context
	workers  sync.WaitGroup
	exchange *CoinAPI
	doFn     chan func()
}

func NewCoinAPIRatingService(ctx context.Context, db btclists.RateDB, exchange *CoinAPI, coin string, fiat string) *CoinAPIRatingService {
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

	return txp
}

func (t *CoinAPIRatingService) Wait() {
	t.workers.Wait()
}

// Latest implements RateService.Latest method, fulfilling RateService contract.
func (t *CoinAPIRatingService) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var latest, err = t.tdb.Latest(ctx, coin, fiat)
	// if db has no latest ratings data, then fallback quickly to API
	// hopefully db isn't gone rouge or something, and should be ready before
	// next request.
	if err != nil {
		log.Printf("[ERROR] | DB just said no, find out why | %s\n", err)

		if latest, err = t.updateDBWithLatestRating(ctx); err != nil && err != ErrDBError {
			log.Printf("[ERROR] | Oh, we are in trouble now | %s\n", err)
			return latest, err
		}
		err = nil
	}

	return latest, err
}

// At implements RateService.At method, fulfilling RateService contract.
func (t *CoinAPIRatingService) At(ctx context.Context, coin string, fiat string, ts time.Time) (btclists.Rate, error) {
	var ratingForTime, err = t.tdb.At(ctx, coin, fiat, ts)
	if err == nil {
		return ratingForTime, nil
	}

	// DB seems to be lacking such information, hence lets fallback to using API
	// There are two approaches here, each has it's faults:
	// 1. If API supports it, just fetch the specific ratings for required timestamp, but
	//	this means, if more requests like this are coming in within timestamp range before or
	//	after this, we will use up precious request credits
	//
	// 2. If we want to be anticipatory, fetch records say 1hr apart on both ends with desired timestamp in the middle,
	//   this way we mitigate future trips to API (using up precious credits or limits) for possible time ranges
	//	 within this window. But this also needs to be done with consideration to our exchange rate data hold policy.
	//
	// We will keep it simple for now, so option 1.
	var ratingFromAPI, apiErr = t.exchange.Rate(ctx, coin, fiat, ts)
	if apiErr != nil {
		log.Printf("[ERROR] | API has failed us | %s\n", err)
		return btclists.Rate{}, apiErr
	}

	// Save new rating data to db.
	if dbErr := t.tdb.Add(ratingFromAPI); dbErr != nil {
		log.Printf("[CRITICAL] | Failed to save rating to db | %s\n", err)
		// Do not halt response, just ensure our notification system
		// has notified person in charge of db failure.
		//
		// Worse case, request failed and endpoint would respond accordingly.
	}

	return ratingFromAPI, err
}

/* Range implements RateService.Range method, fulfilling RateService contract.
*
*  We are enforcing certain rules to govern how range should work:
*
*  1. If `from` timestamp is in the future of current latest date (say 1 hour ahead),
*	then consider this a future timestamp and just return an empty list
*
*  2. If `from` is below last available record date known in db, then this is beyond
* 	what we currently have, so shoot directly to API for now (we may later add restrictions to this)
*   to see if its something we can get historically.
*
*  3. If `from` within existing record range, even if value may not necessary be accurate
*	to the last minute, we can provide some level of confidence on the available range we
* 	historically have in db.
*
* Note: Results may be returned if it was pulled from the API and the returned error was
* possibly a DB failure. Caller should decide on how to proceed.
*
* */
func (t *CoinAPIRatingService) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	var results []btclists.Rate

	var latest, lastErr = t.tdb.Latest(ctx, coin, fiat)
	if lastErr != nil {
		// fail fast. I could be many things, but we won't mitigate
		// these here, let call fail and force new call by caller.
		return results, lastErr
	}

	// this is in the future
	if from.After(latest.Time) {
		return results, nil
	}

	var oldest, oldErr = t.tdb.Oldest(ctx, coin, fiat)
	if oldErr != nil {
		// fail fast. I could be many things, but we won't mitigate
		// these here, let call fail and force new call by caller.
		return results, oldErr
	}

	// if we are outside oldest, then pull directly from API
	// and serve that as results after saving batch.
	if from.Before(oldest.Time) {
		var apiErr error
		results, apiErr = t.exchange.Range(ctx, coin, fiat, from, to, MaxLimit)
		if apiErr != nil {
			log.Printf("[ERROR] | API fails us | %s\n", apiErr)
			return results, apiErr
		}

		if dbSaveErr := t.tdb.AddBatch(results); dbSaveErr != nil {
			log.Printf("[CRITICAL] | DB failures are not good | %s\n", dbSaveErr)
			return results, dbSaveErr
		}

		return results, nil
	}

	var err error
	results, err = t.tdb.Range(ctx, coin, fiat, from, to)
	if err != nil {
		log.Printf("[ERROR] | failed to retrieve result | %s\n", err)
	}
	return results, err
}

// updateDBWithLatestRating fetches current coin-fiat rating from API, it
// may return fetched ratings with error if db, fails to insert ratings
// successfully.
func (t *CoinAPIRatingService) updateDBWithLatestRating(ctx context.Context) (btclists.Rate, error) {
	// retrieve latest ratings pair for current time.
	var latestRating, err = t.exchange.Rate(ctx, t.coin, t.fiat, zeroTime)
	if err != nil {
		log.Printf("[ERROR] | Failed to update latest rating  | %s\n", err)
		return btclists.Rate{}, nil
	}

	// send latest ratings into db.
	if dbErr := t.tdb.Add(latestRating); dbErr != nil {
		log.Printf("[CRITICAL] | Bad News, Failed to update db | %s\n", dbErr)
		err = ErrDBError
	}
	return latestRating, err
}

func (t *CoinAPIRatingService) manageLatestWorker() error {
	t.workers.Add(1)
	go func() {
		defer t.workers.Done()

		var ticker = time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		if _, err := t.updateDBWithLatestRating(t.ctx); err != nil {
			log.Printf("[CRITICAL] | Failed initial latest update | %s\n", err)
		}

		for {
			select {
			case <-t.ctx.Done():
				// time to end this.
				return
			case <-ticker.C:
				// call db update method for latest ratings.
				if _, err := t.updateDBWithLatestRating(t.ctx); err != nil {
					log.Printf("[CRITICAL] | DB won't agree, we lost this one | %s\n", err)
				}
			}
		}
	}()

	return nil
}
