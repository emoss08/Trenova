//go:build ignore

package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
)

const constraint = "//go:build integration\n\n"

func main() {
	for _, dir := range os.Args[1:] {
		path := filepath.Join(dir, "keel_gen.go")
		src, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if bytes.HasPrefix(src, []byte(constraint)) {
			continue
		}
		if err = os.WriteFile(path, append([]byte(constraint), src...), 0o600); err != nil {
			log.Fatal(err)
		}
	}
}
