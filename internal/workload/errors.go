package workload

import (
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	apperrors "ops-platform/internal/pkg/errors"
)

func resourcesUnavailable() *apperrors.AppError {
	return apperrors.New(apperrors.CodeKubernetesUnavailable, "kubernetes client unavailable", http.StatusServiceUnavailable)
}

func mapKubernetesError(err error, message string) error {
	if err == nil {
		return nil
	}
	switch {
	case apierrors.IsNotFound(err):
		return apperrors.Wrap(err, apperrors.CodeNotFound, message, http.StatusNotFound)
	case apierrors.IsForbidden(err):
		return apperrors.Wrap(err, apperrors.CodePermissionDenied, "kubernetes permission denied", http.StatusForbidden)
	default:
		return apperrors.Wrap(err, apperrors.CodeKubernetesUnavailable, message, http.StatusServiceUnavailable)
	}
}
