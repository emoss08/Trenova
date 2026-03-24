package dbhelper

func WrapWildcard(query string) string {
	return "%" + query + "%"
}
