package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kyma-project/rt-bootstrapper/internal/webhook/k8s"
)

const (
	AnnotationAlterImgRegistry      = "rt-cfg.kyma-project.io/alter-img-registry"
	AnnotationSetPullSecret         = "rt-cfg.kyma-project.io/add-img-pull-secret"
	AnnotationAddClusterTrustBundle = "rt-cfg.kyma-project.io/add-cluster-trust-bundle"
	AnnotationSetFipsMode           = "rt-cfg.kyma-project.io/set-fips-mode"
	AnnotationSetLandscape          = "rt-cfg.kyma-project.io/set-landscape"
	AnnotationAll                   = "rt-cfg.kyma-project.io/all"
	AnnotationModified              = "rt-bootstrapper.kyma-project.io/modified"
	FiledManager                    = "rt-bootstrapper"
	EnvKymaFipsModeEnabled          = "KYMA_FIPS_MODE_ENABLED"
	EnvFipsModeEnabled              = "FIPS_MODE_ENABLED"
	EnvKymaLandscape                = "KYMA_LANDSCAPE"
	ConfigMapKey                    = "rt-bootstrapper-config.json"
)

// KnownFeatureKeys lists valid rt-cfg feature annotation keys (excluding AnnotationAll).
var KnownFeatureKeys = []string{
	AnnotationAlterImgRegistry,
	AnnotationSetPullSecret,
	AnnotationAddClusterTrustBundle,
	AnnotationSetFipsMode,
	AnnotationSetLandscape,
}

type NamespaceFeatures map[string][]string

type Config struct {
	Overrides                 map[string]string       `json:"overrides" validate:"required"`
	ClusterTrustBundleMapping *k8s.ClusterTrustBundle `json:"clusterTrustBundle,omitempty"`
	NamespaceFeatures         *NamespaceFeatures      `json:"namespaceFeatures,omitempty"`
	AvailableFeatures         []string                `json:"availableFeatures,omitempty"` // Empty or omitted: no customer-catalog features; see NewConfig logging.
}

// NamespaceDefaultFeatures returns namespace-scoped default feature annotations for a
// namespace, expanding AnnotationAll using AvailableFeatures.
func (c *Config) NamespaceDefaultFeatures(nsName string) map[string]string {
	if c.NamespaceFeatures == nil {
		return map[string]string{}
	}
	featureList, found := (*c.NamespaceFeatures)[nsName]
	if !found {
		return map[string]string{}
	}

	result := make(map[string]string)
	for _, feature := range featureList {
		if feature == AnnotationAll {
			for _, af := range c.AvailableFeatures {
				result[af] = "true"
			}
			continue
		}
		result[feature] = "true"
	}
	return result
}

// ExpandAnnotationAll returns a copy of annotations where rt-cfg.kyma-project.io/all: "true"
// is expanded to each key in AvailableFeatures with value "true". Other keys are kept;
// an explicit annotation for a feature is not overwritten by the expansion.
func (c *Config) ExpandAnnotationAll(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}
	if annotations[AnnotationAll] != "true" {
		return annotations
	}
	out := make(map[string]string, len(annotations)+len(c.AvailableFeatures))
	for k, v := range annotations {
		if k == AnnotationAll {
			continue
		}
		out[k] = v
	}
	for _, af := range c.AvailableFeatures {
		if _, ok := out[af]; !ok {
			out[af] = "true"
		}
	}
	return out
}

// ErrFeatureValidationFailed is returned when a config feature name is invalid or inconsistent.
var ErrFeatureValidationFailed = errors.New("feature validation failed")

func sanitizeAvailableFeatures(availableFeatures []string, known []string) ([]string, error) {
	if len(availableFeatures) == 0 {
		slog.Warn("rt-bootstrapper config: availableFeatures is empty or omitted; no features are allowed from the customer catalog")
		return []string{}, nil
	}

	for _, f := range availableFeatures {
		if !slices.Contains(known, f) {
			return nil, fmt.Errorf("%w: unknown feature %q", ErrFeatureValidationFailed, f)
		}
	}
	return availableFeatures, nil
}

func validateFeatureAliases(c *Config, known []string) error {
	availableFeatures, err := sanitizeAvailableFeatures(c.AvailableFeatures, known)
	if err != nil {
		return err
	}

	if c.NamespaceFeatures == nil || len(*c.NamespaceFeatures) == 0 {
		slog.Warn("rt-bootstrapper config: namespaceFeatures is empty or omitted; no per-namespace default feature lists are configured")
		return nil
	}

	for _, declaredNsFeatures := range *c.NamespaceFeatures {
		if err := validateNsFeatures(declaredNsFeatures, availableFeatures, known); err != nil {
			return err
		}
	}
	return nil
}

// validateNsFeatures checks one namespace feature list: each entry is
// rt-cfg.kyma-project.io/all or a known feature key; when the catalog is
// non-empty, each concrete entry must be in it. all with an empty catalog is logged and ignored.
func validateNsFeatures(declaredNsFeatures []string, availableFeatures []string, known []string) error {
	for _, entry := range declaredNsFeatures {
		if entry == AnnotationAll {
			if len(availableFeatures) == 0 {
				slog.Warn("rt-bootstrapper config: namespaceFeatures contains rt-cfg.kyma-project.io/all but availableFeatures is empty; the alias has no effect")
			}
			continue
		}
		if !slices.Contains(known, entry) {
			return fmt.Errorf("%w: unknown feature %q", ErrFeatureValidationFailed, entry)
		}
		if len(availableFeatures) > 0 {
			if !slices.Contains(availableFeatures, entry) {
				return fmt.Errorf("%w: feature %q is not in availableFeatures", ErrFeatureValidationFailed, entry)
			}
		}
	}
	return nil
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(p []byte) error {
	var v any
	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func NewConfig(r io.Reader) (*Config, error) {
	var out Config
	err := json.NewDecoder(r).Decode(&out)
	if err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(out); err != nil {
		return nil, err
	}

	if err := validateFeatureAliases(&out, KnownFeatureKeys); err != nil {
		return nil, err
	}
	return &out, nil
}
