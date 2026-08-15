package resources

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

type deploymentDetailCache struct {
	ResourceCache
	deployment  *appsv1.Deployment
	replicaSets []appsv1.ReplicaSet
	pods        []corev1.Pod
	events      []corev1.Event
}

func (c *deploymentDetailCache) Ready() bool { return true }

func (c *deploymentDetailCache) GetDeployment(_, _ string) (*appsv1.Deployment, error) {
	return c.deployment, nil
}

func (c *deploymentDetailCache) ListReplicaSets(_ string) ([]appsv1.ReplicaSet, error) {
	return c.replicaSets, nil
}

func (c *deploymentDetailCache) ListPods(_ string) ([]corev1.Pod, error) { return c.pods, nil }

func (c *deploymentDetailCache) ListEvents(_ string) ([]corev1.Event, error) { return c.events, nil }

func TestDeploymentDetailUsesReadyCacheWithoutAPIReads(t *testing.T) {
	controller := true
	cache := &deploymentDetailCache{
		deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "demo", UID: types.UID("deployment-uid")}},
		replicaSets: []appsv1.ReplicaSet{{ObjectMeta: metav1.ObjectMeta{
			Name:      "app-rs",
			Namespace: "demo",
			UID:       types.UID("rs-uid"),
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment", Name: "app", UID: types.UID("deployment-uid"), Controller: &controller,
			}},
		}}},
		pods: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "demo",
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ReplicaSet", Name: "app-rs", UID: types.UID("rs-uid"), Controller: &controller,
			}},
		}}},
	}
	client := fake.NewSimpleClientset()
	detail, err := NewCachedService(client, cache, "test").GetDeployment(context.Background(), "demo", "app")
	if err != nil {
		t.Fatalf("get deployment detail from cache: %v", err)
	}
	if detail.Name != "app" || len(detail.ReplicaSets) != 1 || len(detail.Pods) != 1 {
		t.Fatalf("unexpected deployment detail: %#v", detail)
	}
	if len(client.Actions()) != 0 {
		t.Fatalf("ready cache should avoid Kubernetes API reads: %#v", client.Actions())
	}
}
