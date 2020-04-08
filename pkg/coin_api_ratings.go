package pkg

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/influx6/btclists"
)

var (
	zeroTime                 = time.Time{}
	ErrDBError               = errors.New("db error occurred")
	_          CoinMarketAPI = (*CoinAPI)(nil)
)

// CoinMarketAPI exposes the minimal contract desirable for an exchange service api.
//
// NOTE: Partial need existed because I wanted this user of the API (i.e CoinRatingService) tested on
// specific behaviour. If such tests are not really necessary, then using an interface
// is not necessary and too much may lead to interface poisoning.
//
type CoinMarketAPI interface {
	Rate(ctx context.Context, coin string, fiat string, at time.Time) (btclists.Rate, error)
	RangeFrom(ctx context.Context, coin string, fiat string, from time.Time, limit int) ([]btclists.Rate, error)
	Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time, limit int) ([]btclists.Rate, error)
}

// CoinRatingService implements the btclists.RateService, and periodically
// pulls latest exchange rate from db per minute.
type CoinRatingService struct {
	fiat     string
	coin     string
	exchange CoinMarketAPI
	tdb      btclists.RatesDB
	ctx      context.Context
	workers  sync.WaitGroup
}

func NewCoinRatingService(ctx context.Context, db btclists.RatesDB, exchange CoinMarketAPI, coin string, fiat string) *CoinRatingService {
	var waiter sync.WaitGroup
	txp := &CoinRatingService{
		fiat:     fiat,
		coin:     coin,
		ctx:      ctx,
		tdb:      db,
		workers:  waiter,
		exchange: exchange,
	}

	// spawn single worker for latest
	txp.manageLatestWorker()

	return txp
}

func (t *CoinRatingService) Wait() {
	t.workers.Wait()
}

// Latest implements RateService.Latest method, fulfilling RateService contract.
func (t *CoinRatingService) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var latest, err = t.tdb.Latest(ctx, coin, fiat)

	// if db has no latest ratings data, then fallback quickly to API
	// hopefully db isn't gone rouge or something, and should be ready before
	// next request.
	if err != nil {
		log.Printf("[ERROR] | DB just said no, find out why | %s\n", err)

		if latest, err = t.getAndUpdateWithLatest(ctx); err != nil && err != ErrDBError {
			log.Printf("[ERROR] | Oh, we are in trouble now | %s\n", err)
			return latest, err
		}
		err = nil
	}

	return latest, err
}

// At implements RateService.At method, fulfilling RateService contract.
//
// Function may return retrieved result with error if db insertion failed.
// Handle as you wish.
func (t *CoinRatingService) At(ctx context.Context, coin string, fiat string, ts time.Time) (btclists.Rate, error) {
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
	if dbErr := t.tdb.Add(ctx, ratingFromAPI); dbErr != nil {
		log.Printf("[CRITICAL] | Failed to save rating to db | %s\n", dbErr)

		// Returning rating with DBError error.
		return ratingFromAPI, ErrDBError
	}

	return ratingFromAPI, nil
}

/* Average implements RatingsAverageServe interface.
*
* If we can't find suitable records in db, then API will be used, else we pull
* available records in DB, even if a bit lagging behind latest from API.
*
* Note: Average will be returned if it was pulled from the API even if an error occurred
* whilst saving retrieved ratings into DB. Caller should decide on how to proceed.
*
 */
func (t *CoinRatingService) Average(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (decimal.Decimal, error) {
	var average decimal.Decimal

	// Do we have have any records for this range ?
	var total, terr = t.tdb.CountForRange(ctx, coin, fiat, from, to)
	if terr != nil {
		// fail fast. It could be many things, but we won't mitigate
		// these here, let call fail and force new call by caller.
		return average, terr
	}

	// Pull from API and calculate average
	if total == 0 {
		var results, apiErr = t.exchange.Range(ctx, coin, fiat, from, to, MaxLimit)
		if apiErr != nil {
			log.Printf("[ERROR] | API fails us | %s\n", apiErr)
			return average, apiErr
		}

		for _, result := range results {
			average = average.Add(result.Rate)
		}

		average = average.Div(decimal.NewFromInt(int64(len(results))))

		var dbSaveErr error
		if dbSaveErr = t.tdb.AddBatch(ctx, results); dbSaveErr != nil {
			log.Printf("[CRITICAL] | DB failures are not good | %s\n", dbSaveErr)
		}

		return average, dbSaveErr
	}

	var err error
	average, err = t.tdb.AverageForRange(ctx, coin, fiat, from, to)
	if err != nil {
		log.Printf("[ERROR] | Failed to retreive average | %s\n", err)
	}
	return average, err
}

/* Range implements RateService.Range method, fulfilling RateService contract.
*
*  We are enforcing certain rules to govern how range should work:
*
*  1. If time range is not in db, this will occur if time range is too far back
*	or in the future, if so then we pull from API and store new info, serving
*	API results.
*
*  2. If time range is in db, even if not totally accurate to existing records in API
*	we will serve those and allow our service to catch up.
*
* Note: Results may be returned if it was pulled from the API and the returned error was
* possibly a DB insert failure. Caller should decide on how to proceed.
*
* */
func (t *CoinRatingService) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	var results []btclists.Rate

	var total, terr = t.tdb.CountForRange(ctx, coin, fiat, from, to)
	if terr != nil {
		// fail fast. It could be many things, but we won't mitigate
		// these here, let call fail and force new call by caller.
		return results, terr
	}

	// if we have no records for said time range, then pull directly from API
	// and serve that as results after saving.
	if total == 0 {
		var apiErr error
		results, apiErr = t.exchange.Range(ctx, coin, fiat, from, to, MaxLimit)
		if apiErr != nil {
			log.Printf("[ERROR] | API fails us | %s\n", apiErr)
			return results, apiErr
		}

		if dbSaveErr := t.tdb.AddBatch(ctx, results); dbSaveErr != nil {
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

// getAndUpdateWithLatest fetches current coin-fiat rating from API, it
// may return fetched ratings with error if db, fails to insert ratings
// successfully.
func (t *CoinRatingService) getAndUpdateWithLatest(ctx context.Context) (btclists.Rate, error) {
	// retrieve latest ratings pair for current time.
	var latestRating, err = t.exchange.Rate(ctx, t.coin, t.fiat, zeroTime)
	if err != nil {
		log.Printf("[ERROR] | Failed to update latest rating  | %s\n", err)
		return btclists.Rate{}, nil
	}

	// send latest ratings into db.
	if dbErr := t.tdb.Add(ctx, latestRating); dbErr != nil {
		log.Printf("[CRITICAL] | Bad News, Failed to update db | %s\n", dbErr)
		err = ErrDBError
	}
	return latestRating, err
}

func (t *CoinRatingService) manageLatestWorker() {
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
				if _, err := t.getAndUpdateWithLatest(t.ctx); err != nil {
					log.Printf("[CRITICAL] | DB won't agree, we lost this one | %s\n", err)
				}
			}
		}
	}()
}
