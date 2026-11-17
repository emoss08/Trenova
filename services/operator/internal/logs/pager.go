package logs

import (
	"context"
	"errors"
	"time"
)

const (
	pageLimit    = 2000
	minPageLimit = 10
)

type pager struct {
	src       Source
	component string
	since     time.Time
	end       time.Time
	limit     int
	boundary  time.Time
	atEdge    map[string]int
	buf       []Entry
	done      bool
}

func newPager(src Source, component string, since, until time.Time, limit int) *pager {
	return &pager{
		src: src, component: component, since: since, end: until, limit: limit, atEdge: map[string]int{},
	}
}

func (p *pager) next(ctx context.Context) (Entry, bool, error) {
	for len(p.buf) == 0 {
		if p.done {
			return Entry{}, false, nil
		}
		if err := p.fill(ctx); err != nil {
			return Entry{}, false, err
		}
	}
	e := p.buf[0]
	p.buf = p.buf[1:]
	return e, true, nil
}

func (p *pager) fill(ctx context.Context) error {
	if !p.end.After(p.since) {
		p.done = true
		return nil
	}
	page, err := p.src.NewestFirst(ctx, PageQuery{
		Component: p.component, Start: p.since, End: p.end, Limit: p.limit,
	})
	if errors.Is(err, ErrPageTooLarge) && p.limit > minPageLimit {
		p.limit = max(p.limit/2, minPageLimit)
		return nil
	}
	if err != nil {
		return err
	}
	if len(page) == 0 {
		p.done = true
		return nil
	}
	skip := map[string]int{}
	for i := range page {
		e := &page[i]
		if e.Time.Equal(p.boundary) {
			k := edgeKey(e)
			if skip[k] < p.atEdge[k] {
				skip[k]++
				continue
			}
		}
		p.buf = append(p.buf, *e)
	}
	oldest := page[len(page)-1].Time
	edge := map[string]int{}
	for i := range page {
		if page[i].Time.Equal(oldest) {
			edge[edgeKey(&page[i])]++
		}
	}
	switch {
	case len(p.buf) > 0:
		p.boundary, p.atEdge, p.end = oldest, edge, oldest.Add(time.Nanosecond)
	case len(page) >= p.limit:
		p.boundary, p.atEdge, p.end = time.Time{}, map[string]int{}, oldest
	default:
		p.done = true
	}
	return nil
}

func edgeKey(e *Entry) string {
	return e.stream + "\x00" + e.Line
}
