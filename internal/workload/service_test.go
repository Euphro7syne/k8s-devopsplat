package workload

import (
	"context"
	"net/http"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	apperrors "ops-platform/internal/pkg/errors"
)

func TestScaleDeployment(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "api:v1"}}},
			},
		},
	})
	service := NewService(client)

	result, err := service.ScaleDeployment(context.Background(), "default", "api", 3)
	if err != nil {
		t.Fatalf("scale deployment: %v", err)
	}
	if result.Deployment == nil || result.Deployment.Replicas != 3 {
		t.Fatalf("expected replicas to be 3, got %#v", result.Deployment)
	}
}

func TestRestartDeploymentAnnotatesTemplate(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "api:v1"}}},
			},
		},
	})
	service := NewService(client)

	if _, err := service.RestartDeployment(context.Background(), "default", "api"); err != nil {
		t.Fatalf("restart deployment: %v", err)
	}
	updated, err := client.AppsV1().Deployments("default").Get(context.Background(), "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if updated.Spec.Template.Annotations["ops.platform/restarted-at"] == "" {
		t.Fatalf("expected restart annotation")
	}
}

func TestScaleAndRestartStatefulSet(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "default"},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "database", Image: "postgres:16-alpine"}}},
			},
		},
	})
	service := NewService(client)

	scaled, err := service.ScaleStatefulSet(context.Background(), "default", "database", 3)
	if err != nil {
		t.Fatalf("scale statefulset: %v", err)
	}
	if scaled.StatefulSet == nil || scaled.StatefulSet.Replicas != 3 {
		t.Fatalf("expected statefulset replicas to be 3, got %#v", scaled.StatefulSet)
	}
	if _, err := service.RestartStatefulSet(context.Background(), "default", "database"); err != nil {
		t.Fatalf("restart statefulset: %v", err)
	}
	updated, err := client.AppsV1().StatefulSets("default").Get(context.Background(), "database", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get statefulset: %v", err)
	}
	if updated.Spec.Template.Annotations["ops.platform/restarted-at"] == "" {
		t.Fatalf("expected statefulset restart annotation")
	}
}

func TestRestartControllerManagedPod(t *testing.T) {
	controller := true
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "api", Controller: &controller},
			},
		},
	})
	service := NewService(client)

	result, err := service.RestartPod(context.Background(), "default", "api-abc", true)
	if err != nil {
		t.Fatalf("restart managed pod: %v", err)
	}
	if result.Operation != "restart" || result.Kind != "Pod" {
		t.Fatalf("unexpected restart result: %#v", result)
	}
	if _, err := client.CoreV1().Pods("default").Get(context.Background(), "api-abc", metav1.GetOptions{}); err == nil {
		t.Fatalf("expected pod to be deleted for controller recreation")
	}
}

func TestRestartPodRejectsMissingConfirmationAndStandalonePod(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "standalone", Namespace: "default"}})
	service := NewService(client)

	_, err := service.RestartPod(context.Background(), "default", "standalone", false)
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeInvalidArgument || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	_, err = service.RestartPod(context.Background(), "default", "standalone", true)
	appErr = apperrors.From(err)
	if appErr.Code != apperrors.CodeConflict || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("expected standalone pod conflict, got %v", err)
	}
}

func TestRestartPodRejectsJobManagedPod(t *testing.T) {
	controller := true
	client := fake.NewSimpleClientset(
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "report", Namespace: "default"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "report-abc", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "batch/v1", Kind: "Job", Name: "report", Controller: &controller}},
			},
		},
	)
	service := NewService(client)

	_, err := service.RestartPod(context.Background(), "default", "report-abc", true)
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeConflict || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("expected job pod conflict, got %v", err)
	}
	if _, getErr := client.CoreV1().Pods("default").Get(context.Background(), "report-abc", metav1.GetOptions{}); getErr != nil {
		t.Fatalf("job pod should not be deleted: %v", getErr)
	}
}

func TestRestartStatefulSetRejectsOnDeleteStrategy(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "default"},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
		},
	})
	service := NewService(client)

	_, err := service.RestartStatefulSet(context.Background(), "default", "database")
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeConflict || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("expected OnDelete strategy conflict, got %v", err)
	}
}

func TestRestartDaemonSetAnnotatesTemplate(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "node-agent", Namespace: "default"},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: "example/agent:v1"}}},
			},
		},
	})
	service := NewService(client)

	result, err := service.RestartDaemonSet(context.Background(), "default", "node-agent")
	if err != nil {
		t.Fatalf("restart daemonset: %v", err)
	}
	if result.DaemonSet == nil || result.Operation != "restart" {
		t.Fatalf("unexpected restart result: %#v", result)
	}
	updated, err := client.AppsV1().DaemonSets("default").Get(context.Background(), "node-agent", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get daemonset: %v", err)
	}
	if updated.Spec.Template.Annotations["ops.platform/restarted-at"] == "" {
		t.Fatalf("expected daemonset restart annotation")
	}
}

func TestRestartDaemonSetRejectsOnDeleteStrategy(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "node-agent", Namespace: "default"},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType},
		},
	})
	service := NewService(client)

	_, err := service.RestartDaemonSet(context.Background(), "default", "node-agent")
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeConflict || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("expected OnDelete strategy conflict, got %v", err)
	}
}

func TestSetCronJobSuspendRequiresConfirmationAndUpdatesSpec(t *testing.T) {
	initial := false
	client := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "report", Namespace: "default"},
		Spec:       batchv1.CronJobSpec{Schedule: "0 * * * *", Suspend: &initial},
	})
	service := NewService(client)

	_, err := service.SetCronJobSuspend(context.Background(), "default", "report", true, false)
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeInvalidArgument || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected confirmation error, got %v", err)
	}

	result, err := service.SetCronJobSuspend(context.Background(), "default", "report", true, true)
	if err != nil {
		t.Fatalf("suspend cronjob: %v", err)
	}
	if result.Operation != "suspend" || result.CronJob == nil || !result.CronJob.Suspend {
		t.Fatalf("unexpected suspend result: %#v", result)
	}
	updated, err := client.BatchV1().CronJobs("default").Get(context.Background(), "report", metav1.GetOptions{})
	if err != nil || updated.Spec.Suspend == nil || !*updated.Spec.Suspend {
		t.Fatalf("cronjob was not suspended: %#v err=%v", updated, err)
	}

	result, err = service.SetCronJobSuspend(context.Background(), "default", "report", false, true)
	if err != nil {
		t.Fatalf("resume cronjob: %v", err)
	}
	if result.Operation != "resume" || result.CronJob == nil || result.CronJob.Suspend {
		t.Fatalf("unexpected resume result: %#v", result)
	}
}
