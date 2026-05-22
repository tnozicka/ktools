package collect

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsPublicSecretKey(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "ca.crt is public",
			key:      "ca.crt",
			expected: true,
		},
		{
			name:     "tls.crt is public",
			key:      "tls.crt",
			expected: true,
		},
		{
			name:     "service-ca.crt is public",
			key:      "service-ca.crt",
			expected: true,
		},
		{
			name:     "tls.key is not public",
			key:      "tls.key",
			expected: false,
		},
		{
			name:     "arbitrary key is not public",
			key:      "password",
			expected: false,
		},
		{
			name:     "empty key is not public",
			key:      "",
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isPublicSecretKey(tc.key)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("expected and got differ:\n%s", cmp.Diff(tc.expected, got))
			}
		})
	}
}
