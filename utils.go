package vvvot

import (
	"context"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

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
