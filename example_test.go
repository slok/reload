package reload_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/slok/reload"
)

func ExampleReloader_basic() {
	reloadSvc := reload.NewManager()

	// Add reloader.
	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 1: %s\n", id)
		return nil
	}))

	// Run ticker and add as a reload notifier.
	{
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-t.C
			return "ticker", nil
		}))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 550*time.Millisecond)
	defer cancel()

	_ = reloadSvc.Run(ctx)

	// Output:
	//Reloader 1: ticker
	//Reloader 1: ticker
	//Reloader 1: ticker
	//Reloader 1: ticker
	//Reloader 1: ticker
}

func ExampleReloader_basicHTTP() {
	reloadSvc := reload.NewManager()

	// Add reloader.
	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 1: %s\n", id)
		return nil
	}))

	// Run http server and add as a reload notifier.
	serverURL := ""
	{
		reloadC := make(chan string)
		h := http.NewServeMux()
		h.Handle("/-/reload", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			fmt.Println("Triggering HTTP reload...")
			reloadC <- "http"
		}))
		server := httptest.NewServer(h)
		defer server.Close()
		serverURL = server.URL

		reloadSvc.On(reload.NotifierChan(reloadC))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Make a reload HTTP request.
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = http.Get(serverURL + "/-/reload")
	}()

	_ = reloadSvc.Run(ctx)

	// Output:
	//Triggering HTTP reload...
	//Reloader 1: http
}

func ExampleReloader_priority() {
	reloadSvc := reload.NewManager()

	// Add reloaders with different reload priorities.
	reloadSvc.Add(10, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 2: %s\n", id)
		return nil
	}))

	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 1: %s\n", id)
		return nil
	}))

	reloadSvc.Add(20, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 3: %s\n", id)
		return nil
	}))

	// Run ticker and add as a reload notifier.
	{
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-t.C
			return "ticker", nil
		}))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 550*time.Millisecond)
	defer cancel()

	_ = reloadSvc.Run(ctx)

	// Output:
	//Reloader 1: ticker
	//Reloader 2: ticker
	//Reloader 3: ticker
	//Reloader 1: ticker
	//Reloader 2: ticker
	//Reloader 3: ticker
	//Reloader 1: ticker
	//Reloader 2: ticker
	//Reloader 3: ticker
	//Reloader 1: ticker
	//Reloader 2: ticker
	//Reloader 3: ticker
	//Reloader 1: ticker
	//Reloader 2: ticker
	//Reloader 3: ticker

}

func ExampleReloader_multinotifier() {
	reloadSvc := reload.NewManager()

	// Add reloaders.
	reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
		fmt.Printf("Reloader 1: %s\n", id)
		return nil
	}))

	// Run ticker 1 and add as a reload notifier.
	{
		t := time.NewTicker(150 * time.Millisecond)
		defer t.Stop()

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-t.C
			return "ticker1", nil
		}))
	}

	// Run ticker 2 and add as a reload notifier.
	{
		t := time.NewTicker(80 * time.Millisecond)
		defer t.Stop()

		reloadSvc.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-t.C
			return "ticker2", nil
		}))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = reloadSvc.Run(ctx)

	// Output:
	//Reloader 1: ticker2
	//Reloader 1: ticker1
	//Reloader 1: ticker2
	//Reloader 1: ticker2
	//Reloader 1: ticker1
	//Reloader 1: ticker2
	//Reloader 1: ticker2
	//Reloader 1: ticker1
	//Reloader 1: ticker2
}
