package redis

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
)

type SetArgs struct {
	Mode string
	TTL  time.Duration
}

func (c cmdable) Do(ctx context.Context, args ...any) *Cmd {
	return c.cmd(ctx, args...)
}

func (c cmdable) Ping(ctx context.Context) *StatusCmd {
	return c.status(ctx, "ping")
}

func (c cmdable) ClientID(ctx context.Context) *IntCmd {
	return c.integer(ctx, "client", "id")
}

func (c cmdable) FlushDB(ctx context.Context) *StatusCmd {
	return c.status(ctx, "flushdb")
}

func (c cmdable) FlushAll(ctx context.Context) *StatusCmd {
	return c.status(ctx, "flushall")
}

func (c cmdable) Get(ctx context.Context, key string) *StringCmd {
	return c.str(ctx, "get", key)
}

func (c cmdable) GetDel(ctx context.Context, key string) *StringCmd {
	return c.str(ctx, "getdel", key)
}

func (c cmdable) Set(ctx context.Context, key string, value any, ttl time.Duration) *StatusCmd {
	return c.status(ctx, setArgs(key, value, ttl)...)
}

func (c cmdable) SetNX(ctx context.Context, key string, value any, ttl time.Duration) *BoolCmd {
	if ttl <= 0 {
		return c.boolean(ctx, "setnx", key, value)
	}
	return c.boolean(ctx, append(setArgs(key, value, ttl), "nx")...)
}

func (c cmdable) SetArgs(ctx context.Context, key string, value any, a SetArgs) *StatusCmd {
	args := setArgs(key, value, a.TTL)
	if a.Mode != "" {
		args = append(args, a.Mode)
	}
	return c.status(ctx, args...)
}

func (c cmdable) Incr(ctx context.Context, key string) *IntCmd {
	return c.integer(ctx, "incr", key)
}

func (c cmdable) Del(ctx context.Context, keys ...string) *IntCmd {
	return c.integer(ctx, appendStrings([]any{"del"}, keys)...)
}

func (c cmdable) Unlink(ctx context.Context, keys ...string) *IntCmd {
	return c.integer(ctx, appendStrings([]any{"unlink"}, keys)...)
}

func (c cmdable) Exists(ctx context.Context, keys ...string) *IntCmd {
	return c.integer(ctx, appendStrings([]any{"exists"}, keys)...)
}

func (c cmdable) Keys(ctx context.Context, pattern string) *StringSliceCmd {
	return c.strs(ctx, "keys", pattern)
}

func (c cmdable) Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd {
	args := []any{"scan", cursor}
	if match != "" {
		args = append(args, "match", match)
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	cmd := &ScanCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) Expire(ctx context.Context, key string, ttl time.Duration) *BoolCmd {
	return c.boolean(ctx, "expire", key, seconds(ttl))
}

func (c cmdable) ExpireNX(ctx context.Context, key string, ttl time.Duration) *BoolCmd {
	return c.boolean(ctx, "expire", key, seconds(ttl), "nx")
}

func (c cmdable) ExpireGT(ctx context.Context, key string, ttl time.Duration) *BoolCmd {
	return c.boolean(ctx, "expire", key, seconds(ttl), "gt")
}

func (c cmdable) ExpireAt(ctx context.Context, key string, at time.Time) *BoolCmd {
	return c.boolean(ctx, "expireat", key, at.Unix())
}

func (c cmdable) TTL(ctx context.Context, key string) *DurationCmd {
	return c.duration(ctx, time.Second, "ttl", key)
}

func (c cmdable) PTTL(ctx context.Context, key string) *DurationCmd {
	return c.duration(ctx, time.Millisecond, "pttl", key)
}

func (c cmdable) HSet(ctx context.Context, key string, values ...any) *IntCmd {
	return c.integer(ctx, appendValues([]any{"hset", key}, values)...)
}

func (c cmdable) HGet(ctx context.Context, key, field string) *StringCmd {
	return c.str(ctx, "hget", key, field)
}

func (c cmdable) HMGet(ctx context.Context, key string, fields ...string) *SliceCmd {
	cmd := &SliceCmd{baseCmd: baseCmd{args: appendStrings([]any{"hmget", key}, fields)}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) HGetAll(ctx context.Context, key string) *MapStringStringCmd {
	cmd := &MapStringStringCmd{baseCmd: baseCmd{args: []any{"hgetall", key}}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) HDel(ctx context.Context, key string, fields ...string) *IntCmd {
	return c.integer(ctx, appendStrings([]any{"hdel", key}, fields)...)
}

func (c cmdable) SAdd(ctx context.Context, key string, members ...any) *IntCmd {
	return c.integer(ctx, appendValues([]any{"sadd", key}, members)...)
}

func (c cmdable) SRem(ctx context.Context, key string, members ...any) *IntCmd {
	return c.integer(ctx, appendValues([]any{"srem", key}, members)...)
}

func (c cmdable) SMembers(ctx context.Context, key string) *StringSliceCmd {
	return c.strs(ctx, "smembers", key)
}

func (c cmdable) LPush(ctx context.Context, key string, values ...any) *IntCmd {
	return c.integer(ctx, appendValues([]any{"lpush", key}, values)...)
}

func (c cmdable) LLen(ctx context.Context, key string) *IntCmd {
	return c.integer(ctx, "llen", key)
}

func (c cmdable) RPopCount(ctx context.Context, key string, count int) *StringSliceCmd {
	return c.strs(ctx, "rpop", key, count)
}

func (c cmdable) JSONGet(ctx context.Context, key string, paths ...string) *JSONCmd {
	cmd := &JSONCmd{baseCmd: baseCmd{args: appendStrings([]any{"JSON.GET", key}, paths)}}
	c.run(ctx, cmd)
	return cmd
}

func (c cmdable) JSONSet(ctx context.Context, key, path string, value any) *StatusCmd {
	var doc string
	switch v := value.(type) {
	case string:
		doc = v
	case []byte:
		doc = string(v)
	default:
		encoded, err := sonic.MarshalString(v)
		if err != nil {
			return &StatusCmd{baseCmd: baseCmd{args: []any{"JSON.SET", key, path}, err: err}}
		}
		doc = encoded
	}
	return c.status(ctx, "JSON.SET", key, path, doc)
}

func (c cmdable) Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd {
	return c.eval(ctx, "eval", script, keys, args)
}

func (c cmdable) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd {
	return c.eval(ctx, "evalsha", sha1, keys, args)
}

func (c cmdable) eval(ctx context.Context, name, body string, keys []string, args []any) *Cmd {
	cmdArgs := make([]any, 0, 3+len(keys)+len(args))
	cmdArgs = appendStrings(append(cmdArgs, name, body, len(keys)), keys)
	return c.cmd(ctx, append(cmdArgs, args...)...)
}

func setArgs(key string, value any, ttl time.Duration) []any {
	args := make([]any, 3, 6)
	args[0], args[1], args[2] = "set", key, value
	switch {
	case ttl <= 0:
		return args
	case ttl < time.Second || ttl%time.Second != 0:
		return append(args, "px", milliseconds(ttl))
	default:
		return append(args, "ex", int64(ttl/time.Second))
	}
}

func seconds(d time.Duration) int64 {
	if d > 0 && d < time.Second {
		return 1
	}
	return int64(d / time.Second)
}

func milliseconds(d time.Duration) int64 {
	if d > 0 && d < time.Millisecond {
		return 1
	}
	return int64(d / time.Millisecond)
}

func appendStrings(dst []any, values []string) []any {
	for _, v := range values {
		dst = append(dst, v)
	}
	return dst
}

func appendValues(dst, values []any) []any {
	if len(values) != 1 {
		return append(dst, values...)
	}
	switch v := values[0].(type) {
	case []string:
		return appendStrings(dst, v)
	case []any:
		return append(dst, v...)
	case map[string]any:
		for key, val := range v {
			dst = append(dst, key, val)
		}
		return dst
	case map[string]string:
		for key, val := range v {
			dst = append(dst, key, val)
		}
		return dst
	default:
		return append(dst, v)
	}
}
