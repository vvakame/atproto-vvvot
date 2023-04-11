package main

import (
	"context"
	"os"

	cliutil "github.com/bluesky-social/indigo/cmd/gosky/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/k0kubun/pp/v3"
	vvvot "github.com/vvakame/atproto-vvvot"
	"github.com/vvakame/atproto-vvvot/internal/cliutils"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stderr)))

	xrpcc := &xrpc.Client{
		Client: cliutil.NewHttpClient(),
		Host:   "https://bsky.social",
	}

	auth, err := cliutils.LoadAuthInfo(ctx, xrpcc)
	if err != nil {
		slog.Error("error on cliutils.LoadAuthInfo", "error", err)
		panic(err)
	}

	xrpcc.Auth = auth

	respList, err := vvvot.ProcessNotifications(ctx, xrpcc)
	if err != nil {
		slog.Error("error on vvvot.ProcessNotifications", "error", err)
		panic(err)
	}

	pp.Println(respList)
}
