package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/xrpc"
	vvvot "github.com/vvakame/atproto-vvvot"
	"github.com/vvakame/atproto-vvvot/internal/cliutils"
	"golang.org/x/exp/slog"
)

func New(xrpcc *xrpc.Client) (*Handler, error) {
	if xrpcc.Auth == nil {
		return nil, errors.New("xrpc client doesn't have auth info")
	}

	return &Handler{xrpcc: xrpcc}, nil
}

type Handler struct {
	xrpcc *xrpc.Client
}

func (h *Handler) Serve(mux *http.ServeMux) {
	mux.HandleFunc("/api/processNotifications", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := cliutils.CheckTokenExpired(ctx, h.xrpcc)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.ErrorCtx(ctx, "error on cliutils.CheckTokenExpired", "error", err)
			return
		}

		err = h.ProcessNotifications(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.ErrorCtx(ctx, "failed to process notifications", "error", err)
			return
		}
	})
}

func (h *Handler) ProcessNotifications(ctx context.Context) error {
	respList, err := vvvot.ProcessNotifications(ctx, h.xrpcc)
	if err != nil {
		return err
	}

	slog.InfoCtx(ctx, "processed message", "count", len(respList))

	return nil
}
