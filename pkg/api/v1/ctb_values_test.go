package v1_test

import (
	"testing"

	v1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestCTBMountEnabled(t *testing.T) {
	tcs := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{name: "absent", annotations: map[string]string{}, want: false},
		{name: "true", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "true"}, want: true},
		{name: "false", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "false"}, want: false},
		{name: "restart-on-change (removed)", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "restart-on-change"}, want: false},
		{name: "unknown value", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "garbage"}, want: false},
		{name: "nil map", annotations: nil, want: false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, v1.CTBMountEnabled(tc.annotations))
		})
	}
}

func TestCTBHashPresent(t *testing.T) {
	tcs := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{name: "ctb-hash present", annotations: map[string]string{v1.AnnotationCTBHash: "abc123"}, want: true},
		{name: "ctb-hash absent", annotations: map[string]string{}, want: false},
		{name: "nil annotations", annotations: nil, want: false},
		{name: "other annotations only", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "true"}, want: false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, v1.CTBHashPresent(tc.annotations))
		})
	}
}

func TestCTBRestartEnabled(t *testing.T) {
	tcs := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{name: "absent", annotations: map[string]string{}, want: false},
		{name: "true", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "true"}, want: true},
		{name: "false", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "false"}, want: false},
		{name: "restart-on-change (removed)", annotations: map[string]string{v1.AnnotationAddClusterTrustBundle: "restart-on-change"}, want: false},
		{name: "nil map", annotations: nil, want: false},
		{name: "ctb-hash only (namespace opt-in)", annotations: map[string]string{v1.AnnotationCTBHash: "somehash"}, want: true},
		{name: "ctb-hash and add-cluster-trust-bundle true", annotations: map[string]string{v1.AnnotationCTBHash: "somehash", v1.AnnotationAddClusterTrustBundle: "true"}, want: true},
		{name: "neither annotation", annotations: map[string]string{"unrelated": "value"}, want: false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, v1.CTBRestartEnabled(tc.annotations))
		})
	}
}
