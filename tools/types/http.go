package types

type HTTPResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  any    `json:"results"`
}

type ValidationErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr"`
}

type ValidationErrorResponse struct {
	Type   string                  `json:"type"`
	Errors []ValidationErrorDetail `json:"errors"`
}
