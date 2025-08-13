package x12

import (
	"bytes"
	"errors"
)

// DetectDelimiters inspects the beginning of the payload to infer separators.
// Strategy:
// - The element separator is the single character immediately after 'ISA'.
// - Parse 16 elements of ISA using that separator; the next byte is the segment terminator.
// - ISA16 (the last element) is the component separator (single char).
// - ISA11 may be repetition separator in newer versions; if not set, 0.
func DetectDelimiters(data []byte) (Delimiters, error) {
	var d Delimiters
	if len(data) < 4 {
		return d, errors.New("payload too short to detect ISA header")
	}
	if string(data[0:3]) != "ISA" {
		return d, errors.New("payload does not start with ISA segment")
	}
	d.Element = data[3]

	// ! Robust approach: count element separators after the tag. In ISA, there are 15
	// ! element separators before ISA16 (the component separator), which is exactly 1 char.
	// ! The following char is the segment terminator.
	count := 0
	i := 4
	for i < len(data) && count < 15 {
		if data[i] == d.Element {
			count++
		}
		i++
	}
	if count < 15 || i+1 >= len(data) {
		// ! Fallback: scan first 200 bytes for a plausible segment terminator and component.
		scan := data
		if len(scan) > 200 {
			scan = data[:200]
		}
		for _, cand := range []byte{'~', '\n', '\r'} {
			if idx := bytes.IndexByte(scan, cand); idx > 0 {
				d.Segment = cand
				d.Component = scan[idx-1]
				d.Repetition = 0
				return d, nil
			}
		}
		return d, errors.New("unable to detect ISA16 and segment terminator")
	}
	d.Component = data[i]
	d.Segment = data[i+1]

	// ! Repetition separator: only defined in 005010+ (ISA11). Keep 0 by default to avoid
	// ! misinterpreting 004010's ISA11 (often 'U') as repetition sep.
	d.Repetition = 0
	return d, nil
}
