package resolver

import "sync"

// walkMemo wraps an entity for one evaluation so the Moves and Stops walk
// happens once, however many computed variables read from it. Every field
// accessor unwraps it, so computed functions never know it is there.
type walkMemo struct {
	entity   any
	once     sync.Once
	moves    []any
	movesErr error
	stops    []any
}

// Memoize returns a view of the entity whose stop walk is shared by every
// computed function called with it. Wrapping twice returns the same view.
func Memoize(entity any) any {
	if _, ok := entity.(*walkMemo); ok {
		return entity
	}
	return &walkMemo{entity: entity}
}

// Unwrap returns the entity behind a memoized view, or the value itself.
func Unwrap(entity any) any {
	if memo, ok := entity.(*walkMemo); ok {
		return memo.entity
	}
	return entity
}

func (m *walkMemo) walk() {
	m.once.Do(func() {
		m.moves, m.movesErr = getFieldSlice(m.entity, "Moves")
		m.stops = collectStops(m.moves)
	})
}

func collectStops(moves []any) []any {
	stops := make([]any, 0, len(moves)*2)
	for _, move := range moves {
		moveStops, err := getFieldSlice(move, "Stops")
		if err != nil {
			continue
		}
		stops = append(stops, moveStops...)
	}
	return stops
}

// movesOf reads the entity's moves, once per evaluation when memoized.
func movesOf(entity any) ([]any, error) {
	if memo, ok := entity.(*walkMemo); ok {
		memo.walk()
		return memo.moves, memo.movesErr
	}
	return getFieldSlice(entity, "Moves")
}

// orderedStops reads every stop across the entity's moves in move order,
// once per evaluation when memoized.
func orderedStops(entity any) ([]any, error) {
	if memo, ok := entity.(*walkMemo); ok {
		memo.walk()
		return memo.stops, memo.movesErr
	}
	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return nil, err
	}
	return collectStops(moves), nil
}
