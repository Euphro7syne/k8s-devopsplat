package resources

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (s *Service) cacheReady() bool { return s.cache != nil && s.cache.Ready() }

func (s *Service) listNamespaces(ctx context.Context, _ string) ([]corev1.Namespace, error) {
	if s.cacheReady() {
		items, err := s.cache.ListNamespaces()
		return items, err
	}
	items, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getNamespace(ctx context.Context, name, _ string) (*corev1.Namespace, error) {
	if s.cacheReady() {
		item, err := s.cache.GetNamespace(name)
		return item, err
	}
	item, err := s.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listNodes(ctx context.Context, _ string) ([]corev1.Node, error) {
	if s.cacheReady() {
		items, err := s.cache.ListNodes()
		return items, err
	}
	items, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getNode(ctx context.Context, name, _ string) (*corev1.Node, error) {
	if s.cacheReady() {
		item, err := s.cache.GetNode(name)
		return item, err
	}
	item, err := s.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listPods(ctx context.Context, namespace, _ string) ([]corev1.Pod, error) {
	if s.cacheReady() {
		items, err := s.cache.ListPods(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getPod(ctx context.Context, namespace, name, _ string) (*corev1.Pod, error) {
	if s.cacheReady() {
		item, err := s.cache.GetPod(namespace, name)
		return item, err
	}
	item, err := s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listEvents(ctx context.Context, namespace, _ string) ([]corev1.Event, error) {
	if s.cacheReady() {
		items, err := s.cache.ListEvents(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) listDeployments(ctx context.Context, namespace, _ string) ([]appsv1.Deployment, error) {
	if s.cacheReady() {
		items, err := s.cache.ListDeployments(namespace)
		return items, err
	}
	items, err := s.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getDeployment(ctx context.Context, namespace, name, _ string) (*appsv1.Deployment, error) {
	if s.cacheReady() {
		item, err := s.cache.GetDeployment(namespace, name)
		return item, err
	}
	item, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listStatefulSets(ctx context.Context, namespace, _ string) ([]appsv1.StatefulSet, error) {
	if s.cacheReady() {
		items, err := s.cache.ListStatefulSets(namespace)
		return items, err
	}
	items, err := s.client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getStatefulSet(ctx context.Context, namespace, name, _ string) (*appsv1.StatefulSet, error) {
	if s.cacheReady() {
		item, err := s.cache.GetStatefulSet(namespace, name)
		return item, err
	}
	item, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listDaemonSets(ctx context.Context, namespace, _ string) ([]appsv1.DaemonSet, error) {
	if s.cacheReady() {
		items, err := s.cache.ListDaemonSets(namespace)
		return items, err
	}
	items, err := s.client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getDaemonSet(ctx context.Context, namespace, name, _ string) (*appsv1.DaemonSet, error) {
	if s.cacheReady() {
		item, err := s.cache.GetDaemonSet(namespace, name)
		return item, err
	}
	item, err := s.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listReplicaSets(ctx context.Context, namespace, _ string) ([]appsv1.ReplicaSet, error) {
	if s.cacheReady() {
		items, err := s.cache.ListReplicaSets(namespace)
		return items, err
	}
	items, err := s.client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getReplicaSet(ctx context.Context, namespace, name, _ string) (*appsv1.ReplicaSet, error) {
	if s.cacheReady() {
		item, err := s.cache.GetReplicaSet(namespace, name)
		return item, err
	}
	item, err := s.client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listJobs(ctx context.Context, namespace, _ string) ([]batchv1.Job, error) {
	if s.cacheReady() {
		items, err := s.cache.ListJobs(namespace)
		return items, err
	}
	items, err := s.client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getJob(ctx context.Context, namespace, name, _ string) (*batchv1.Job, error) {
	if s.cacheReady() {
		item, err := s.cache.GetJob(namespace, name)
		return item, err
	}
	item, err := s.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listCronJobs(ctx context.Context, namespace, _ string) ([]batchv1.CronJob, error) {
	if s.cacheReady() {
		items, err := s.cache.ListCronJobs(namespace)
		return items, err
	}
	items, err := s.client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getCronJob(ctx context.Context, namespace, name, _ string) (*batchv1.CronJob, error) {
	if s.cacheReady() {
		item, err := s.cache.GetCronJob(namespace, name)
		return item, err
	}
	item, err := s.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listServices(ctx context.Context, namespace, _ string) ([]corev1.Service, error) {
	if s.cacheReady() {
		items, err := s.cache.ListServices(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getService(ctx context.Context, namespace, name, _ string) (*corev1.Service, error) {
	if s.cacheReady() {
		item, err := s.cache.GetService(namespace, name)
		return item, err
	}
	item, err := s.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listEndpoints(ctx context.Context, namespace, _ string) ([]corev1.Endpoints, error) {
	if s.cacheReady() {
		items, err := s.cache.ListEndpoints(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getEndpoints(ctx context.Context, namespace, name, _ string) (*corev1.Endpoints, error) {
	if s.cacheReady() {
		item, err := s.cache.GetEndpoints(namespace, name)
		return item, err
	}
	item, err := s.client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listEndpointSlices(ctx context.Context, namespace string, selector labels.Selector, _ string) ([]discoveryv1.EndpointSlice, error) {
	if s.cacheReady() {
		items, err := s.cache.ListEndpointSlices(namespace, selector)
		return items, err
	}
	options := metav1.ListOptions{}
	if selector != nil {
		options.LabelSelector = selector.String()
	}
	items, err := s.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, options)
	return items.Items, err
}

func (s *Service) listIngresses(ctx context.Context, namespace, _ string) ([]networkingv1.Ingress, error) {
	if s.cacheReady() {
		items, err := s.cache.ListIngresses(namespace)
		return items, err
	}
	items, err := s.client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getIngress(ctx context.Context, namespace, name, _ string) (*networkingv1.Ingress, error) {
	if s.cacheReady() {
		item, err := s.cache.GetIngress(namespace, name)
		return item, err
	}
	item, err := s.client.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listPVCs(ctx context.Context, namespace, _ string) ([]corev1.PersistentVolumeClaim, error) {
	if s.cacheReady() {
		items, err := s.cache.ListPVCs(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getPVC(ctx context.Context, namespace, name, _ string) (*corev1.PersistentVolumeClaim, error) {
	if s.cacheReady() {
		item, err := s.cache.GetPVC(namespace, name)
		return item, err
	}
	item, err := s.client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listPVs(ctx context.Context, _ string) ([]corev1.PersistentVolume, error) {
	if s.cacheReady() {
		items, err := s.cache.ListPVs()
		return items, err
	}
	items, err := s.client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getPV(ctx context.Context, name, _ string) (*corev1.PersistentVolume, error) {
	if s.cacheReady() {
		item, err := s.cache.GetPV(name)
		return item, err
	}
	item, err := s.client.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	return item, err
}

func (s *Service) listConfigMaps(ctx context.Context, namespace, _ string) ([]corev1.ConfigMap, error) {
	if s.cacheReady() {
		items, err := s.cache.ListConfigMaps(namespace)
		return items, err
	}
	items, err := s.client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) listStorageClasses(ctx context.Context, _ string) ([]storagev1.StorageClass, error) {
	if s.cacheReady() {
		items, err := s.cache.ListStorageClasses()
		return items, err
	}
	items, err := s.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	return items.Items, err
}

func (s *Service) getStorageClass(ctx context.Context, name, _ string) (*storagev1.StorageClass, error) {
	if s.cacheReady() {
		item, err := s.cache.GetStorageClass(name)
		return item, err
	}
	item, err := s.client.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	return item, err
}
