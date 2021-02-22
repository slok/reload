package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/slok/reload"
)

const (
	reloadKeySIGHUP  = "SIGHUP"
	reloadKeySIGINT  = "SIGINT"
	reloadKeySIGTERM = "SIGTERM"
)

func main() {

	reloadSvc := reload.NewManager()

	// Add reloaders.
	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 1: %s\n", id)
		return nil
	}))

	reloadSvc.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 2: %s\n", id)
		return nil
	}))

	reloadSvc.Add(200, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 3: %s\n", id)
		return nil
	}))

	reloadSvc.Add(300, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 4: %s\n", id)
		return nil
	}))

	reloadSvc.Add(400, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 5: %s\n", id)
		return nil
	}))

	reloadSvc.Add(-100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("reloader 6: %s\n", id)
		return nil
	}))

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

	err := reloadSvc.Run(context.Background())
	if err != nil {
		log.Panic(err.Error())
	}
}
