package v1_test

import (
	"strings"
	"testing"

	v1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	tcs := []struct {
		name     string
		val      string
		expected v1.Config
	}{
		{
			name: "happy path",
			val: `{ 
  "imagePullSecretName": "ipsn2",
  "imagePullSecretNamespace": "ipsns2",
  "secretSyncInterval": "10m",
  "overrides": { "rn2": "orn2" }
}`,
			expected: v1.Config{
				Overrides: map[string]string{
					"rn2": "orn2",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.val)
			actual, err := v1.NewConfig(r)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, *actual)
		})
	}
}

func TestNewConfig_featureValidation(t *testing.T) {
	base := `{"overrides":{"x":"y"}`
	minAll := base + `,"availableFeatures":["` + v1.AnnotationAlterImgRegistry + `","` + v1.AnnotationSetPullSecret + `"],"namespaceFeatures":{"ns":["` + v1.AnnotationAll + `"]}}`

	tcs := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name: "all with availableFeatures expands valid",
			json: minAll,
		},
		{
			name: "legacy explicit features without availableFeatures",
			json: base + `,"namespaceFeatures":{"ns":["` + v1.AnnotationAlterImgRegistry + `"]}}`,
		},
		{
			name:    "unknown in availableFeatures",
			json:    base + `,"availableFeatures":["rt-cfg.kyma-project.io/unknown"],"namespaceFeatures":{"ns":["` + v1.AnnotationAlterImgRegistry + `"]}}`,
			wantErr: "unknown feature",
		},
		{
			name: "all without availableFeatures",
			json: base + `,"namespaceFeatures":{"ns":["` + v1.AnnotationAll + `"]}}`,
		},
		{
			name: "all with empty availableFeatures array",
			json: base + `,"availableFeatures":[],"namespaceFeatures":{"ns":["` + v1.AnnotationAll + `"]}}`,
		},
		{
			name: "empty availableFeatures when key present",
			json: base + `,"availableFeatures":[]}`,
		},
		{
			name: "explicit namespace feature not in availableFeatures when set",
			json: base + `,"availableFeatures":["` + v1.AnnotationAlterImgRegistry + `"],"namespaceFeatures":{"ns":["` +
				v1.AnnotationAlterImgRegistry + `","` + v1.AnnotationSetPullSecret + `"]}}`,
			wantErr: "not in availableFeatures",
		},
		{
			name:    "unknown token in namespaceFeatures",
			json:    base + `,"namespaceFeatures":{"ns":["not-a-feature"]}}`,
			wantErr: "unknown feature",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.json)
			_, err := v1.NewConfig(r)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestConfig_NamespaceDefaultFeatures(t *testing.T) {
	nsFeatures := v1.NamespaceFeatures{
		"kyma-system": {v1.AnnotationAll, v1.AnnotationAlterImgRegistry},
		"other":       {v1.AnnotationSetPullSecret},
	}
	cfg := &v1.Config{
		NamespaceFeatures: &nsFeatures,
		AvailableFeatures: []string{v1.AnnotationAlterImgRegistry, v1.AnnotationSetPullSecret},
	}

	got := cfg.NamespaceDefaultFeatures("kyma-system")
	assert.Equal(t, map[string]string{
		v1.AnnotationAlterImgRegistry: "true",
		v1.AnnotationSetPullSecret:    "true",
	}, got)

	assert.Equal(t, map[string]string{v1.AnnotationSetPullSecret: "true"}, cfg.NamespaceDefaultFeatures("other"))
	assert.Empty(t, cfg.NamespaceDefaultFeatures("missing"))
}

func TestConfig_NamespaceDefaultFeatures_nilNamespaceFeatures(t *testing.T) {
	cfg := &v1.Config{Overrides: map[string]string{"a": "b"}}
	assert.Empty(t, cfg.NamespaceDefaultFeatures("ns"))
}

func TestConfig_ExpandAnnotationAll(t *testing.T) {
	cfg := &v1.Config{
		Overrides:         map[string]string{"x": "y"},
		AvailableFeatures: []string{v1.AnnotationAlterImgRegistry, v1.AnnotationSetPullSecret},
	}

	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, cfg.ExpandAnnotationAll(nil))
	})
	t.Run("no all key", func(t *testing.T) {
		in := map[string]string{v1.AnnotationAlterImgRegistry: "true"}
		assert.Equal(t, in, cfg.ExpandAnnotationAll(in))
	})
	t.Run("all true expands", func(t *testing.T) {
		got := cfg.ExpandAnnotationAll(map[string]string{v1.AnnotationAll: "true"})
		assert.Equal(t, map[string]string{
			v1.AnnotationAlterImgRegistry: "true",
			v1.AnnotationSetPullSecret:    "true",
		}, got)
	})
	t.Run("all with explicit override preserved", func(t *testing.T) {
		got := cfg.ExpandAnnotationAll(map[string]string{
			v1.AnnotationAll:              "true",
			v1.AnnotationAlterImgRegistry: "false",
		})
		assert.Equal(t, map[string]string{
			v1.AnnotationAlterImgRegistry: "false",
			v1.AnnotationSetPullSecret:    "true",
		}, got)
	})
	t.Run("all not true unchanged", func(t *testing.T) {
		in := map[string]string{v1.AnnotationAll: "false"}
		assert.Equal(t, in, cfg.ExpandAnnotationAll(in))
	})
}

func TestConfig_ExpandAnnotationAll_PreservesCTBValues(t *testing.T) {
	cfg := &v1.Config{
		Overrides: map[string]string{"x": "y"},
		AvailableFeatures: []string{
			v1.AnnotationAlterImgRegistry,
			v1.AnnotationSetPullSecret,
			v1.AnnotationAddClusterTrustBundle,
		},
	}

	t.Run("restart-on-change preserved over all expansion", func(t *testing.T) {
		got := cfg.ExpandAnnotationAll(map[string]string{
			v1.AnnotationAll:                "true",
			v1.AnnotationAddClusterTrustBundle: "restart-on-change",
		})
		assert.Equal(t, "restart-on-change", got[v1.AnnotationAddClusterTrustBundle])
		assert.Equal(t, "true", got[v1.AnnotationAlterImgRegistry])
		assert.Equal(t, "true", got[v1.AnnotationSetPullSecret])
	})

	t.Run("false preserved over all expansion", func(t *testing.T) {
		got := cfg.ExpandAnnotationAll(map[string]string{
			v1.AnnotationAll:                "true",
			v1.AnnotationAddClusterTrustBundle: "false",
		})
		assert.Equal(t, "false", got[v1.AnnotationAddClusterTrustBundle])
		assert.Equal(t, "true", got[v1.AnnotationAlterImgRegistry])
	})
}
