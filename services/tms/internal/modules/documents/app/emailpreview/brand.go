package emailpreview

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/pkg/emaildesign"
)

const markSize = 60

var sampleMark = sync.OnceValue(func() string {
	img := image.NewRGBA(image.Rect(0, 0, markSize, markSize))
	field := color.RGBA{R: 0x1f, G: 0x4d, B: 0x3a, A: 0xff}
	petal := color.RGBA{R: 0xe8, G: 0xd9, B: 0xb5, A: 0xff}
	for y := range markSize {
		for x := range markSize {
			img.Set(x, y, field)
			dx, dy := x-markSize/2, y-markSize/2
			if dx*dx+dy*dy < 14*14 && (dx > -4 && dx < 4 || dy > -4 && dy < 4) {
				img.Set(x, y, petal)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
})

func SampleBrand() emaildesign.Brand {
	return emaildesign.Brand{Name: templatecatalog.SampleCompanyName, Mark: sampleMark()}
}
