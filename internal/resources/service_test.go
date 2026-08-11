package resources

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
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

func TestUpdateResourceYAMLUpdatesDeployment(t *testing.T) {
	initialReplicas := int32(1)
	updatedReplicas := int32(3)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &initialReplicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api", Image: "example/api:v1"}},
					},
				},
			},
		},
	)
	service := NewService(client, "test")

	raw, err := yaml.Marshal(&appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &updatedReplicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "api", Image: "example/api:v2"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal deployment yaml: %v", err)
	}

	result, err := service.UpdateResourceYAML(context.Background(), ResourceYAMLUpdateRequest{
		Kind:      "deployment",
		Namespace: "default",
		Name:      "api",
		YAML:      string(raw),
	})
	if err != nil {
		t.Fatalf("update yaml: %v", err)
	}
	if result.Kind != "Deployment" || result.Operation != "update" {
		t.Fatalf("unexpected update result: %#v", result)
	}
	deployment, err := client.AppsV1().Deployments("default").Get(context.Background(), "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get updated deployment: %v", err)
	}
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != updatedReplicas {
		t.Fatalf("expected replicas %d, got %#v", updatedReplicas, deployment.Spec.Replicas)
	}
	if got := deployment.Spec.Template.Spec.Containers[0].Image; got != "example/api:v2" {
		t.Fatalf("expected updated image, got %s", got)
	}
}

func TestUpdateResourceYAMLRejectsConfigMap(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
			ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"},
			Data:       map[string]string{"mode": "old"},
		},
	)
	service := NewService(client, "test")
	raw, err := yaml.Marshal(&corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"},
		Data:       map[string]string{"mode": "new"},
	})
	if err != nil {
		t.Fatalf("marshal configmap yaml: %v", err)
	}

	_, err = service.UpdateResourceYAML(context.Background(), ResourceYAMLUpdateRequest{
		Kind:      "configmap",
		Namespace: "default",
		Name:      "app-config",
		YAML:      string(raw),
	})
	if err == nil {
		t.Fatalf("expected configmap yaml update to be rejected")
	}
}
