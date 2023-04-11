package slogtypes

import "golang.org/x/exp/slog"

type Password string

var _ slog.LogValuer = Password("")

func (p Password) LogValue() slog.Value {
	return slog.StringValue("**censored**")
}
