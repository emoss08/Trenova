//go:build fuzz

package fileutils

import (
	"mime"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzContentDispositionNamesAFileWithoutAPath(f *testing.F) {
	for _, seed := range []string{
		"invoice.pdf", "../../etc/passwd", `C:\Users\billing\pod.pdf`, "pod\r\nX-Injected: 1.pdf",
		"Relevé de compte.pdf", "", ".", "/", "日本語の請求書.pdf", "a\"b;c=d.pdf", "\x00\xff.pdf",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, filename string) {
		header := ContentDisposition("attachment", filename)

		disposition, params, err := mime.ParseMediaType(header)
		if err != nil {
			t.Fatalf("%q gave a header that does not parse: %q: %v", filename, header, err)
		}
		if disposition != "attachment" {
			t.Fatalf("%q changed the disposition to %q", filename, disposition)
		}
		name := params["filename"]
		if name == "" || strings.ContainsAny(name, "/\\\r\n") {
			t.Fatalf("%q named the download %q", filename, name)
		}
	})
}

func FuzzSanitizeDisplayFilenameKeepsAValidShortName(f *testing.F) {
	for _, seed := range []struct {
		name   string
		maxLen int
	}{
		{"Q3 report", 120}, {"Relevé de compte", 6}, {"a/b\\c:d*e?f\"g<h>i|j", 40},
		{"日本", 2}, {"   ", 10}, {"line\r\nbreak", 5}, {"\xff\xfe", 1}, {"plain", 0},
	} {
		f.Add(seed.name, seed.maxLen)
	}
	f.Fuzz(func(t *testing.T, name string, maxLen int) {
		maxLen %= 512
		got := SanitizeDisplayFilename(name, "report", maxLen)

		if got == "" {
			t.Fatalf("%q came back empty", name)
		}
		if utf8.ValidString(name) && !utf8.ValidString(got) {
			t.Fatalf("%q at %d bytes came back as invalid UTF-8: %q", name, maxLen, got)
		}
		if maxLen > 0 && got != "report" && len(got) > maxLen {
			t.Fatalf("%q came back longer than %d bytes: %q", name, maxLen, got)
		}
		if strings.ContainsAny(got, "/\\:*?\"<>|\r\n") {
			t.Fatalf("%q kept a character a file name cannot hold: %q", name, got)
		}
	})
}
