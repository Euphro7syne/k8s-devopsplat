package workload

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
