package samsara

type Client interface {
	GetTags() ([]Tag, error)
}

type client struct {
}
