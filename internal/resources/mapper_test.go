package resources

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

type mapperCache struct {
	ResourceCache
	ready      bool
	replicaSet *appsv1.ReplicaSet
	job        *batchv1.Job
}

func (c *mapperCache) Ready() bool { return c.ready }

func (c *mapperCache) GetReplicaSet(_, _ string) (*appsv1.ReplicaSet, error) {
	return c.replicaSet, nil
}

func (c *mapperCache) GetJob(_, _ string) (*batchv1.Job, error) { return c.job, nil }

func TestResourceMapperUsesCacheForReplicaSetDeploymentChain(t *testing.T) {
	controller := true
	cache := &mapperCache{ready: true, replicaSet: &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-rs",
			Namespace: "demo",
			UID:       types.UID("rs-uid"),
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment", Name: "app", UID: types.UID("deployment-uid"), Controller: &controller,
			}},
		},
	}}
	client := fake.NewSimpleClientset()
	mapper := NewResourceMapper(client, cache)
	resolution := mapper.PodController(context.Background(), corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "app-pod",
		Namespace: "demo",
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "ReplicaSet", Name: "app-rs", UID: types.UID("rs-uid"), Controller: &controller,
		}},
	}})

	if len(resolution.Chain) != 2 || resolution.Chain[1].Kind != "Deployment" || resolution.Chain[1].Name != "app" {
		t.Fatalf("unexpected controller chain: %#v", resolution.Chain)
	}
	if resolution.TopUID != "deployment-uid" {
		t.Fatalf("unexpected top controller UID: %q", resolution.TopUID)
	}
	if len(client.Actions()) != 0 {
		t.Fatalf("cache hit should not call Kubernetes API: %#v", client.Actions())
	}
}

func TestResourceMapperResolvesJobCronJobChain(t *testing.T) {
	controller := true
	cache := &mapperCache{ready: true, job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name:      "backup-123",
		Namespace: "demo",
		UID:       types.UID("job-uid"),
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "CronJob", Name: "backup", UID: types.UID("cronjob-uid"), Controller: &controller,
		}},
	}}}
	resolution := NewResourceMapper(fake.NewSimpleClientset(), cache).PodController(context.Background(), corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo",
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "Job", Name: "backup-123", UID: types.UID("job-uid"), Controller: &controller,
		}},
	}})
	if len(resolution.Chain) != 2 || resolution.Chain[1].Kind != "CronJob" || resolution.TopUID != "cronjob-uid" {
		t.Fatalf("unexpected Job controller resolution: %#v", resolution)
	}
}

func TestResourceMapperRejectsMismatchedOwnerUID(t *testing.T) {
	controller := true
	cache := &mapperCache{ready: true, replicaSet: &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:      "reused-name",
		Namespace: "demo",
		UID:       types.UID("new-rs-uid"),
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "Deployment", Name: "wrong", UID: types.UID("wrong-deployment-uid"), Controller: &controller,
		}},
	}}}
	resolution := NewResourceMapper(fake.NewSimpleClientset(), cache).PodController(context.Background(), corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo",
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "ReplicaSet", Name: "reused-name", UID: types.UID("old-rs-uid"), Controller: &controller,
		}},
	}})
	if len(resolution.Chain) != 1 || resolution.TopUID != "old-rs-uid" {
		t.Fatalf("mismatched UID must not attach the cached parent: %#v", resolution)
	}
}

func TestResourceMapperFallsBackToAPIWhenCacheNotReady(t *testing.T) {
	controller := true
	client := fake.NewSimpleClientset(&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:      "app-rs",
		Namespace: "demo",
		UID:       types.UID("rs-uid"),
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "Deployment", Name: "app", UID: types.UID("deployment-uid"), Controller: &controller,
		}},
	}})
	resolution := NewResourceMapper(client, &mapperCache{ready: false}).PodController(context.Background(), corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "demo",
		OwnerReferences: []metav1.OwnerReference{{
			Kind: "ReplicaSet", Name: "app-rs", UID: types.UID("rs-uid"), Controller: &controller,
		}},
	}})
	if len(resolution.Chain) != 2 || resolution.TopUID != "deployment-uid" {
		t.Fatalf("unexpected API fallback resolution: %#v", resolution)
	}
	if len(client.Actions()) != 1 || client.Actions()[0].GetVerb() != "get" || client.Actions()[0].GetResource().Resource != "replicasets" {
		t.Fatalf("expected one ReplicaSet API get, got %#v", client.Actions())
	}
}
