/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func TestParseNamespaceList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "empty string yields no namespaces", input: "", expected: nil},
		{name: "single namespace", input: "team-a", expected: []string{"team-a"}},
		{name: "multiple namespaces", input: "team-a,team-b", expected: []string{"team-a", "team-b"}},
		{name: "whitespace is trimmed", input: " team-a , team-b ", expected: []string{"team-a", "team-b"}},
		{name: "empty entries are dropped", input: "team-a,,team-b,", expected: []string{"team-a", "team-b"}},
		{name: "only separators yields no namespaces", input: ", ,", expected: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseNamespaceList(tc.input))
		})
	}
}

func TestSecretCacheOptions(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		wantScoped bool
		wantNs     []string
	}{
		{
			name:       "no namespaces keeps the default cluster-wide cache",
			namespaces: nil,
			wantScoped: false,
		},
		{
			name:       "namespaces scope only the Secrets cache",
			namespaces: []string{"team-a", "team-b"},
			wantScoped: true,
			wantNs:     []string{"team-a", "team-b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := secretCacheOptions(tc.namespaces)

			if !tc.wantScoped {
				// Zero options == the manager default (cluster-wide caching),
				// proving the default behaviour is unchanged when the flag is
				// not set.
				assert.Equal(t, cache.Options{}, opts,
					"cache options must be the zero value when no namespaces are configured")
				return
			}

			assert.Len(t, opts.ByObject, 1, "only the Secrets object should be scoped")
			// ByObject keys are compared by type at runtime, so look the entry
			// up by asserting on the key's dynamic type instead of a map index.
			var byObject cache.ByObject
			var found bool
			for obj, scoped := range opts.ByObject {
				if _, ok := obj.(*corev1.Secret); ok {
					byObject = scoped
					found = true
				}
			}
			if !found {
				t.Fatal("expected a ByObject entry for *corev1.Secret")
			}
			assert.Len(t, byObject.Namespaces, len(tc.wantNs))
			for _, ns := range tc.wantNs {
				_, ok := byObject.Namespaces[ns]
				assert.True(t, ok, "expected namespace %q to be in the Secrets cache scope", ns)
			}
		})
	}
}
