package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

type Cmder interface {
	Name() string
	Args() []any
	Err() error
	base() *baseCmd
	readReply(v resp.Value) error
}

type baseCmd struct {
	args     []any
	err      error
	block    time.Duration
	blocking bool
}

func (c *baseCmd) Name() string {
	if len(c.args) == 0 {
		return ""
	}
	name, _ := c.args[0].(string)
	return strings.ToLower(name)
}

func (c *baseCmd) Args() []any { return c.args }

func (c *baseCmd) Err() error { return c.err }

func (c *baseCmd) base() *baseCmd { return c }

func applyReply(cmd Cmder, v resp.Value) {
	b := cmd.base()
	if v.Kind == resp.KindError {
		b.err = resp.Error(v.Str)
		return
	}
	b.err = cmd.readReply(v)
}

type cmdable func(ctx context.Context, cmd Cmder) error

func (c cmdable) run(ctx context.Context, cmd Cmder) {
	_ = c(ctx, cmd)
}

type Cmd struct {
	baseCmd
	val any
}

func (c cmdable) cmd(ctx context.Context, args ...any) *Cmd {
	cmd := &Cmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *Cmd) readReply(v resp.Value) error {
	if v.Kind == resp.KindNil {
		return Nil
	}
	c.val = v.Any()
	return nil
}

func (c *Cmd) Text() (string, error) {
	if c.err != nil {
		return "", c.err
	}
	s, ok := c.val.(string)
	if !ok {
		return "", fmt.Errorf("redis: unexpected type=%T for String", c.val)
	}
	return s, nil
}

func (c *Cmd) Int() (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	n, err := anyInt(c.val)
	return int(n), err
}

func (c *Cmd) Slice() ([]any, error) {
	if c.err != nil {
		return nil, c.err
	}
	s, ok := c.val.([]any)
	if !ok {
		return nil, fmt.Errorf("redis: unexpected type=%T for Slice", c.val)
	}
	return s, nil
}

func (c *Cmd) StringSlice() ([]string, error) {
	elems, err := c.Slice()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(elems))
	for idx, elem := range elems {
		s, ok := elem.(string)
		if !ok {
			return nil, fmt.Errorf("redis: unexpected element type=%T for StringSlice", elem)
		}
		out[idx] = s
	}
	return out, nil
}

func (c *Cmd) Int64Slice() ([]int64, error) {
	elems, err := c.Slice()
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(elems))
	for idx, elem := range elems {
		if out[idx], err = anyInt(elem); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func anyInt(v any) (int64, error) {
	switch v := v.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("redis: unexpected type=%T for Int64", v)
	}
}

type StringCmd struct {
	baseCmd
	val string
}

func (c cmdable) str(ctx context.Context, args ...any) *StringCmd {
	cmd := &StringCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *StringCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Text()
	return err
}

func (c *StringCmd) Val() string { return c.val }

func (c *StringCmd) Result() (string, error) { return c.val, c.err }

func (c *StringCmd) Bytes() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(c.val), nil
}

func (c *StringCmd) Int64() (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return strconv.ParseInt(c.val, 10, 64)
}

type StatusCmd struct {
	baseCmd
	val string
}

func (c cmdable) status(ctx context.Context, args ...any) *StatusCmd {
	cmd := &StatusCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *StatusCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Text()
	return err
}

func (c *StatusCmd) Result() (string, error) { return c.val, c.err }

type IntCmd struct {
	baseCmd
	val int64
}

func (c cmdable) integer(ctx context.Context, args ...any) *IntCmd {
	cmd := &IntCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *IntCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Integer()
	return err
}

func (c *IntCmd) Val() int64 { return c.val }

func (c *IntCmd) Result() (int64, error) { return c.val, c.err }

type BoolCmd struct {
	baseCmd
	val bool
}

func (c cmdable) boolean(ctx context.Context, args ...any) *BoolCmd {
	cmd := &BoolCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *BoolCmd) readReply(v resp.Value) error {
	switch v.Kind {
	case resp.KindNil:
		c.val = false
	case resp.KindInt:
		c.val = v.Int != 0
	case resp.KindStatus, resp.KindBulk:
		c.val = v.Str == "OK"
	default:
		return v.Unexpected()
	}
	return nil
}

func (c *BoolCmd) Result() (bool, error) { return c.val, c.err }

type FloatCmd struct {
	baseCmd
	val float64
}

func (c *FloatCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Float()
	return err
}

func (c *FloatCmd) Result() (float64, error) { return c.val, c.err }

type FloatSliceCmd struct {
	baseCmd
	val []float64
}

func (c *FloatSliceCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Floats()
	return err
}

func (c *FloatSliceCmd) Val() []float64 { return c.val }

func (c *FloatSliceCmd) Result() ([]float64, error) { return c.val, c.err }

type DurationCmd struct {
	baseCmd
	val       time.Duration
	precision time.Duration
}

func (c cmdable) duration(ctx context.Context, precision time.Duration, args ...any) *DurationCmd {
	cmd := &DurationCmd{baseCmd: baseCmd{args: args}, precision: precision}
	c.run(ctx, cmd)
	return cmd
}

func (c *DurationCmd) readReply(v resp.Value) error {
	n, err := v.Integer()
	if err != nil {
		return err
	}
	c.val = time.Duration(n)
	if n >= 0 {
		c.val *= c.precision
	}
	return nil
}

func (c *DurationCmd) Val() time.Duration { return c.val }

func (c *DurationCmd) Result() (time.Duration, error) { return c.val, c.err }

type StringSliceCmd struct {
	baseCmd
	val []string
}

func (c cmdable) strs(ctx context.Context, args ...any) *StringSliceCmd {
	cmd := &StringSliceCmd{baseCmd: baseCmd{args: args}}
	c.run(ctx, cmd)
	return cmd
}

func (c *StringSliceCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Strings()
	return err
}

func (c *StringSliceCmd) Result() ([]string, error) { return c.val, c.err }

type SliceCmd struct {
	baseCmd
	val []any
}

func (c *SliceCmd) readReply(v resp.Value) error {
	if _, err := v.Array(); err != nil {
		return err
	}
	c.val, _ = v.Any().([]any)
	return nil
}

func (c *SliceCmd) Result() ([]any, error) { return c.val, c.err }

type MapStringStringCmd struct {
	baseCmd
	val map[string]string
}

func (c *MapStringStringCmd) readReply(v resp.Value) error {
	flat, err := v.Strings()
	if err != nil {
		return err
	}
	c.val = make(map[string]string, len(flat)/2)
	for idx := 0; idx+1 < len(flat); idx += 2 {
		c.val[flat[idx]] = flat[idx+1]
	}
	return nil
}

func (c *MapStringStringCmd) Result() (map[string]string, error) { return c.val, c.err }

type Z struct {
	Score  float64
	Member any
}

type ZSliceCmd struct {
	baseCmd
	val []Z
}

func (c *ZSliceCmd) readReply(v resp.Value) error {
	flat, err := v.Strings()
	if err != nil {
		return err
	}
	c.val = make([]Z, 0, len(flat)/2)
	for idx := 0; idx+1 < len(flat); idx += 2 {
		score, err := strconv.ParseFloat(flat[idx+1], 64)
		if err != nil {
			return err
		}
		c.val = append(c.val, Z{Score: score, Member: flat[idx]})
	}
	return nil
}

func (c *ZSliceCmd) Result() ([]Z, error) { return c.val, c.err }

type ScanCmd struct {
	baseCmd
	keys   []string
	cursor uint64
}

func (c *ScanCmd) readReply(v resp.Value) error {
	elems, err := v.Array()
	if err != nil {
		return err
	}
	if len(elems) != 2 {
		return v.Unexpected()
	}
	cursor, err := elems[0].Text()
	if err != nil {
		return err
	}
	if c.cursor, err = strconv.ParseUint(cursor, 10, 64); err != nil {
		return err
	}
	c.keys, err = elems[1].Strings()
	return err
}

func (c *ScanCmd) Result() ([]string, uint64, error) { return c.keys, c.cursor, c.err }

type JSONCmd struct {
	baseCmd
	val string
}

func (c *JSONCmd) readReply(v resp.Value) (err error) {
	c.val, err = v.Text()
	return err
}

func (c *JSONCmd) Result() (string, error) { return c.val, c.err }
