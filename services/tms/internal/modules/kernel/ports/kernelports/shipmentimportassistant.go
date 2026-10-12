package kernelports

type StreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
	At    int64  `json:"-"`
}
