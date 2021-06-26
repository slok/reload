package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slok/reload"
)

const (
	reloadKeySIGHUP   = "signal-sighup"
	reloadKeySIGINT   = "signal-sigint"
	reloadKeySIGTERM  = "signal-sigterm"
	reloadKeyWebhhok1 = "signal-wh-1"
	reloadKeyWebhhok2 = "signal-wh-2"
)

func main() {

	reloadSvc := reload.NewManager()

	// Add reloaders.
	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 1: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 2: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 3: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 4: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 5: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		// If we receive a SIGINT we will fail we will end with an error.
		if id == reloadKeySIGINT {
			return fmt.Errorf("faking that we can't reload")
		}

		fmt.Printf("reloader 6: %s\n", id)
		return nil
	}))

	// Signal reloader.
	{
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			// Reload on SIGHUP or SIGINT.
			select {
			case <-ctx.Done():
			case sig := <-sigs:
				switch sig {
				case syscall.SIGHUP:
					return reloadKeySIGHUP, nil
				case syscall.SIGINT:
					return reloadKeySIGINT, nil
				case syscall.SIGTERM:
					return reloadKeySIGTERM, nil
				}
			}

			return "", nil
		}))
	}

	// Time ticker reloader.
	{
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			select {
			case <-ctx.Done():
				return "", nil // End execution.
			case tickerT := <-t.C:
				return tickerT.String(), nil
			}
		}))
	}

	// HTTP webhook reloader.
	{
		c := make(chan string)
		router := http.NewServeMux()
		router.Handle("/w1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { c <- reloadKeyWebhhok1 }))
		router.Handle("/w2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { c <- reloadKeyWebhhok2 }))

		addr := ":8080"
		fmt.Printf("webhook '/w1' and '/w2' listening at %s...\n", addr)
		go func() {
			err := http.ListenAndServe(addr, router)
			if err != nil {
				panic(err)
			}
		}()

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			return <-c, nil
		}))
	}

	err := reloadSvc.Run(context.Background())
	if err != nil {
		log.Panic(err.Error())
	}
}
