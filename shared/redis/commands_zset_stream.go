package redis

import (
	"context"
	"time"
)

type ZRangeArgs struct {
	Key     string
	Start   any
	Stop    any
	ByScore bool
	Rev     bool
}

func (c cmdable) ZAdd(ctx context.Context, key string, members ...Z) *IntCmd {
	return c.zadd(ctx, key, "", members)
}

func (c cmdable) ZAddXX(ctx context.Context, key string, members ...Z) *IntCmd {
	return c.zadd(ctx, key, "xx", members)
}

func (c cmdable) ZAddGT(ctx context.Context, key string, members ...Z) *IntCmd {
	return c.zadd(ctx, key, "gt", members)
}

func (c cmdable) zadd(ctx context.Context, key, flag string, members []Z) *IntCmd {
	args := make([]any, 0, 3+2*len(members))
	args = append(args, "zadd", key)
	if flag != "" {
		args = append(args, flag)
	}
	for _, m := range members {
		args = append(args, m.Score, m.Member)
	}
	return c.integer(ctx, args...)
}

func (c cmdable) ZRem(ctx context.Context, key string, members ...any) *IntCmd {
	return c.integer(ctx, appendValues([]any{"zrem", key}, members)...)
}

func (c cmdable) ZRange(ctx context.Context, key string, start, stop int64) *StringSliceCmd {
	return c.strs(ctx, "zrange", key, start, stop)
}

func (c cmdable) ZRangeArgs(ctx context.Context, z ZRangeArgs) *StringSliceCmd {
	args := []any{"zrange", z.Key, z.Start, z.Stop}
	if z.Rev && z.ByScore {
		args[2], args[3] = z.Stop, z.Start
	}
	if z.ByScore {
		args = append(args, "byscore")
	}
	if z.Rev {
		args = append(args, "rev")
	}
	return c.strs(ctx, args...)
}

func (c cmdable) ZRemRangeByScore(ctx context.Context, key, minScore, maxScore string) *IntCmd {
	return c.integer(ctx, "zremrangebyscore", key, minScore, maxScore)
}

func (c cmdable) ZCard(ctx context.Context, key string) *IntCmd {
	return c.integer(ctx, "zcard", key)
}

func (c cmdable) ZScore(ctx context.Context, key, member string) *FloatCmd {
	cmd := &FloatCmd{baseCmd: baseCmd{args: []any{"zscore", key, member}}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) ZMScore(ctx context.Context, key string, members ...string) *FloatSliceCmd {
	cmd := &FloatSliceCmd{baseCmd: baseCmd{args: appendStrings([]any{"zmscore", key}, members)}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) ZPopMin(ctx context.Context, key string, count int64) *ZSliceCmd {
	cmd := &ZSliceCmd{baseCmd: baseCmd{args: []any{"zpopmin", key, count}}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) XAdd(ctx context.Context, a *XAddArgs) *StringCmd {
	args := []any{"xadd", a.Stream}
	if a.MaxLen > 0 {
		if a.Approx {
			args = append(args, "maxlen", "~", a.MaxLen)
		} else {
			args = append(args, "maxlen", a.MaxLen)
		}
	}
	if a.ID != "" {
		args = append(args, a.ID)
	} else {
		args = append(args, "*")
	}
	return c.str(ctx, appendValues(args, []any{a.Values})...)
}

func (c cmdable) XRead(ctx context.Context, a *XReadArgs) *XStreamSliceCmd {
	args := []any{"xread"}
	if a.Count > 0 {
		args = append(args, "count", a.Count)
	}
	return c.streams(ctx, args, a.Block, a.Streams)
}

func (c cmdable) XReadGroup(ctx context.Context, a *XReadGroupArgs) *XStreamSliceCmd {
	args := []any{"xreadgroup", "group", a.Group, a.Consumer}
	if a.Count > 0 {
		args = append(args, "count", a.Count)
	}
	if a.NoAck {
		args = append(args, "noack")
	}
	return c.streams(ctx, args, a.Block, a.Streams)
}

func (c cmdable) streams(
	ctx context.Context,
	args []any,
	block time.Duration,
	keys []string,
) *XStreamSliceCmd {
	if block >= 0 {
		args = append(args, "block", int64(block/time.Millisecond))
	}
	args = appendStrings(append(args, "streams"), keys)
	cmd := &XStreamSliceCmd{baseCmd: baseCmd{args: args}}
	if block >= 0 {
		cmd.blocking, cmd.block = true, block
	}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) XAck(ctx context.Context, stream, group string, ids ...string) *IntCmd {
	return c.integer(ctx, appendStrings([]any{"xack", stream, group}, ids)...)
}

func (c cmdable) XClaim(ctx context.Context, a *XClaimArgs) *XMessageSliceCmd {
	args := []any{"xclaim", a.Stream, a.Group, a.Consumer, int64(a.MinIdle / time.Millisecond)}
	return c.messages(ctx, appendStrings(args, a.Messages)...)
}

func (c cmdable) XPendingExt(ctx context.Context, a *XPendingExtArgs) *XPendingExtCmd {
	args := []any{"xpending", a.Stream, a.Group}
	if a.Idle != 0 {
		args = append(args, "idle", int64(a.Idle/time.Millisecond))
	}
	args = append(args, a.Start, a.End, a.Count)
	if a.Consumer != "" {
		args = append(args, a.Consumer)
	}
	cmd := &XPendingExtCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) XGroupCreateMkStream(ctx context.Context, stream, group, start string) *StatusCmd {
	return c.status(ctx, "xgroup", "create", stream, group, start, "mkstream")
}

func (c cmdable) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *IntCmd {
	return c.integer(ctx, "xgroup", "delconsumer", stream, group, consumer)
}

func (c cmdable) XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *IntCmd {
	args := []any{"xtrim", key, "maxlen", "~", maxLen}
	if limit > 0 {
		args = append(args, "limit", limit)
	}
	return c.integer(ctx, args...)
}

func (c cmdable) XRangeN(ctx context.Context, key, start, stop string, n int64) *XMessageSliceCmd {
	return c.messages(ctx, "xrange", key, start, stop, "count", n)
}

func (c cmdable) XRevRangeN(
	ctx context.Context,
	key, start, stop string,
	n int64,
) *XMessageSliceCmd {
	return c.messages(ctx, "xrevrange", key, start, stop, "count", n)
}

func (c cmdable) messages(ctx context.Context, args ...any) *XMessageSliceCmd {
	cmd := &XMessageSliceCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) XLen(ctx context.Context, stream string) *IntCmd {
	return c.integer(ctx, "xlen", stream)
}

func (c cmdable) XInfoGroups(ctx context.Context, stream string) *XInfoGroupsCmd {
	cmd := &XInfoGroupsCmd{baseCmd: baseCmd{args: []any{"xinfo", "groups", stream}}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) XInfoConsumers(ctx context.Context, stream, group string) *XInfoConsumersCmd {
	cmd := &XInfoConsumersCmd{baseCmd: baseCmd{args: []any{"xinfo", "consumers", stream, group}}}
	c.run(ctx, cmd)
	return cmd
}
