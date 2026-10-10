package cachescope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestParse(t *testing.T) {
	cases := map[string][]string{
		"":                     nil,
		"   ":                  nil,
		"hermes-ck":            {"hermes-ck"},
		" hermes-ck , other ":  {"hermes-ck", "other"},
		"hermes-ck,,hermes-ck": {"hermes-ck"},
		",":                    nil,
	}
	for in, want := range cases {
		assert.Equal(t, want, Parse(in), "input %q", in)
	}
}

func TestOptions_EmptyKeepsClusterWideDefault(t *testing.T) {
	o := Options(nil)
	assert.Empty(t, o.ByObject)
	assert.Empty(t, o.DefaultNamespaces)
}

func TestOptions_ScopesOnlySecrets(t *testing.T) {
	o := Options([]string{"hermes-ck"})
	assert.Empty(t, o.DefaultNamespaces, "only Secrets are scoped; other kinds keep their cluster-wide cache")
	require.Len(t, o.ByObject, 1)
	var found bool
	for obj, cfg := range o.ByObject {
		if _, ok := obj.(*corev1.Secret); ok {
			found = true
			assert.Len(t, cfg.Namespaces, 1)
			assert.Contains(t, cfg.Namespaces, "hermes-ck")
		}
		var _ client.Object = obj
	}
	assert.True(t, found, "Secret must be scoped")
}

func TestOptions_MultipleNamespaces(t *testing.T) {
	o := Options([]string{"a", "b"})
	for obj, cfg := range o.ByObject {
		if _, ok := obj.(*corev1.Secret); ok {
			assert.Len(t, cfg.Namespaces, 2)
		}
	}
}
