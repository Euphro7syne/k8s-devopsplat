package workload

import (
	"context"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/resources"
)

type Service struct {
	client resources.KubernetesClient
}

func NewService(client resources.KubernetesClient) *Service {
	return &Service{client: client}
}

func (s *Service) DeletePod(ctx context.Context, namespace, name string, confirm bool) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	if !confirm {
		return OperationResult{}, apperrors.New(apperrors.CodeInvalidArgument, "confirm=true is required", http.StatusBadRequest)
	}
	if err := s.client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return OperationResult{}, mapKubernetesError(err, "delete pod failed")
	}
	return OperationResult{
		Kind:      "Pod",
		Namespace: namespace,
		Name:      name,
		Operation: "delete",
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	if replicas < 0 || replicas > 100 {
		return OperationResult{}, apperrors.New(apperrors.CodeInvalidArgument, "replicas must be between 0 and 100", http.StatusBadRequest)
	}
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get deployment failed")
	}
	deployment.Spec.Replicas = &replicas
	updated, err := s.client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "scale deployment failed")
	}
	summary := resources.DeploymentSummaryFrom(*updated)
	return OperationResult{
		Kind:       "Deployment",
		Namespace:  namespace,
		Name:       name,
		Operation:  "scale",
		UpdatedAt:  time.Now().UTC(),
		Deployment: &summary,
	}, nil
}

func (s *Service) RestartDeployment(ctx context.Context, namespace, name string) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get deployment failed")
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	now := time.Now().UTC()
	deployment.Spec.Template.Annotations["ops.platform/restarted-at"] = now.Format(time.RFC3339)
	updated, err := s.client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "restart deployment failed")
	}
	summary := resources.DeploymentSummaryFrom(*updated)
	return OperationResult{
		Kind:       "Deployment",
		Namespace:  namespace,
		Name:       name,
		Operation:  "restart",
		UpdatedAt:  now,
		Deployment: &summary,
	}, nil
}
