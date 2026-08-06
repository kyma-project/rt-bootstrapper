package v1

const (
	// CTBValueTrue enables CTB volume mounting (existing behavior).
	CTBValueTrue = "true"
	// CTBValueFalse explicitly opts out of CTB volume mounting.
	CTBValueFalse = "false"
	// CTBValueRestartOnChange enables CTB mounting and controller-managed restart.
	CTBValueRestartOnChange = "restart-on-change"
)

// CTBMountEnabled returns true if the annotation value signals that the
// ClusterTrustBundle projected volume should be mounted.
func CTBMountEnabled(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	if !ok {
		return false
	}
	return v == CTBValueTrue || v == CTBValueRestartOnChange
}

// CTBExplicitOptOut returns true if the pod explicitly opts out via "false".
func CTBExplicitOptOut(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	return ok && v == CTBValueFalse
}

// CTBRestartEnabled returns true if the annotation value signals that the
// pod should be restarted when the CTB CA changes.
func CTBRestartEnabled(annotations map[string]string) bool {
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	if !ok {
		return false
	}
	return v == CTBValueRestartOnChange
}
