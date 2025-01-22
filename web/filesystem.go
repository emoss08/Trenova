package web

import "embed"

// Embed a directory
//
//go:embed report-templates/*
var embedDirTemplates embed.FS

func GetTemplatesFS() embed.FS {
	return embedDirTemplates
}
