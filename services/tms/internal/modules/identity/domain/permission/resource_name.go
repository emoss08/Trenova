package permission

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
