// How to use the app:
// - Configure using `config.json` or other file using `--configuration-path` flag.
// - Use `curl http://127.0.0.1:8080/curl` to make a request to the URL defined on the config.
// - Use `curl http://127.0.0.1:8080/print` to print the message defined on the config.
// - Use `curl http://127.0.0.1:8080/-/stop` shutdowns the app correctly.
// - Use `curl http://127.0.0.1:8080/-/reload` to reload the configuration.
// - Use `kill -1 ${APP_PID}` to reload the configuration.
// - Make changes in the configuration file to reload the configuration.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/oklog/run"
	log "github.com/sirupsen/logrus"
	"github.com/slok/reload"
	"gopkg.in/alecthomas/kingpin.v2"
)

// CmdConfig is the configuration of the command.
type CmdConfig struct {
	File          string
	ListenAddress string
}

// NewCmdConfig returns the application.
func NewCmdConfig(args []string) (*CmdConfig, error) {
	c := CmdConfig{}
	app := kingpin.New("simpleapp", "Example of a simple application with multiple reloaders")
	app.DefaultEnvars()

	// Configure.
	app.Flag("configuration-path", "Configuration file to watch").Short('c').Default("config.json").StringVar(&c.File)
	app.Flag("listen-address", "The address to listen for HTTP requests").Short('l').Default(":8080").StringVar(&c.ListenAddress)

	// Parse.
	_, err := app.Parse(args)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func Run(ctx context.Context) error {
	ctx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	cmdConfig, err := NewCmdConfig(os.Args[1:])
	if err != nil {
		return err
	}

	// Configuration service.
	configSvc, err := NewConfigLoader(cmdConfig.File)
	if err != nil {
		return err
	}

	// Printer service.
	printer, err := NewPrinter(configSvc)
	if err != nil {
		return err
	}

	// Curler service.
	curler, err := NewCurler(configSvc)
	if err != nil {
		return err
	}

	var (
		runGroup      run.Group
		reloadManager = reload.NewManager()
	)

	// Run hot-reload manager.
	{
		// Add all app reloaders in order.
		reloadManager.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			// If configuration fails ignore reload with a warning.
			err := configSvc.Reload(ctx)
			if err != nil {
				log.Warningf("Config could not be reloaded: %s", err)
				return nil
			}

			log.Infof("Config reloaded")
			return nil
		}))

		reloadManager.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			log.Infof("Printer reloaded")
			return printer.Reload(ctx)
		}))

		reloadManager.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			log.Infof("Curler reloaded")
			return curler.Reload(ctx)
		}))

		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				log.Infof("Starting reload manager")
				return reloadManager.Run(ctx)
			},
			func(_ error) {
				log.Infof("Stopping reload manager")
				cancel()
			},
		)
	}

	// Wait for OS signals.
	{
		// Add OS signal reload notifier.
		reloadC := make(chan struct{})
		reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-reloadC
			return "sighup", nil
		}))

		sigC := make(chan os.Signal, 1)
		exitC := make(chan struct{})
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		runGroup.Add(
			func() error {
				log.Infof("Starting OS signal handler")
				for {
					select {
					case s := <-sigC:
						log.Infof("Signal received: %q", s)
						// Don't stop if SIGHUP, only reload.
						if s == syscall.SIGHUP {
							reloadC <- struct{}{}
							continue
						}

						return nil
					case <-exitC:
						return nil
					}
				}
			},
			func(_ error) {
				log.Infof("Stopping OS signal handler")
				close(exitC)
			},
		)
	}

	// Run HTTP server.
	{
		mux := http.NewServeMux()

		// Add HTTP based reload notifier.
		reloadChan := make(chan string)
		reloadManager.On(reload.NotifierChan(reloadChan))
		mux.Handle("/-/reload", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reloadChan <- "http"
		}))

		// Add Service handlers.
		mux.Handle("/print", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			printer.Print()
		}))

		mux.Handle("/curl", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := curler.Curl()
			if err != nil {
				log.Errorf("Curl failed: %s", err)
			}
		}))

		// Add operational stop handler.
		mux.Handle("/-/stop", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rootCancel() // Stop the world!
		}))

		// Add profiling.
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		server := &http.Server{
			Addr:    cmdConfig.ListenAddress,
			Handler: mux,
		}

		runGroup.Add(
			func() error {
				log.Infof("HTTP server listening at %q", cmdConfig.ListenAddress)
				return server.ListenAndServe()
			},
			func(_ error) {
				log.Infof("Stopping HTTP server")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					log.Errorf("Could not shut down http server: %s", err)
				}
			},
		)
	}

	// Run file watcher
	{
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		err = watcher.Add(cmdConfig.File)
		if err != nil {
			return fmt.Errorf("could not add file watcher for %s: %w", cmdConfig.File, err)
		}

		// Add file watcher based reload notifier.
		reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			select {
			case <-watcher.Events:
				return "file-watch", nil
			case err := <-watcher.Errors:
				return "", err
			}
		}))

		ctx, cancel := context.WithCancel(ctx)
		runGroup.Add(
			func() error {
				// Block forever until the watcher stops.
				log.Infof("File watcher with %q config file running", cmdConfig.File)
				<-ctx.Done()
				return nil
			},
			func(_ error) {
				log.Infof("Stopping file watcher")
				watcher.Close()
				cancel()
			},
		)
	}

	return runGroup.Run()
}

func main() {
	ctx := context.Background()
	err := Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}
