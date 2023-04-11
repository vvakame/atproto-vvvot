package retdid

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

func ReplyUserDID(ctx context.Context, xrpcc *xrpc.Client, nf *bsky.NotificationListNotifications_Notification) (ok bool, err error) {
	post, ok := nf.Record.Val.(*bsky.FeedPost)
	if !ok {
		return false, fmt.Errorf("record type is not 'app.bsky.feed.post'")
	}

	if !isMentionedToMe(ctx, xrpcc.Auth, post) {
		slog.DebugCtx(ctx, "this post doesn't mentioned to me")
		return false, nil
	}

	threadResp, err := bsky.FeedGetPostThread(ctx, xrpcc, 10, nf.Uri)
	if err != nil {
		slog.Error("error on bsky.FeedGetPostThread", "error", err)
		return false, err
	}

	if isRepliedAlready(ctx, xrpcc.Auth, threadResp) {
		slog.InfoCtx(ctx, "already replied")
		return true, nil
	}

	post = &bsky.FeedPost{
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
		return false, err
	}

	slog.InfoCtx(ctx, "message posted", "uri", recordResp.Uri, "cid", recordResp.Cid)

	return true, nil
}

func isMentionedToMe(ctx context.Context, me *xrpc.AuthInfo, post *bsky.FeedPost) bool {
	if me.Did != "" {
		for _, facet := range post.Facets {
			for _, f := range facet.Features {
				if v := f.RichtextFacet_Mention; v != nil {
					if me.Did == v.Did {
						return true
					}
				}
			}
		}
	}
	if me.Handle != "" {
		if strings.Contains(post.Text, me.Handle) {
			return true
		}
	}

	return false
}

func isRepliedAlready(ctx context.Context, me *xrpc.AuthInfo, thread *bsky.FeedGetPostThread_Output) bool {
	if thread.Thread == nil {
		return false
	}
	if thread.Thread.FeedDefs_ThreadViewPost == nil {
		return false
	}
	for _, reply := range thread.Thread.FeedDefs_ThreadViewPost.Replies {
		if reply.FeedDefs_ThreadViewPost == nil {
			continue
		}
		if reply.FeedDefs_ThreadViewPost.Post.Author.Did == me.Did {
			return true
		}
	}

	return false
}
