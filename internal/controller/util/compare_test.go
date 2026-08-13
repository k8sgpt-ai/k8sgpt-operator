package util

import "testing"

func TestIsStringInSliceRequiresExactMatch(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		allowed []string
		want    bool
	}{
		{name: "exact match", kind: "Deployment", allowed: []string{"Pod", "Deployment"}, want: true},
		{name: "partial match is rejected", kind: "Pod", allowed: []string{"PodDisruptionBudget"}, want: false},
		{name: "empty allowlist", kind: "Pod", allowed: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStringInSlice(tt.kind, tt.allowed); got != tt.want {
				t.Fatalf("IsStringInSlice(%q, %v) = %t, want %t", tt.kind, tt.allowed, got, tt.want)
			}
		})
	}
}
