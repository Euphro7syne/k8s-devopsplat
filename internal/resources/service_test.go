package resources

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPodsMarksAbnormalPod(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "bad-pod", Namespace: "default"},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "app",
						Ready: false,
						State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
					},
				},
			},
		},
	)
	service := NewService(client, "test")

	result, err := service.ListPods(context.Background(), "default", ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list pods: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one pod, got %d", result.Total)
	}
	if result.Items[0].Status != "CrashLoopBackOff" {
		t.Fatalf("expected CrashLoopBackOff, got %s", result.Items[0].Status)
	}
}

func TestListDeployments(t *testing.T) {
	replicas := int32(2)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api", Image: "example/api:v1"}},
					},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
		},
	)
	service := NewService(client, "test")

	result, err := service.ListDeployments(context.Background(), "default", ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one deployment, got %d", result.Total)
	}
	if result.Items[0].Images[0] != "example/api:v1" {
		t.Fatalf("unexpected image: %s", result.Items[0].Images[0])
	}
}
