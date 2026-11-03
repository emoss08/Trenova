package templateengine

import (
	"embed"
	"encoding/base64"
	"strings"
)

//go:embed fonts/*.woff2
var fontFS embed.FS

const (
	latinRange = "U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC," +
		"U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212," +
		"U+2215,U+FEFF,U+FFFD"
	latinExtRange = "U+0100-02BA,U+02BD-02C5,U+02C7-02CC,U+02CE-02D7,U+02DD-02FF,U+0304," +
		"U+0308,U+0329,U+1D00-1DBF,U+1E00-1E9F,U+1EF2-1EFF,U+2020,U+20A0-20AB," +
		"U+20AD-20C0,U+2113,U+2C60-2C7F,U+A720-A7FF"
)

type fontFace struct {
	family string
	file   string
	weight string
	format string
	ranges string
}

var documentFontFaces = []fontFace{
	{"Geist", "geist-latin-ext.woff2", "100 900", "woff2-variations", latinExtRange},
	{"Geist", "geist-latin.woff2", "100 900", "woff2-variations", latinRange},
	{"Geist Mono", "geist-mono-400.woff2", "400", "woff2", latinRange},
	{"Geist Mono", "geist-mono-500.woff2", "500", "woff2", latinRange},
}

var documentFonts = buildDocumentFonts()

func buildDocumentFonts() string {
	var b strings.Builder
	for _, face := range documentFontFaces {
		data, err := fontFS.ReadFile("fonts/" + face.file)
		if err != nil {
			panic("templateengine: embedded font " + face.file + " is missing: " + err.Error())
		}
		b.WriteString("@font-face{font-family:\"")
		b.WriteString(face.family)
		b.WriteString("\";font-style:normal;font-display:block;font-weight:")
		b.WriteString(face.weight)
		b.WriteString(";src:url(data:font/woff2;base64,")
		b.WriteString(base64.StdEncoding.EncodeToString(data))
		b.WriteString(") format(\"")
		b.WriteString(face.format)
		b.WriteString("\");unicode-range:")
		b.WriteString(face.ranges)
		b.WriteString(";}\n")
	}
	return b.String()
}
