package reference

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/seed"
)

//go:embed data
var data embed.FS

var Seeds = seed.NewRegistry()

func fingerprintOf(build func() any) func() string {
	return func() string {
		b, err := sonic.ConfigStd.Marshal(build())
		if err != nil {
			panic(err)
		}
		sum := sha256.Sum256(b)
		return hex.EncodeToString(sum[:])
	}
}
