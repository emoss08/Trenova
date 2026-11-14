package tenant

import "github.com/emoss08/trenova/pkg/errortypes"

type BrandImage string

const (
	BrandImageLogo BrandImage = "logo"
	BrandImageIcon BrandImage = "icon"
)

func ParseBrandImage(value string) (BrandImage, error) {
	switch image := BrandImage(value); image {
	case BrandImageLogo, BrandImageIcon:
		return image, nil
	default:
		return "", errortypes.NewNotFoundError("Organization image not found")
	}
}

func (o *Organization) BrandImageKey(image BrandImage) string {
	if image == BrandImageIcon {
		return o.IconURL
	}
	return o.LogoURL
}

func (o *Organization) SetBrandImageKey(image BrandImage, key string) {
	if image == BrandImageIcon {
		o.IconURL = key
		return
	}
	o.LogoURL = key
}
