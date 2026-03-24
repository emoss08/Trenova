package docs

import _ "embed"

//go:embed swagger.json
var rawSpec []byte

func ReadSpec() []byte {
	return rawSpec
}
