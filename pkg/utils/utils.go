package utils

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// PtrBool returns a pointer to a bool.
func PtrBool(b bool) *bool {
	return &b
}
