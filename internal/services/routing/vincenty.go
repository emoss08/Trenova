// Credit: https://github.com/jftuga/geodist/

package routing

import (
	"errors"
	"math"

	"github.com/emoss08/trenova/internal/util"
)

const (
	SemiMajorAxis    = 6378137           // Semi-major axis of the WGS-84 ellipsoid (in meters).
	SemiMinorAxis    = 6356752.3142      // Semi-minor axis of the WGS-84 ellipsoid (in meters).
	FlatteningFactor = 1 / 298.257223563 // Flattening factor of the WGS-84 ellipsoid.
)

// Coord represents a geographic coordinate using latitude and longitude.
type Coord struct {
	Lat float64 // Latitude in decimal degrees.
	Lon float64 // Longitude in decimal degrees.
}

// String returns a string representation of the coordinate.
//
// Returns:
//
//	string - A string representation of the coordinate in the format "latitude,longitude".
//
// Example:
//
//	coord := Coord{Lat: 40.7128, Lon: -74.0060}
//	fmt.Println(coord.String()) // Output: "40.7128,-74.006"
func (c Coord) String() string {
	return util.FloatToString(c.Lat) + "," + util.FloatToString(c.Lon)
}

// VincentyDistance calculates the distance between two geographic coordinates using the Vincenty formula.
// It returns the distance in miles and kilometers, and an error if the computation fails to converge.
//
// Arguments:
//
//	p1, p2 - Coordinates of the two points.
//
// Returns:
//
//	distance in miles (float64),
//	distance in kilometers (float64),
//	error - error details if the computation does not converge.
func VincentyDistance(p1, p2 Coord) (float64, float64, error) {
	radFactor := math.Pi / 180
	p1.Lat *= radFactor
	p1.Lon *= radFactor
	p2.Lat *= radFactor
	p2.Lon *= radFactor

	l := p2.Lon - p1.Lon
	u1 := math.Atan((1 - FlatteningFactor) * math.Tan(p1.Lat))
	u2 := math.Atan((1 - FlatteningFactor) * math.Tan(p2.Lat))

	sinU1, cosU1 := math.Sin(u1), math.Cos(u1)
	sinU2, cosU2 := math.Sin(u2), math.Cos(u2)

	lambda := l
	lambdaP := 2 * math.Pi
	iterLimit := 20

	var sinLambda, cosLambda, sinSigma, cosSigma, sigma, sinAlpha, cosSqAlpha, cos2SigmaM, c float64

	for math.Abs(lambda-lambdaP) > 1e-12 && iterLimit > 0 {
		iterLimit--
		sinLambda = math.Sin(lambda)
		cosLambda = math.Cos(lambda)

		sinSigma = math.Sqrt(math.Pow(cosU2*sinLambda, 2) + math.Pow(cosU1*sinU2-sinU1*cosU2*cosLambda, 2))
		if sinSigma == 0 {
			return 0, 0, nil // co-incident points
		}

		cosSigma = sinU1*sinU2 + cosU1*cosU2*cosLambda
		sigma = math.Atan2(sinSigma, cosSigma)
		sinAlpha = cosU1 * cosU2 * sinLambda / sinSigma
		cosSqAlpha = 1 - sinAlpha*sinAlpha
		cos2SigmaM = cosSigma - 2*sinU1*sinU2/cosSqAlpha
		if math.IsNaN(cos2SigmaM) {
			cos2SigmaM = 0 // equatorial line: cosSqAlpha=0 (prevents division by zero)
		}

		c = FlatteningFactor / 16 * cosSqAlpha * (4 + FlatteningFactor*(4-3*cosSqAlpha))
		lambdaP = lambda
		lambda = l + (1-c)*FlatteningFactor*sinAlpha*(sigma+c*sinSigma*(cos2SigmaM+c*cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)))
	}

	if iterLimit == 0 {
		return -1, -1, errors.New("vincenty formula failed to converge") // formula failed to converge
	}

	uSq := cosSqAlpha * (SemiMajorAxis*SemiMajorAxis - SemiMinorAxis*SemiMinorAxis) / (SemiMinorAxis * SemiMinorAxis)
	a := 1 + uSq/16384*(4096+uSq*(-768+uSq*(320-175*uSq)))
	b := uSq / 1024 * (256 + uSq*(-128+uSq*(74-47*uSq)))
	deltaSigma := b * sinSigma * (cos2SigmaM + b/4*(cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)-b/6*cos2SigmaM*(-3+4*sinSigma*sinSigma)*(-3+4*cos2SigmaM*cos2SigmaM)))

	meters := SemiMinorAxis * a * (sigma - deltaSigma)
	kilometers := meters / 1000
	miles := kilometers * 0.621371

	return miles, kilometers, nil
}
