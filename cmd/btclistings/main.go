package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/influx6/btclists/pkg"
)

const (
	CryptoCoin   = "BTC"
	FiatCurrency = "USD"
)

var (
	PORT           = os.Getenv("PORT")
	HOST           = os.Getenv("HOST")
	COIN_API_TOKEN = os.Getenv("COIN_API_TOKEN")
	DATABASE_URL   = os.Getenv("DATABASE_URL")

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
)
var (
	signals = []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGABRT,
	}
)

type loggingClient struct{}

func (c loggingClient) Do(req *http.Request) (*http.Response, error) {
	log.Printf("[BTC Listings] | [COIN API] | %s\n", req.URL.String())

	var res, err = httpClient.Do(req)
	if err != nil {
		log.Printf("[BTC Listings] | [COIN API] | %s | %d\n", req.URL.String(), res.StatusCode)
	}
	return res, err
}

func main() {
	var stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, signals...)

	var ctx, ctxCancelFunc = context.WithCancel(context.Background())

	// listen for close signal and cancel root context.
	go func() {
		<-stopChan
		ctxCancelFunc()
		log.Println("[BTC Listings] | received closed signal")
	}()

	var db, err = pkg.NewPostgresDBFromURL(DATABASE_URL, "ratings")
	if err != nil {
		log.Fatalf("[BTC Listings] | Failed to create database from url: %s", err)
		return
	}

	if pingErr := db.DB().PingContext(ctx); pingErr != nil {
		log.Fatalf("[BTC Listings] | Failed to verify database connection: %s", pingErr)
		return
	}

	// setup api service implementation
	var coinAPI = pkg.NewCoinAPI(pkg.CoinApiProdURL, COIN_API_TOKEN, &loggingClient{})

	var ratingService = pkg.NewCoinRatingService(ctx, db, coinAPI)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/latest", pkg.GetLatest(ratingService, FiatCurrency, CryptoCoin))
	router.Get("/at", pkg.GetLatestAt(ratingService, FiatCurrency, CryptoCoin))
	router.Get("/avg", pkg.GetAverageFor(ratingService, ratingService, FiatCurrency, CryptoCoin))

	var addr = fmt.Sprintf("%s:%s", HOST, PORT)
	var server = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	var waiter sync.WaitGroup
	waiter.Add(2)

	// Start routing for periodic updates
	go func() {
		defer waiter.Done()
		defer log.Println("[BTC Listings] | periodic rating update routine stopped")

		log.Println("[BTC Listings] | Starting periodic rating update routine")
		pkg.PeriodicRatingUpdate(ctx, db, coinAPI, CryptoCoin, FiatCurrency)

		defer log.Println("[BTC Listings] | stopping periodic rating update routine")
	}()

	// listen for closed signal to closer server
	go func() {
		defer waiter.Done()
		<-ctx.Done()

		// shut server down in 1 minute.
		var wait5, _ = context.WithTimeout(ctx, time.Minute*1)
		if err := server.Shutdown(wait5); err != nil {
			log.Printf("[BTC Listings] | Server shutdown had issues | %s\n", err)
			return
		}
		log.Println("[BTC Listings] | Server successfully shutdown")
	}()

	//  boot up http server
	log.Printf("[BTC Listings] | Booting up http server | %s\n", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Println("[BTC Listings] | Server shutting down, if you did this, I will find you... :)")
	}

	// ensure all go-routines are clean-ed out.
	waiter.Wait()
}
