package vvvot

import (
	"context"
	"fmt"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"golang.org/x/exp/slog"
)

type ReplyAccountCreatedAt struct {
}

func (reply *ReplyAccountCreatedAt) isResponse() {}

func isReplyAccountCreatedAtRequest(ctx context.Context, me *xrpc.AuthInfo, feedPost *bsky.FeedPost) bool {
	s := feedPost.Text
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@"+me.Handle)
	s = strings.TrimSpace(s)

	return s == "birthday"
}

func replyAccountCreatedAt(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification) (Response, error) {
	post, ok := nf.Record.Val.(*bsky.FeedPost)
	if !ok {
		return nil, fmt.Errorf("record type is not 'app.bsky.feed.post'")
	}

	var text string

	if nf.Author.IndexedAt == nil {
		text = "sorry, your indexedAt is null"
	} else {
		at, err := time.Parse(time.RFC3339Nano, *nf.Author.IndexedAt)
		if err != nil {
			return nil, err
		}
		text = fmt.Sprintf(
			`your indexedAt is %s (UTC) / %s (JST)`,
			at.In(time.UTC).Format(time.DateTime),
			at.In(time.FixedZone("Asia/Tokyo", 9*60*60)).Format(time.DateTime),
		)
	}

	post = &bsky.FeedPost{
		Text:      text,
		CreatedAt: time.Now().Local().Format(time.RFC3339),
		Reply: &bsky.FeedPost_ReplyRef{
			Parent: &comatproto.RepoStrongRef{
				Cid: nf.Cid,
				Uri: nf.Uri,
			},
			Root: &comatproto.RepoStrongRef{
				Cid: nf.Cid,
				Uri: nf.Uri,
			},
		},
	}

	recordResp, err := comatproto.RepoCreateRecord(ctx, xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: post,
		},
	})
	if err != nil {
		slog.Error("error on comatproto.RepoCreateRecord", "error", err, "resp", recordResp)
		return nil, err
	}

	slog.InfoCtx(ctx, "message posted", "uri", recordResp.Uri, "cid", recordResp.Cid)

	resp := &ReplyAccountCreatedAt{}

	return resp, nil
}
