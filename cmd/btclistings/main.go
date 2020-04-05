package btclistings

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var (
	PORT            = os.Getenv("PORT")
	HOST            = os.Getenv("HOST")
	COINLAYER_TOKEN = os.Getenv("COINLAYER_TOKEN")

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
)
var (
	stopChan = make(chan os.Signal, 1)
	signals  = []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGABRT,
	}
)

func main() {
	var ctx, ctxCancelFunc = context.WithCancel(context.Background())

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	//router.Get("/last", handlers.)

	var addr = fmt.Sprintf("%s:%s", HOST, PORT)
	var server = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
	}

	signal.Notify(stopChan, signals...)
	go func() {
		<-stopChan
		ctxCancelFunc()
	}()

	go func() {
		<-ctx.Done()

		var wait5, _ = context.WithTimeout(ctx, time.Minute*1)
		if err := server.Shutdown(wait5); err != nil {
			log.Printf("[ALERT] Something bad occured whilst shutting server down,...you know about this don't you?")
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Println("[ALERT] Server shutting down, if you did this, I will find you...")
	}
}
