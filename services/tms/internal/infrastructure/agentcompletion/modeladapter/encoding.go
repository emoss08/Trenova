package modeladapter

import "github.com/bytedance/sonic"

// requestJSON encodes what is sent to a provider with map keys in order. Tool
// schemas, replayed tool-call arguments and merged vendor fields are maps, and
// walked in Go's random order they made every request different bytes, so no
// provider could reuse the prefix it cached from the last one.
var requestJSON = sonic.Config{SortMapKeys: true}.Froze()
