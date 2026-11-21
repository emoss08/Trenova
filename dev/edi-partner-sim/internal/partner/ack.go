package partner

import "math/rand/v2"

func (a *Acknowledgment) Silent(rng *rand.Rand) bool {
	return a.Sends == AckNone || a.Never > 0 && rng.Float64() < a.Never
}
