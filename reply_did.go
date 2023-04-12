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
	"golang.org/x/exp/slog"
)

type ResponseReplyDID struct {
	Base   *bsky.NotificationListNotifications_Notification
	Input  *comatproto.RepoCreateRecord_Input
	Output *comatproto.RepoCreateRecord_Output
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
	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				Text:      fmt.Sprintf(`Hi, @%s ! your DID is "%s"`, nf.Author.Handle, nf.Author.Did),
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
		slog.Error("error raised by com.atproto.repo.createRecord", "error", err)
		return nil, err
	}

	slog.InfoCtx(ctx, "message posted", "uri", output.Uri, "cid", output.Cid)

	resp := &ResponseReplyDID{
		Base:   nf,
		Input:  input,
		Output: output,
	}

	return resp, nil
}
