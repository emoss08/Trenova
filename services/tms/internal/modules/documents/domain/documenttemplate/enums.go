package documenttemplate

import "errors"

type PageSize string

const (
	PageSizeLetter = PageSize("Letter")
	PageSizeA4     = PageSize("A4")
	PageSizeLegal  = PageSize("Legal")
)

func (p PageSize) String() string { return string(p) }

func (p PageSize) IsValid() bool {
	switch p {
	case PageSizeLetter, PageSizeA4, PageSizeLegal:
		return true
	default:
		return false
	}
}

func PageSizeFromString(v string) (PageSize, error) {
	switch v {
	case "Letter":
		return PageSizeLetter, nil
	case "A4":
		return PageSizeA4, nil
	case "Legal":
		return PageSizeLegal, nil
	default:
		return "", errors.New("invalid document template page size")
	}
}

// Dimensions returns the page width and height in inches, portrait-oriented.
// Gotenberg takes paper size in inches, so inches are the storage unit for the
// page box even though margins are authored in millimeters.
func (p PageSize) Dimensions() (width, height float64) {
	switch p {
	case PageSizeA4:
		return 8.27, 11.69
	case PageSizeLegal:
		return 8.5, 14
	case PageSizeLetter:
		return 8.5, 11
	default:
		return 8.5, 11
	}
}

type Orientation string

const (
	OrientationPortrait  = Orientation("Portrait")
	OrientationLandscape = Orientation("Landscape")
)

func (o Orientation) String() string { return string(o) }

func (o Orientation) IsValid() bool {
	switch o {
	case OrientationPortrait, OrientationLandscape:
		return true
	default:
		return false
	}
}

func (o Orientation) IsLandscape() bool { return o == OrientationLandscape }

func OrientationFromString(v string) (Orientation, error) {
	switch v {
	case "Portrait":
		return OrientationPortrait, nil
	case "Landscape":
		return OrientationLandscape, nil
	default:
		return "", errors.New("invalid document template orientation")
	}
}

// millimetersPerInch converts authored margins to the inches Gotenberg expects.
const millimetersPerInch = 25.4

// maxMarginMillimeters bounds a single margin so a template cannot request a
// page box with no printable area left.
const maxMarginMillimeters = 100

// Margins holds the four page margins in millimeters, matching the unit the old
// gofpdf invoice renderer used so authored values carry over unchanged.
type Margins struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
}

// DefaultMargins is the page box a template gets when it declares none.
func DefaultMargins() Margins {
	return Margins{Top: 20, Bottom: 20, Left: 20, Right: 20}
}

func (m Margins) IsValid() bool {
	for _, v := range [...]float64{m.Top, m.Bottom, m.Left, m.Right} {
		if v < 0 || v > maxMarginMillimeters {
			return false
		}
	}
	return true
}

// Inches converts to the unit the rendering adapter submits.
func (m Margins) Inches() Margins {
	return Margins{
		Top:    m.Top / millimetersPerInch,
		Bottom: m.Bottom / millimetersPerInch,
		Left:   m.Left / millimetersPerInch,
		Right:  m.Right / millimetersPerInch,
	}
}
