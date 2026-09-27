package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{
			name:     "Contains string",
			slice:    []string{"apple", "banana", "cherry"},
			s:        "banana",
			expected: true,
		},
		{
			name:     "Does not contain string",
			slice:    []string{"apple", "banana", "cherry"},
			s:        "grape",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsString(tt.slice, tt.s)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_PtrBool(t *testing.T) {
	tests := []struct {
		name     string
		b        bool
		expected *bool
	}{
		{
			name:     "Pointer to true",
			b:        true,
			expected: PtrBool(true),
		},
		{
			name:     "Pointer to false",
			b:        false,
			expected: PtrBool(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PtrBool(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
