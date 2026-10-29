package permission

import "github.com/emoss08/trenova/shared/textcase"

const maxResourceNameLength = 64

func LooksLikeResource(name string) bool {
	if name == "" || len(name) > maxResourceNameLength {
		return false
	}
	for idx := range len(name) {
		char := name[idx]
		if (char < 'a' || char > 'z') && char != '_' {
			return false
		}
	}
	return name[0] != '_' && name[len(name)-1] != '_'
}

// Noun is how one of the resource's records is named in a sentence: "worker
// credential", "shipment".
func (r Resource) Noun() string {
	return textcase.HumanizeSnakeCasePhrase(string(r))
}
