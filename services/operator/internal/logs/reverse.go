package logs

import (
	"bytes"
	"errors"
	"io"
)

const reverseChunk = 64 << 10

func reverseLines(r io.ReaderAt, size int64, emit func(line []byte) error) error {
	pos := size
	var pending []byte
	for {
		if len(pending) > 0 {
			body := pending[:len(pending)-1]
			if i := bytes.LastIndexByte(body, '\n'); i >= 0 {
				if err := emit(body[i+1:]); err != nil {
					return err
				}
				pending = pending[:i+1]
				continue
			}
			if pos == 0 {
				return emit(body)
			}
		}
		if pos == 0 {
			return nil
		}
		n := min(int64(reverseChunk), pos)
		pos -= n
		chunk := make([]byte, int(n)+len(pending))
		if _, err := r.ReadAt(chunk[:n], pos); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		copy(chunk[n:], pending)
		pending = chunk
	}
}
