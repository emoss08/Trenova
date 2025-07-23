// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package web

import "embed"

// Embed a directory
//
//go:embed report-templates/*
var embedDirTemplates embed.FS

func GetTemplatesFS() embed.FS {
	return embedDirTemplates
}
