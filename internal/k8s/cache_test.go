package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResourceCacheSyncsSelectedResourcesAndWatchesUpdates(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "demo"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "demo"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "demo"}},
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "local-path"}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "must-not-be-cached", Namespace: "demo"},
			Data:       map[string][]byte{"password": []byte("plaintext")},
		},
	)
	cache := NewResourceCache(client, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := cache.Start(ctx, 3*time.Second); err != nil {
		t.Fatalf("start resource cache: %v", err)
	}
	if !cache.Ready() {
		t.Fatalf("expected resource cache to be ready")
	}

	if namespaces, err := cache.ListNamespaces(); err != nil || len(namespaces) != 1 || namespaces[0].Name != "demo" {
		t.Fatalf("unexpected cached namespaces: %#v, err=%v", namespaces, err)
	}
	if pod, err := cache.GetPod("demo", "pod-a"); err != nil || pod.Name != "pod-a" {
		t.Fatalf("unexpected cached pod: %#v, err=%v", pod, err)
	}
	if deployments, err := cache.ListDeployments("demo"); err != nil || len(deployments) != 1 || deployments[0].Name != "app" {
		t.Fatalf("unexpected cached deployments: %#v, err=%v", deployments, err)
	}
	if storageClass, err := cache.GetStorageClass("local-path"); err != nil || storageClass.Name != "local-path" {
		t.Fatalf("unexpected cached storage class: %#v, err=%v", storageClass, err)
	}

	created, err := client.AppsV1().Deployments("demo").Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "watched", Namespace: "demo", Labels: map[string]string{"version": "v1"}},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create watched deployment: %v", err)
	}
	waitForCache(t, func() bool {
		item, getErr := cache.GetDeployment("demo", created.Name)
		return getErr == nil && item.Labels["version"] == "v1"
	})

	created.Labels["version"] = "v2"
	if _, err := client.AppsV1().Deployments("demo").Update(ctx, created, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update watched deployment: %v", err)
	}
	waitForCache(t, func() bool {
		item, getErr := cache.GetDeployment("demo", created.Name)
		return getErr == nil && item.Labels["version"] == "v2"
	})

	for _, action := range client.Actions() {
		if action.GetResource().Resource == "secrets" && (action.GetVerb() == "list" || action.GetVerb() == "watch") {
			t.Fatalf("Secret must not have a shared informer action: %s %s", action.GetVerb(), action.GetResource().Resource)
		}
	}

	cancel()
	waitForCache(t, func() bool { return !cache.Ready() })
}

func waitForCache(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for informer cache condition")
}
