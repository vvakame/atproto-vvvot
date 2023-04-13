package vvvot

import (
	"context"
	"fmt"
	"strings"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

type ReplyAccountCreatedAt struct {
	Base           *bsky.NotificationListNotifications_Notification
	Input          *comatproto.RepoCreateRecord_Input
	Output         *comatproto.RepoCreateRecord_Output
	FoundIndexedAt bool
}

func (reply *ReplyAccountCreatedAt) isResponse() {}

func isReplyAccountCreatedAtRequest(ctx context.Context, me *xrpc.AuthInfo, feedPost *bsky.FeedPost) bool {
	s := feedPost.Text
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@"+me.Handle)
	s = strings.TrimSpace(s)

	return s == "birthday"
}

func replyAccountCreatedAt(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification) (_ Response, err error) {
	ctx, span := otel.Tracer("vvvot").Start(ctx, "replyAccountCreatedAt")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	var foundIndexedAt bool
	var text string
	if nf.Author.IndexedAt != nil && *nf.Author.IndexedAt != "" {
		foundIndexedAt = true
		at, err := time.Parse(time.RFC3339Nano, *nf.Author.IndexedAt)
		if err != nil {
			return nil, err
		}
		text = fmt.Sprintf(
			`your indexedAt is %s (UTC) / %s (JST)`,
			at.In(time.UTC).Format(time.DateTime),
			at.In(time.FixedZone("Asia/Tokyo", 9*60*60)).Format(time.DateTime),
		)
	} else {
		text = "sorry, your indexedAt is null"
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
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
			},
		},
	}

	output, err := comatproto.RepoCreateRecord(ctx, xrpcc, input)
	if err != nil {
		slog.Error("error raised by com.atproto.repo.createRecord", "error", err, "output", output)
		return nil, err
	}

	slog.InfoCtx(ctx, "message posted", "uri", output.Uri, "cid", output.Cid)

	resp := &ReplyAccountCreatedAt{
		Base:           nf,
		Input:          input,
		Output:         output,
		FoundIndexedAt: foundIndexedAt,
	}

	return resp, nil
}
