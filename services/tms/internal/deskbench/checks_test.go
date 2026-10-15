package deskbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplyStates_ReadsASmallCountInWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reply string
		value string
		want  bool
	}{
		{reply: "saved with four document profiles", value: "4", want: true},
		{reply: "saved with 4 document profiles", value: "4", want: true},
		{reply: "Four profiles were saved", value: "4", want: true},
		{reply: "saved with fourteen profiles", value: "4", want: false},
		{reply: "nothing was saved for anyone", value: "1", want: false},
		{reply: "saved forty profiles", value: "40", want: false},
		{reply: "saved 2.5 profiles", value: "2.5", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.reply, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, replyStates(tt.reply, tt.value))
		})
	}
}
