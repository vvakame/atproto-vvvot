package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/vvakame/atproto-vvvot/httpapi"
	"github.com/vvakame/atproto-vvvot/internal/cliutils"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	xrpcc := &xrpc.Client{
		Client: http.DefaultClient,
		Host:   "https://bsky.social",
	}

	auth, err := cliutils.LoadAuthInfo(ctx, xrpcc)
	if err != nil {
		slog.Error("error on cliutils.LoadAuthInfo", "error", err)
		panic(err)
	}

	xrpcc.Auth = auth

	mux := http.NewServeMux()

	h, err := httpapi.New(xrpcc)
	if err != nil {
		slog.Error("error on httpapi.New", "error", err)
		panic(err)
	}

	h.Serve(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error received from srv.ListenAndServe", "error", err)
			panic(err)
		}
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		slog.Error("error received from srv.Shutdown", "error", err)
		panic(err)
	}

	slog.Info("server shutdown properly")
}
