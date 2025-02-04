package util

import (
	"github.com/agnivade/levenshtein"
	"strings"
)

func IsStringInSlice(a string, b []string) bool {
	if len(b) == 0 {
		return false
	}
	for _, i := range b {
		if strings.Contains(i, a) {
			return true
		}
	}
	return false
}

func SimilarityScore(text1 string, text2 string) float64 {
	// Calculate the Levenshtein distance.
	distance := levenshtein.ComputeDistance(text1, text2)
	// Calculate the maximum length between the two strings.
	maxLength := max(len(text1), len(text2))
	// Calculate the similarity score as a percentage.
	similarity := (1.0 - float64(distance)/float64(maxLength)) * 100.0
	return similarity
}
