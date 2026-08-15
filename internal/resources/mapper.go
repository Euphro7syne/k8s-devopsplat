package resources

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

type ControllerResolution struct {
	Chain  []OwnerReference
	TopUID string
}

type ResourceMapper struct {
	client KubernetesClient
	cache  ResourceCache
}

func NewResourceMapper(client KubernetesClient, cache ResourceCache) *ResourceMapper {
	return &ResourceMapper{client: client, cache: cache}
}

func (m *ResourceMapper) PodController(ctx context.Context, pod corev1.Pod) ControllerResolution {
	controller, ok := controllerOwnerReference(pod.OwnerReferences)
	if !ok {
		return ControllerResolution{}
	}
	result := ControllerResolution{
		Chain:  []OwnerReference{ownerReference(controller)},
		TopUID: string(controller.UID),
	}

	switch strings.ToLower(controller.Kind) {
	case "replicaset":
		item, err := m.replicaSet(ctx, pod.Namespace, controller.Name)
		if err == nil && ownerUIDMatches(controller.UID, item.UID) {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				result.Chain = append(result.Chain, ownerReference(parent))
				result.TopUID = string(parent.UID)
			}
		}
	case "job":
		item, err := m.job(ctx, pod.Namespace, controller.Name)
		if err == nil && ownerUIDMatches(controller.UID, item.UID) {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				result.Chain = append(result.Chain, ownerReference(parent))
				result.TopUID = string(parent.UID)
			}
		}
	}
	return result
}

func (m *ResourceMapper) replicaSet(ctx context.Context, namespace, name string) (*appsv1.ReplicaSet, error) {
	if m.cache != nil && m.cache.Ready() {
		return m.cache.GetReplicaSet(namespace, name)
	}
	return m.client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (m *ResourceMapper) job(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	if m.cache != nil && m.cache.Ready() {
		return m.cache.GetJob(namespace, name)
	}
	return m.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

func ownerUIDMatches(referenceUID, objectUID k8stypes.UID) bool {
	return referenceUID == "" || referenceUID == objectUID
}
