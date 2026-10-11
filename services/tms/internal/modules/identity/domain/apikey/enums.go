package apikey

type Status string

const (
	StatusActive  = Status("active")
	StatusRevoked = Status("revoked")
)
