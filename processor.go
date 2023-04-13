package vvvot

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

type Response interface {
	isResponse()
}

func ProcessNotifications(ctx context.Context, xrpcc *xrpc.Client) (_ []Response, err error) {
	ctx, span := otel.Tracer("vvvot").Start(ctx, "ProcessNotifications")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	now := time.Now()

	unreadResp, err := bsky.NotificationGetUnreadCount(ctx, xrpcc)
	if err != nil {
		slog.ErrorCtx(ctx, "error raised by app.bsky.notification.getUnreadCount", "error", err)
		return nil, err
	}

	slog.DebugCtx(ctx, "check unread count", "count", unreadResp.Count)

	respList := make([]Response, 0)
	limit := int64(20)
	var cursor string
OUTER:
	for {
		resp, err := bsky.NotificationListNotifications(ctx, xrpcc, cursor, limit)
		if err != nil {
			slog.ErrorCtx(ctx, "error raised by app.bsky.notification.listNotifications", "error", err)
			return nil, err
		}

		slog.DebugCtx(ctx, "response about app.bsky.notification.listNotifications", "cursor", resp.Cursor, "length", len(resp.Notifications))

		for idx, nf := range resp.Notifications {
			slog.DebugCtx(
				ctx,
				"notification",
				"index", idx,
				"reason", nf.Reason,
				"author", nf.Author.Handle,
				"cid", nf.Cid,
				"isRead", nf.IsRead,
			)

			switch v := nf.Record.Val.(type) {
			case *bsky.FeedPost:
				slog.DebugCtx(ctx, "feed post", "author", nf.Author.Did, "text", v.Text)

				if !isMentionedToMe(ctx, xrpcc.Auth, v) {
					slog.DebugCtx(ctx, "this post doesn't mentioned to me")
					continue
				}

				threadResp, err := bsky.FeedGetPostThread(ctx, xrpcc, 10, nf.Uri)
				if err != nil {
					slog.Error("error raised by app.bsky.feed.getPostThread", "error", err)
					return nil, err
				}

				if isRepliedAlready(ctx, xrpcc.Auth, threadResp) {
					slog.DebugCtx(ctx, "found newest replied post", "cid", nf.Cid)
					break OUTER
				}

				switch {
				case isReplyDIDRequest(ctx, xrpcc.Auth, v):
					resp, err := replyDID(ctx, xrpcc, nf)
					if err != nil {
						return nil, err
					}

					respList = append(respList, resp)

				case isReplyAccountCreatedAtRequest(ctx, xrpcc.Auth, v):
					resp, err := replyAccountCreatedAt(ctx, xrpcc, nf)
					if err != nil {
						return nil, err
					}

					respList = append(respList, resp)

				default:
					slog.DebugCtx(ctx, "nothing to find keyword", "text", v.Text)
				}

			case *bsky.FeedRepost:
				slog.DebugCtx(ctx, "feed repost", "subjectCid", v.Subject.Cid, "subjectUri", v.Subject.Uri)
			case *bsky.FeedLike:
				slog.DebugCtx(ctx, "feed like", "subjectCid", v.Subject.Cid, "subjectUri", v.Subject.Uri)
			case *bsky.GraphFollow:
				slog.DebugCtx(ctx, "graph follow", "subject", v.Subject)
			default:
				slog.WarnCtx(ctx, "unknown record type", "type", fmt.Sprintf("%T", v))
			}
		}

		if resp.Cursor != nil && *resp.Cursor != "" {
			cursor = *resp.Cursor
			continue
		}

		break
	}

	slog.InfoCtx(ctx, "reply count", "count", len(respList))

	if unreadResp.Count != 0 {
		err = bsky.NotificationUpdateSeen(ctx, xrpcc, &bsky.NotificationUpdateSeen_Input{
			SeenAt: now.Format(time.RFC3339Nano),
		})
		if err != nil {
			slog.ErrorCtx(ctx, "error raised by app.bsky.notification.updateSeen", "error", err)
			return nil, err
		}

		slog.DebugCtx(ctx, "update notification seen", "now", now)
	}

	return respList, nil
}
