package pkg_test

import (
	"time"

	"github.com/influx6/btclists"
	"github.com/shopspring/decimal"
)

const (
	APIURI   = "http://say-what-api"
	APIToken = "a-wee-little-token"
	COIN     = "BTC"
	FIAT     = "USD"
)

var (
	someTime               = time.Now().UTC()
	someTimeLater          = someTime.Add(time.Hour * 3600).UTC()
	someTimeFormatted      = someTime.Format(btclists.DateTimeFormat)
	someOtherTimeFormatted = someTimeLater.Format(btclists.DateTimeFormat)
	someRate               = btclists.Rate{
		Rate: decimal.NewFromFloat(43.322),
		Date: someTime,
		Coin: COIN,
		Fiat: FIAT,
	}
)
