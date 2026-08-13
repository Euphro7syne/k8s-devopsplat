package workload

import (
	"context"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
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

func (s *Service) RestartPod(ctx context.Context, namespace, name string, confirm bool) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	if !confirm {
		return OperationResult{}, apperrors.New(apperrors.CodeInvalidArgument, "confirm=true is required", http.StatusBadRequest)
	}
	pod, err := s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get pod before restart failed")
	}
	controller := metav1.GetControllerOf(pod)
	if controller == nil {
		return OperationResult{}, apperrors.New(apperrors.CodeConflict, "pod is not managed by a restartable workload controller", http.StatusConflict)
	}
	if controller.Kind == "Job" {
		return OperationResult{}, apperrors.New(apperrors.CodeConflict, "job pods cannot be restarted safely; deleting a pod does not rerun a completed job", http.StatusConflict)
	}
	if !restartablePodController(controller.Kind) {
		return OperationResult{}, apperrors.New(apperrors.CodeConflict, "pod is not managed by a restartable workload controller", http.StatusConflict)
	}
	if err := s.client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return OperationResult{}, mapKubernetesError(err, "restart pod failed")
	}
	return OperationResult{
		Kind:      "Pod",
		Namespace: namespace,
		Name:      name,
		Operation: "restart",
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

func (s *Service) ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	if replicas < 0 || replicas > 100 {
		return OperationResult{}, apperrors.New(apperrors.CodeInvalidArgument, "replicas must be between 0 and 100", http.StatusBadRequest)
	}
	item, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get statefulset failed")
	}
	item.Spec.Replicas = &replicas
	updated, err := s.client.AppsV1().StatefulSets(namespace).Update(ctx, item, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "scale statefulset failed")
	}
	summary := resources.StatefulSetSummaryFrom(*updated)
	return OperationResult{
		Kind:        "StatefulSet",
		Namespace:   namespace,
		Name:        name,
		Operation:   "scale",
		UpdatedAt:   time.Now().UTC(),
		StatefulSet: &summary,
	}, nil
}

func (s *Service) RestartStatefulSet(ctx context.Context, namespace, name string) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	item, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get statefulset failed")
	}
	if item.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType {
		return OperationResult{}, apperrors.New(apperrors.CodeConflict, "statefulset uses OnDelete update strategy; restart its managed pods individually", http.StatusConflict)
	}
	if item.Spec.Template.Annotations == nil {
		item.Spec.Template.Annotations = map[string]string{}
	}
	now := time.Now().UTC()
	item.Spec.Template.Annotations["ops.platform/restarted-at"] = now.Format(time.RFC3339)
	updated, err := s.client.AppsV1().StatefulSets(namespace).Update(ctx, item, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "restart statefulset failed")
	}
	summary := resources.StatefulSetSummaryFrom(*updated)
	return OperationResult{
		Kind:        "StatefulSet",
		Namespace:   namespace,
		Name:        name,
		Operation:   "restart",
		UpdatedAt:   now,
		StatefulSet: &summary,
	}, nil
}

func (s *Service) RestartDaemonSet(ctx context.Context, namespace, name string) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	item, err := s.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get daemonset failed")
	}
	if item.Spec.UpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		return OperationResult{}, apperrors.New(apperrors.CodeConflict, "daemonset uses OnDelete update strategy; restart its managed pods individually", http.StatusConflict)
	}
	if item.Spec.Template.Annotations == nil {
		item.Spec.Template.Annotations = map[string]string{}
	}
	now := time.Now().UTC()
	item.Spec.Template.Annotations["ops.platform/restarted-at"] = now.Format(time.RFC3339)
	updated, err := s.client.AppsV1().DaemonSets(namespace).Update(ctx, item, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "restart daemonset failed")
	}
	summary := resources.DaemonSetSummaryFrom(*updated)
	return OperationResult{
		Kind:      "DaemonSet",
		Namespace: namespace,
		Name:      name,
		Operation: "restart",
		UpdatedAt: now,
		DaemonSet: &summary,
	}, nil
}

func (s *Service) SetCronJobSuspend(ctx context.Context, namespace, name string, suspend, confirm bool) (OperationResult, error) {
	if s.client == nil {
		return OperationResult{}, resourcesUnavailable()
	}
	if !confirm {
		return OperationResult{}, apperrors.New(apperrors.CodeInvalidArgument, "confirm=true is required", http.StatusBadRequest)
	}
	item, err := s.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "get cronjob failed")
	}
	item.Spec.Suspend = &suspend
	updated, err := s.client.BatchV1().CronJobs(namespace).Update(ctx, item, metav1.UpdateOptions{})
	if err != nil {
		return OperationResult{}, mapKubernetesError(err, "update cronjob suspend state failed")
	}
	summary := resources.CronJobSummaryFrom(*updated)
	operation := "resume"
	if suspend {
		operation = "suspend"
	}
	return OperationResult{
		Kind:      "CronJob",
		Namespace: namespace,
		Name:      name,
		Operation: operation,
		UpdatedAt: time.Now().UTC(),
		CronJob:   &summary,
	}, nil
}

func restartablePodController(kind string) bool {
	switch kind {
	case "ReplicaSet", "StatefulSet", "DaemonSet":
		return true
	default:
		return false
	}
}
