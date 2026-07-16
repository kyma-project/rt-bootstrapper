/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	types.NamespacedName
	SecretSyncInterval time.Duration
	GetConfig          func(context.Context) (*apiv1.Config, error)
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;patch

const (
	logID = "log-id"
)

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	cfg, err := r.GetConfig(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to read config: %w", err)
	}
	if !slices.Contains(cfg.AvailableFeatures, apiv1.AnnotationSetPullSecret) {
		return ctrl.Result{}, nil
	}

	isMasterSecretUpdated := req.Namespace == r.Namespace && req.Name == r.Name
	isCredentialsSecretUpdated := (req.Namespace != "" && req.Namespace != r.Namespace) && req.Name == r.Name
	isNamespaceCreated := req.Namespace == ""

	log := slog.Default().With(
		logID, "reconcile",
		"req", req,
		"is-master-secret-updated", isMasterSecretUpdated,
		"is-credentials-secret-updated", isCredentialsSecretUpdated,
		"is-namespace-created", isNamespaceCreated,
		"uuid", uuid.NewString(),
	)

	log.Debug("fetching master-secret", "namespaced-name", r.NamespacedName)

	var masterSecret corev1.Secret
	if err := r.Get(ctx, r.NamespacedName, &masterSecret); err != nil {
		return ctrl.Result{}, err
	}

	if isMasterSecretUpdated {
		log.Debug("attempting to synchronize secrets")

		var namespaceList corev1.NamespaceList
		if err := r.List(ctx, &namespaceList); err != nil {
			return ctrl.Result{}, err
		}

		errors := []string{}
		for _, namespace := range namespaceList.Items {
			// omit master-secret
			if namespace.Namespace == r.Name {
				continue
			}

			credentialsSecret := createCredentialSecret(r.Name, namespace.Name,
				masterSecret.Data[corev1.DockerConfigJsonKey])

			err := r.Patch(ctx, credentialsSecret, client.Apply, &client.PatchOptions{ //nolint:staticcheck
				FieldManager: apiv1.FiledManager,
			})

			if err != nil {
				errors = append(errors, err.Error())
				continue
			}

			log.WithGroup("secret").Debug("secret patched successfully",
				"name", credentialsSecret.Name,
				"namespace", credentialsSecret.Namespace)
		}

		if len(errors) > 0 {
			msg := strings.Join(errors, ",")
			return ctrl.Result{}, fmt.Errorf(
				"failed to reconcile due to patch errors: %s", msg)
		}

		return ctrl.Result{RequeueAfter: r.SecretSyncInterval}, nil
	}

	if isCredentialsSecretUpdated {
		log.Debug("attempting to synchroinize secret")

		credentialsSecret := createCredentialSecret(r.Name, req.Namespace,
			masterSecret.Data[corev1.DockerConfigJsonKey])

		return ctrl.Result{}, r.Patch(ctx, credentialsSecret, client.Apply, &client.PatchOptions{ //nolint:staticcheck
			FieldManager: apiv1.FiledManager,
			Force:        ptr.To(true),
		})
	}

	if isNamespaceCreated {
		log.Debug("attempting to create secret")

		credentialsSecret := createCredentialSecret(r.Name, req.Name,
			masterSecret.Data[corev1.DockerConfigJsonKey])

		return ctrl.Result{}, r.Patch(ctx, credentialsSecret, client.Apply, &client.PatchOptions{ //nolint:staticcheck
			FieldManager: apiv1.FiledManager,
		})
	}

	log.Warn("unhandled request")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	slog.Debug("setting up with manager",
		"master-secert-name", r.Name,
		"master-secret-namespace", r.Namespace)

	p1 := &createNsPredicate{
		log:                      slog.Default().With(logID, "namespace-predicate"),
		masterSecretNamspaceName: r.Namespace,
	}

	p2 := &masterSecret{
		log:            slog.Default().With(logID, "master-secret-predicate"),
		NamespacedName: r.NamespacedName,
	}

	return ctrl.NewControllerManagedBy(mgr).
		Watches(&corev1.Namespace{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(p1)).
		Watches(&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(p2)).
		Named("docker-credentials").
		Complete(r)
}

func createCredentialSecret(name, ns string, data []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: data,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}
}
