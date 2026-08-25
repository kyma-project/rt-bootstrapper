package v1

const (
	// CTBValueTrue enables CTB volume mounting and controller-managed restart.
	CTBValueTrue = "true"
	// CTBValueFalse explicitly opts out of CTB volume mounting and restart.
	CTBValueFalse = "false"
)

// CTBMountEnabled returns true if the annotation value signals that the
// ClusterTrustBundle projected volume should be mounted.
func CTBMountEnabled(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	if !ok {
		return false
	}
	return v == CTBValueTrue
}

// CTBExplicitOptOut returns true if the pod explicitly opts out via "false".
func CTBExplicitOptOut(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	return ok && v == CTBValueFalse
}

// CTBRestartEnabled returns true if the annotation value signals that the
// pod should be restarted when the CTB CA changes.
// Only "true" enables restart; "false" and any unknown value do not.
func CTBRestartEnabled(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	if !ok {
		return false
	}
	return v == CTBValueTrue
}
