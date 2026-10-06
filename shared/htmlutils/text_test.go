package htmlutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "   ", want: ""},
		{name: "plain fragment", in: "Congrats on sending your <strong>first email</strong>!", want: "Congrats on sending your first email!"},
		{
			name: "paragraphs and breaks",
			in:   "<p>Load 88213</p><p>Pickup Thursday<br>Dock 4</p>",
			want: "Load 88213\n\nPickup Thursday\nDock 4",
		},
		{
			name: "drops head, style and script",
			in:   "<html><head><title>T</title><style>p{color:red}</style></head><body><script>x()</script><div>Rate $1,200</div></body></html>",
			want: "Rate $1,200",
		},
		{
			name: "lists are bulleted",
			in:   "<ul><li>Origin: Dallas</li><li>Destination: Tulsa</li></ul>",
			want: "- Origin: Dallas\n- Destination: Tulsa",
		},
		{
			name: "table cells stay on one line",
			in:   "<table><tr><td>PRO</td><td>12345</td></tr><tr><td>BOL</td><td>B-9</td></tr></table>",
			want: "PRO 12345\nBOL B-9",
		},
		{
			name: "collapses indentation",
			in:   "<div>\n    Please   confirm\n    pickup\n</div>",
			want: "Please confirm pickup",
		},
		{name: "decodes entities", in: "<p>Smith &amp; Sons &lt;ops&gt;</p>", want: "Smith & Sons <ops>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ToText(tt.in))
		})
	}
}
