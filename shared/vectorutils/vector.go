package vectorutils

import (
	"encoding/binary"
	"errors"
	"math"
)

const float32Bytes = 4

var (
	ErrPackedLength   = errors.New("packed vector length is not a multiple of four bytes")
	ErrNotFinite      = errors.New("vector holds a component that is not a finite number")
	ErrZeroMagnitude  = errors.New("vector has zero magnitude")
	ErrDimensionsDiff = errors.New("vectors have different dimensions")
)

func Pack(vector []float32) []byte {
	if len(vector) == 0 {
		return nil
	}

	packed := make([]byte, len(vector)*float32Bytes)
	for idx, value := range vector {
		binary.LittleEndian.PutUint32(packed[idx*float32Bytes:], math.Float32bits(value))
	}

	return packed
}

func Unpack(packed []byte) ([]float32, error) {
	if len(packed)%float32Bytes != 0 {
		return nil, ErrPackedLength
	}
	if len(packed) == 0 {
		return nil, nil
	}

	vector := make([]float32, len(packed)/float32Bytes)
	for idx := range vector {
		value := math.Float32frombits(binary.LittleEndian.Uint32(packed[idx*float32Bytes:]))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, ErrNotFinite
		}
		vector[idx] = value
	}

	return vector, nil
}

func Magnitude(vector []float32) float64 {
	var sum float64
	for _, value := range vector {
		component := float64(value)
		sum += component * component
	}

	return math.Sqrt(sum)
}

func Normalized(vector []float32) ([]float32, error) {
	magnitude := Magnitude(vector)
	if magnitude == 0 {
		return nil, ErrZeroMagnitude
	}
	if math.IsNaN(magnitude) || math.IsInf(magnitude, 0) {
		return nil, ErrNotFinite
	}

	out := make([]float32, len(vector))
	for idx, value := range vector {
		out[idx] = float32(float64(value) / magnitude)
	}

	return out, nil
}

func Dot(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, ErrDimensionsDiff
	}

	var sum float64
	for idx := range a {
		sum += float64(a[idx]) * float64(b[idx])
	}

	return sum, nil
}

func Cosine(a, b []float32) (float64, error) {
	dot, err := Dot(a, b)
	if err != nil {
		return 0, err
	}

	magnitudes := Magnitude(a) * Magnitude(b)
	if magnitudes == 0 {
		return 0, ErrZeroMagnitude
	}

	return dot / magnitudes, nil
}
