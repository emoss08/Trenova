package redis

import (
	"strconv"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

type XMessage struct {
	ID     string
	Values map[string]any
}

type XStream struct {
	Stream   string
	Messages []XMessage
}

type XAddArgs struct {
	Stream string
	MaxLen int64
	Approx bool
	ID     string
	Values any
}

type XReadArgs struct {
	Streams []string
	Count   int64
	Block   time.Duration
}

type XReadGroupArgs struct {
	Group    string
	Consumer string
	Streams  []string
	Count    int64
	Block    time.Duration
	NoAck    bool
}

type XClaimArgs struct {
	Stream   string
	Group    string
	Consumer string
	MinIdle  time.Duration
	Messages []string
}

type XPendingExtArgs struct {
	Stream   string
	Group    string
	Idle     time.Duration
	Start    string
	End      string
	Count    int64
	Consumer string
}

type XPendingExt struct {
	ID         string
	Consumer   string
	Idle       time.Duration
	RetryCount int64
}

type XInfoGroup struct {
	Name string
}

type XInfoConsumer struct {
	Name string
	Idle time.Duration
}

type XMessageSliceCmd struct {
	baseCmd
	val []XMessage
}

func (c *XMessageSliceCmd) readReply(v resp.Value) (err error) {
	c.val, err = readXMessages(v)
	return err
}

func (c *XMessageSliceCmd) Result() ([]XMessage, error) { return c.val, c.err }

type XStreamSliceCmd struct {
	baseCmd
	val []XStream
}

func (c *XStreamSliceCmd) readReply(v resp.Value) error {
	return eachElem(v, func(elem resp.Value) error {
		pair, err := elem.Array()
		if err != nil {
			return err
		}
		if len(pair) != 2 {
			return elem.Unexpected()
		}
		name, err := pair[0].Text()
		if err != nil {
			return err
		}
		msgs, err := readXMessages(pair[1])
		c.val = append(c.val, XStream{Stream: name, Messages: msgs})
		return err
	})
}

func (c *XStreamSliceCmd) Result() ([]XStream, error) { return c.val, c.err }

type XPendingExtCmd struct {
	baseCmd
	val []XPendingExt
}

func (c *XPendingExtCmd) readReply(v resp.Value) error {
	return eachElem(v, func(elem resp.Value) error {
		fields, err := elem.Strings()
		if err != nil {
			return err
		}
		if len(fields) != 4 {
			return elem.Unexpected()
		}
		idle, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return err
		}
		retries, err := strconv.ParseInt(fields[3], 10, 64)
		c.val = append(c.val, XPendingExt{
			ID:         fields[0],
			Consumer:   fields[1],
			Idle:       time.Duration(idle) * time.Millisecond,
			RetryCount: retries,
		})
		return err
	})
}

func (c *XPendingExtCmd) Result() ([]XPendingExt, error) { return c.val, c.err }

type XInfoGroupsCmd struct {
	baseCmd
	val []XInfoGroup
}

func (c *XInfoGroupsCmd) readReply(v resp.Value) error {
	return eachInfoEntry(v, func(fields map[string]resp.Value) error {
		name, err := fields["name"].Text()
		c.val = append(c.val, XInfoGroup{Name: name})
		return err
	})
}

func (c *XInfoGroupsCmd) Result() ([]XInfoGroup, error) { return c.val, c.err }

type XInfoConsumersCmd struct {
	baseCmd
	val []XInfoConsumer
}

func (c *XInfoConsumersCmd) readReply(v resp.Value) error {
	return eachInfoEntry(v, func(fields map[string]resp.Value) error {
		name, err := fields["name"].Text()
		if err != nil {
			return err
		}
		idle, err := fields["idle"].Integer()
		c.val = append(
			c.val,
			XInfoConsumer{Name: name, Idle: time.Duration(idle) * time.Millisecond},
		)
		return err
	})
}

func (c *XInfoConsumersCmd) Result() ([]XInfoConsumer, error) { return c.val, c.err }

func readXMessages(v resp.Value) ([]XMessage, error) {
	msgs := make([]XMessage, 0, len(v.Elems))
	err := eachElem(v, func(elem resp.Value) error {
		if elem.Kind == resp.KindNil {
			return nil
		}
		parts, err := elem.Array()
		if err != nil {
			return err
		}
		if len(parts) < 2 {
			return elem.Unexpected()
		}
		msg := XMessage{}
		if msg.ID, err = parts[0].Text(); err != nil {
			return err
		}
		if parts[1].Kind != resp.KindNil {
			if msg.Values, err = readValues(parts[1]); err != nil {
				return err
			}
		}
		msgs = append(msgs, msg)
		return nil
	})
	return msgs, err
}

func readValues(v resp.Value) (map[string]any, error) {
	flat, err := v.Strings()
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(flat)/2)
	for idx := 0; idx+1 < len(flat); idx += 2 {
		values[flat[idx]] = flat[idx+1]
	}
	return values, nil
}

func eachInfoEntry(v resp.Value, fn func(map[string]resp.Value) error) error {
	return eachElem(v, func(entry resp.Value) error {
		fields, err := entry.Pairs()
		if err != nil {
			return err
		}
		return fn(fields)
	})
}

func eachElem(v resp.Value, fn func(resp.Value) error) error {
	elems, err := v.Array()
	if err != nil {
		return err
	}
	for _, elem := range elems {
		if err = fn(elem); err != nil {
			return err
		}
	}
	return nil
}
