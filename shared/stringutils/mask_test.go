package stringutils

import "testing"

func TestMaskTail(t *testing.T) {
	tests := []struct {
		value   string
		visible int
		want    string
	}{
		{value: "", visible: 4, want: ""},
		{value: "D1234567", visible: 4, want: "••••4567"},
		{value: "1234", visible: 4, want: "••••"},
		{value: "12", visible: 4, want: "••"},
		{value: "abc", visible: 0, want: "•••"},
		{value: "abc", visible: -1, want: "•••"},
		{value: "ñandú99", visible: 2, want: "•••••99"},
	}
	for _, tt := range tests {
		if got := MaskTail(tt.value, tt.visible); got != tt.want {
			t.Errorf("MaskTail(%q, %d) = %q, want %q", tt.value, tt.visible, got, tt.want)
		}
	}
}
