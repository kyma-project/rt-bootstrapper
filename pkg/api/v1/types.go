package v1

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kyma-project/rt-bootstrapper/internal/webhook/k8s"
)

const (
	AnnotationAlterImgRegistry      = "rt-cfg.kyma-project.io/alter-img-registry"
	AnnotationSetPullSecret         = "rt-cfg.kyma-project.io/add-img-pull-secret"
	AnnotationAddClusterTrustBundle = "rt-cfg.kyma-project.io/add-cluster-trust-bundle"
	AnnotationSetFipsMode           = "rt-cfg.kyma-project.io/set-fips-mode"
	AnnotationDefaulted             = "rt-bootstrapper.kyma-project.io/defaulted"
	FiledManager                    = "rt-bootstrapper"
	EnvKymaFipsModeEnabled          = "KYMA_FIPS_MODE_ENABLED"
)

type NamespaceFeatures map[string][]string

func (f NamespaceFeatures) Features(nsName string) map[string]string {
	featureList, found := f[nsName]
	if !found {
		return map[string]string{}
	}

	result := make(map[string]string, len(featureList))
	for _, feature := range featureList {
		result[feature] = "true"
	}
	return result
}

type Config struct {
	Overrides                 map[string]string       `json:"overrides" validate:"required"`
	ImagePullSecretName       string                  `json:"imagePullSecretName" validate:"required"`
	ImagePullSecretNamespace  string                  `json:"imagePullSecretNamespace" validate:"required"`
	SecretSyncInterval        Duration                `json:"secretSyncInterval" validate:"required"`
	ClusterTrustBundleMapping *k8s.ClusterTrustBundle `json:"clusterTrustBundle,omitempty"`
	NamespaceFeatures         *NamespaceFeatures      `json:"namespaceFeatures,omitempty"`
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
	return &out, validate.Struct(out)

}
