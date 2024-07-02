package types

type HTTPResponse[T any] struct {
	Count   int    `json:"count"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
	Results T      `json:"results"`
}
