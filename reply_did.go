package vvvot

import (
	"context"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/vvakame/atproto-vvvot/retdid"
	"golang.org/x/exp/slog"
)

type ResponseReplyDID struct {
}

func (reply *ResponseReplyDID) isResponse() {}

func isReplyDIDRequest(ctx context.Context, me *xrpc.AuthInfo, feedPost *bsky.FeedPost) bool {
	s := feedPost.Text
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@"+me.Handle)
	s = strings.TrimSpace(s)

	return s == "did"
}

func replyDID(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification) (Response, error) {
	ret, err := retdid.ReplyUserDID(ctx, xrpcc, nf)
	if err != nil {
		slog.ErrorCtx(ctx, "error on retdid.ReplyUserDID", "error", err)
		return nil, err
	}
	slog.InfoCtx(ctx, "respond did message", "ret", ret)

	resp := &ResponseReplyDID{}

	return resp, nil
}
