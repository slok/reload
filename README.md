# reload

[![GoDoc](https://godoc.org/github.com/slok/reload?status.svg)](https://godoc.org/github.com/slok/reload)
[![CI](https://github.com/slok/reload/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/reload/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/reload)](https://goreportcard.com/report/github.com/slok/reload)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/reload/master/LICENSE)

`reload` is a universal mechanism to reload components in a Go application. Inspired by [oklog/run] and its simplicity, `reload` is a small Go library that has a simple API where `Notifiers` will trigger the reload process on the `Reloaders`.

The mechanism is based on a reload manager that can have multiple notifiers. These notifiers will be executed and the manager will wait until one of those ends the execution. In that moment it will start the reload process. On the other hand the reload manager can have multiple reloaders, that will be triggered when the reload process starts. This process will be running in this manner forever until the reload manager execution is stopped by an error or end in the context passed on run.

When adding the reloaders to the manager, they are added with a priority integer, these reloaders will be grouped in batches of the same priority. When the reload process is started, the manager will execute each reloaders batch sequentially and in priority order (the reloaders inside the batch will be executed concurrently).

The manager as a security mechanism, is smart enough to ignore a reload trigger if there is already a reloading process being executed in that same moment.

## Status

Is in alpha stage, being tested.

## Getting started

```golang
func main() {
    // Setup reloaders.
    reloadSvc := reload.NewManager()

    reloadSvc.Add(0, reload.ReloaderFunc(func(ctx context.Context, id string) error {
        fmt.Printf("reloader 1: %s\n", id)
        return nil
    }))

    reloadSvc.Add(100, reload.ReloaderFunc(func(ctx context.Context, id string) error {
        fmt.Printf("reloader 2: %s\n", id)
        return nil
    }))


    // Setup notifiers.
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
```

## Examples

Check [examples](_examples/).

[oklog/run]: https://github.com/oklog/run
