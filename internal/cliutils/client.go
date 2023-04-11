package cliutils

import (
	"context"
	"fmt"
	"os"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/vvakame/atproto-vvvot/internal/slogtypes"
	"golang.org/x/exp/slog"
)

func getHandle() string {
	return os.Getenv("ATPROTO_BOT_HANDLE")
}

func getPassword() slogtypes.Password {
	return slogtypes.Password(os.Getenv("ATPROTO_BOT_PASSWORD"))
}

func LoadAuthInfo(ctx context.Context, xrpcc *xrpc.Client) (*xrpc.AuthInfo, error) {
	handle := getHandle()
	password := getPassword()

	slog.DebugCtx(ctx, "create session", "handle", handle, "password", password)

	auth, err := comatproto.ServerCreateSession(ctx, xrpcc, &comatproto.ServerCreateSession_Input{
		Identifier: &handle,
		Password:   string(password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &xrpc.AuthInfo{
		AccessJwt:  auth.AccessJwt,
		RefreshJwt: auth.RefreshJwt,
		Handle:     auth.Handle,
		Did:        auth.Did,
	}, nil
}
