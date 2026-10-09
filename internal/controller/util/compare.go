package util

func IsStringInSlice(a string, b []string) bool {
	if len(b) == 0 {
		return false
	}
	for _, i := range b {
		if i == a {
			return true
		}
	}
	return false
}
