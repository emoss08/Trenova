package board

import (
	"regexp"
)

const (
	MaxReadings        = 6
	MaxReadingKeyRunes = 32
)

var (
	readingKey   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,31}$`)
	readingToken = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]{0,31})\}`)
)

type Reading struct {
	Key string `json:"key"`
	FigureSource
}

func IsReadingKey(key string) bool {
	return readingKey.MatchString(key)
}

func ReadingKey(key string) string {
	return "r." + key
}

func (b *Board) Reading(key string) *Reading {
	for idx := range b.Readings {
		if b.Readings[idx].Key == key {
			return &b.Readings[idx]
		}
	}

	return nil
}

func (f *Filled) Reading(key string) *FilledReading {
	for idx := range f.Readings {
		if f.Readings[idx].Key == key {
			return &f.Readings[idx]
		}
	}

	return nil
}

func (r *FilledReading) Answered() *Value {
	if r.Was != nil {
		return r.Was
	}

	return r.Value
}

func (f *Filled) WriteReadings() {
	f.writeReadings(f.Verdict)
	for idx := range f.Blocks {
		f.writeReadings(f.Blocks[idx].Text)
		f.writeReadings(f.Blocks[idx].Caption)
	}
}

func (f *Filled) writeReadings(segments []FilledSegment) {
	for idx := range segments {
		segment := &segments[idx]
		if segment.Reading == "" {
			continue
		}
		segment.Text = ""
		if reading := f.Reading(segment.Reading); reading != nil {
			segment.Text = FormatValue(reading.Answered())
		}
	}
}
