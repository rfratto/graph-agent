// Command gragent implements an experiemental Prometheus Agent based on graph
// configs, forming the Graph Agent. Graph configs use HCL as the configuration
// langauge and support being dynamically configured at runtime and through the
// expressions found in the configuration file themselves.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"

	"github.com/rfratto/gragent/internal/gragent"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := interruptContext()
	defer cancel()

	var (
		httpListenAddr = ":8080"
		configFile     string
	)

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&httpListenAddr, "server.http-listen-addr", httpListenAddr, "address to listen for http traffic on")
	fs.StringVar(&configFile, "config.file", configFile, "path to config file to load")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	// Validate flags
	if configFile == "" {
		return fmt.Errorf("the -config.file flag is required")
	}

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	s := gragent.NewSystem(l, configFile)

	if err := s.Load(); err != nil {
		return fmt.Errorf("error during the initial gragent load: %w", err)
	}

	// HTTP server
	{
		lis, err := net.Listen("tcp", httpListenAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", httpListenAddr, err)
		}

		r := mux.NewRouter()
		r.Handle("/graph", s.GraphHandler())

		go func() {
			defer cancel()

			level.Info(l).Log("msg", "now listening for http traffic", "addr", httpListenAddr)
			if err := http.Serve(lis, r); err != nil {
				level.Info(l).Log("msg", "http server closed", "err", err)
			}
		}()
	}

	// Gragent
	go func() {
		defer cancel()
		if err := s.Run(ctx); err != nil {
			level.Error(l).Log("msg", "error while running gragent", "err", err)
		}
	}()

	<-ctx.Done()
	return nil
}

type exampleNode struct{ DisplayName string }

func (n exampleNode) Name() string { return n.DisplayName }

func interruptContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		select {
		case <-sig:
		case <-ctx.Done():
		}
		signal.Stop(sig)

		fmt.Fprintln(os.Stderr, "interrupt received")
	}()

	return ctx, cancel
}
