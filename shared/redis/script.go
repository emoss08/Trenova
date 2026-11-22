package redis

import (
	"context"
	"crypto/sha1" //nolint:gosec // EVALSHA names a script by its SHA-1 digest
	"encoding/hex"
)

type Scripter interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd
}

type Script struct {
	src  string
	hash string
}

func NewScript(src string) *Script {
	sum := sha1.Sum([]byte(src)) //nolint:gosec // EVALSHA names a script by its SHA-1 digest
	return &Script{src: src, hash: hex.EncodeToString(sum[:])}
}

func (s *Script) Run(ctx context.Context, c Scripter, keys []string, args ...any) *Cmd {
	cmd := c.EvalSha(ctx, s.hash, keys, args...)
	if hasErrorPrefix(cmd.Err(), "NOSCRIPT") {
		return c.Eval(ctx, s.src, keys, args...)
	}
	return cmd
}
