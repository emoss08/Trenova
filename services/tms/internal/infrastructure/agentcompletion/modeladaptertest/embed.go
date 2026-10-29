package modeladaptertest

import (
	"sync"
)

type EmbedCapture struct {
	Mu   sync.Mutex
	Path string
	Auth string
	Body map[string]any
}

func (c *EmbedCapture) Snapshot() (string, string, map[string]any) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	return c.Path, c.Auth, c.Body
}
