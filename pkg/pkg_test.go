package pkg_test

import (
	"bytes"
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

type ClosingBuffer struct {
	*bytes.Buffer
}

func (b *ClosingBuffer) Close() error {
	return nil
}

var (
	someTime               = time.Now()
	someTimeLater          = someTime.Add(time.Hour * 3600)
	someTimeFormatted      = someTime.Format(btclists.DateTimeFormat)
	someOtherTimeFormatted = someTimeLater.Format(btclists.DateTimeFormat)
	someRate               = btclists.Rate{
		Rate: decimal.NewFromFloat(43.322),
		Time: someTime,
		Coin: COIN,
		Fiat: FIAT,
	}
)
