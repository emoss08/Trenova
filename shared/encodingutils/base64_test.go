package encodingutils

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodedBase64Size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int64
		want int64
	}{
		{name: "zero", size: 0, want: 0},
		{name: "negative", size: -1, want: 0},
		{name: "one byte", size: 1, want: 4},
		{name: "two bytes", size: 2, want: 4},
		{name: "three bytes", size: 3, want: 4},
		{name: "four bytes", size: 4, want: 8},
		{
			name: "ten megabytes",
			size: 10 * 1024 * 1024,
			want: int64(base64.StdEncoding.EncodedLen(10 * 1024 * 1024)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, EncodedBase64Size(tt.size))
		})
	}
}
