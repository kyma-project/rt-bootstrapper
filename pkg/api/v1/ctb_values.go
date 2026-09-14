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

// CTBHashPresent returns true if the pod carries the ctb-hash annotation,
// indicating it was mutated by the CTB webhook (regardless of opt-in source).
func CTBHashPresent(annotations map[string]string) bool {
	_, ok := annotations[AnnotationCTBHash]
	return ok
}

// CTBRestartEnabled returns true if the pod should be restarted when the
// CTB CA changes. A pod is eligible when it carries EITHER the
// add-cluster-trust-bundle: "true" annotation OR the ctb-hash annotation
// (stamped by the webhook on every CTB-opted-in pod).
func CTBRestartEnabled(annotations map[string]string) bool {
	if CTBHashPresent(annotations) {
		return true
	}
	v, ok := annotations[AnnotationAddClusterTrustBundle]
	if !ok {
		return false
	}
	return v == CTBValueTrue
}
