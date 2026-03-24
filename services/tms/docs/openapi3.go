package docs

import _ "embed"

//go:embed openapi-3.json
var rawOpenAPI3Spec []byte

func ReadOpenAPI3Spec() []byte {
	return rawOpenAPI3Spec
}
