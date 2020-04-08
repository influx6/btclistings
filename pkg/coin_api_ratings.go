package pkg

import (
	"context"
	"errors"
	"log"
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

// PeriodicRatingUpdates boots up a loop to periodically pull latest ratings from
// provided exchange service, adding new records to provided db.
func PeriodicRatingUpdate(ctx context.Context, tdb btclists.RatesDB, exchange CoinMarketAPI, coin string, fiat string) {
	var ticker = time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// time to end this.
			return
		case <-ticker.C:
			// call db update method for latest ratings.
			// retrieve latest ratings pair for current time.
			var latestRating, err = exchange.Rate(ctx, coin, fiat, zeroTime)
			if err != nil {
				log.Printf("[BTC Listings] | [ERROR] | Failed to update latest rating  | %s\n", err)
				continue
			}

			// send latest ratings into db.
			if dbErr := tdb.Add(ctx, latestRating); dbErr != nil {
				log.Printf("[BTC Listings] | [CRITICAL] | Bad News, Failed to update db | %s\n", dbErr)
				continue
			}

			log.Printf("[BTC Listings] | [LOG] | updated latest ratings | %s | %s\n", latestRating.Date, latestRating.Rate)
		}
	}
}

//*********************************************
// CoinRatingService
//*********************************************

// CoinRatingService implements the btclists.RateService, and periodically
// pulls latest exchange rate from db per minute.
type CoinRatingService struct {
	exchange CoinMarketAPI
	tdb      btclists.RatesDB
	ctx      context.Context
}

func NewCoinRatingService(ctx context.Context, db btclists.RatesDB, exchange CoinMarketAPI) *CoinRatingService {
	return &CoinRatingService{
		ctx:      ctx,
		tdb:      db,
		exchange: exchange,
	}
}

// Latest implements RateService.Latest method, fulfilling RateService contract.
func (t *CoinRatingService) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var latest, err = t.tdb.Latest(ctx, coin, fiat)
	if err == nil {
		log.Printf("[BTC Listings] | [INFO] | Retreive latest from db | %s | %s\n", latest.Date, latest.Rate)
	}

	// if db has no latest ratings data, then fallback quickly to API
	// hopefully db isn't gone rouge or something, and should be ready before
	// next request.
	if err != nil {
		log.Printf("[BTC Listings] | [ERROR] | DB just said no record, find out why | %s\n", err)

		// retrieve latest ratings pair for current time.
		latest, err = t.exchange.Rate(ctx, coin, fiat, zeroTime)
		if err != nil {
			log.Printf("[BTC Listings] | [ERROR] | Failed to update latest rating  | %s\n", err)
			return btclists.Rate{}, nil
		}

		log.Printf("[BTC Listings] | [INFO] | Retreive latest from API at | %s | %s\n", latest.Date, latest.Rate)

		// send latest ratings into db.
		if dbErr := t.tdb.Add(ctx, latest); dbErr != nil {
			log.Printf("[BTC Listings] | [CRITICAL] | Bad News, Failed to update db | %s\n", dbErr)

			// Return ErrDBError to signal to API we got result but DB insert went a wall
			return latest, ErrDBError
		}
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
		log.Printf("[BTC Listings] | [INFO] | Retreive record from db | %s | %s\n", ratingForTime.Date, ratingForTime.Rate)
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
	// For now, we will keep it simple, so option 1.
	var ratingFromAPI, apiErr = t.exchange.Rate(ctx, coin, fiat, ts)
	if apiErr != nil {
		log.Printf("[BTC Listings] | [ERROR] | API has failed us | %s\n", err)
		return btclists.Rate{}, apiErr
	}

	// Save new rating data to db.
	if dbErr := t.tdb.Add(ctx, ratingFromAPI); dbErr != nil {
		log.Printf("[BTC Listings] | [CRITICAL] | Failed to save rating to db | %s\n", dbErr)

		// Returning rating with DBError error.
		return ratingFromAPI, ErrDBError
	}

	log.Printf("[BTC Listings] | [INFO] | Retreive ratings from API at | %s | %s\n", ratingFromAPI.Date, ratingFromAPI.Rate)
	return ratingFromAPI, nil
}

/* AverageForRange implements RatingsAverageServe interface.
*
* NOTE to reviewer:
* If we can't find suitable records in db, then API will be used, else we pull
* available records in DB, even if lagging behind latest from API.
*
* A possible alternative:
*
* If we wish to ensure consistent accurate response, then we can check if time
* range completely exists in db, by checking `from` against latest and oldest record time,
* if below oldest record time then check pull from API. If above latest record time,
* then pull from API, else if within range, just use db record.
*
*
* Note: Average will be returned if it was pulled from the API even if an error occurred
* whilst saving retrieved ratings into DB. Caller should decide on how to proceed.
*
 */
func (t *CoinRatingService) AverageForRange(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (decimal.Decimal, error) {
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
			log.Printf("[BTC Listings] | [ERROR] | API fails us | %s\n", apiErr)
			return average, apiErr
		}

		if len(results) == 0 {
			log.Println("[BTC Listings] | [INFO] | No records returned from API")
			return average, nil
		}

		for _, result := range results {
			average = average.Add(result.Rate)
		}

		average = average.Div(decimal.NewFromInt(int64(len(results))))

		var dbSaveErr error
		if dbSaveErr = t.tdb.AddBatch(ctx, results); dbSaveErr != nil {
			log.Printf("[BTC Listings] | [CRITICAL] | DB failures are not good | %s\n", dbSaveErr)
		}

		return average, dbSaveErr
	}

	var err error
	average, err = t.tdb.AverageForRange(ctx, coin, fiat, from, to)
	if err != nil {
		log.Printf("[BTC Listings] | [ERROR] | Failed to retreive average | %s\n", err)
	}

	log.Printf("[BTC Listings] | [INFO] | Retreive average | %s | %s | %s\n", from, to, average)
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
			log.Printf("[BTC Listings] | [ERROR] | API fails us | %s\n", apiErr)
			return results, apiErr
		}

		if dbSaveErr := t.tdb.AddBatch(ctx, results); dbSaveErr != nil {
			log.Printf("[BTC Listings] | [CRITICAL] | DB failures are not good | %s\n", dbSaveErr)
			return results, dbSaveErr
		}

		return results, nil
	}

	var err error
	results, err = t.tdb.Range(ctx, coin, fiat, from, to)
	if err != nil {
		log.Printf("[BTC Listings] | [ERROR] | failed to retrieve result | %s\n", err)
	}
	return results, err
}
