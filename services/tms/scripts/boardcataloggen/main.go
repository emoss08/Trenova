package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	target     = "../../client/packages/shared/src/types/generated/board-catalog.ts"
	targetMode = 0o644
)

func main() {
	path := filepath.FromSlash(target)
	if err := os.WriteFile(path, []byte(typeScript()), targetMode); err != nil {
		fmt.Fprintln(os.Stderr, "boardcataloggen:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
}
