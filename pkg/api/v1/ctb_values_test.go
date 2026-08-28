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
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, v1.CTBRestartEnabled(tc.annotations))
		})
	}
}
