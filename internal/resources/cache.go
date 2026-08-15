package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ResourceCache is the read-only cache contract used by the resources module.
// Implementations must return objects that callers treat as immutable.
// Secret is intentionally excluded so plaintext values are not retained in the
// shared informer cache.
type ResourceCache interface {
	Ready() bool

	ListNamespaces() ([]corev1.Namespace, error)
	GetNamespace(name string) (*corev1.Namespace, error)
	ListNodes() ([]corev1.Node, error)
	GetNode(name string) (*corev1.Node, error)
	ListPods(namespace string) ([]corev1.Pod, error)
	GetPod(namespace, name string) (*corev1.Pod, error)
	ListEvents(namespace string) ([]corev1.Event, error)
	ListServices(namespace string) ([]corev1.Service, error)
	GetService(namespace, name string) (*corev1.Service, error)
	ListEndpoints(namespace string) ([]corev1.Endpoints, error)
	GetEndpoints(namespace, name string) (*corev1.Endpoints, error)
	ListPVCs(namespace string) ([]corev1.PersistentVolumeClaim, error)
	GetPVC(namespace, name string) (*corev1.PersistentVolumeClaim, error)
	ListPVs() ([]corev1.PersistentVolume, error)
	GetPV(name string) (*corev1.PersistentVolume, error)
	ListConfigMaps(namespace string) ([]corev1.ConfigMap, error)

	ListDeployments(namespace string) ([]appsv1.Deployment, error)
	GetDeployment(namespace, name string) (*appsv1.Deployment, error)
	ListStatefulSets(namespace string) ([]appsv1.StatefulSet, error)
	GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	ListDaemonSets(namespace string) ([]appsv1.DaemonSet, error)
	GetDaemonSet(namespace, name string) (*appsv1.DaemonSet, error)
	ListReplicaSets(namespace string) ([]appsv1.ReplicaSet, error)
	GetReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error)

	ListJobs(namespace string) ([]batchv1.Job, error)
	GetJob(namespace, name string) (*batchv1.Job, error)
	ListCronJobs(namespace string) ([]batchv1.CronJob, error)
	GetCronJob(namespace, name string) (*batchv1.CronJob, error)

	ListEndpointSlices(namespace string, selector labels.Selector) ([]discoveryv1.EndpointSlice, error)
	ListIngresses(namespace string) ([]networkingv1.Ingress, error)
	GetIngress(namespace, name string) (*networkingv1.Ingress, error)
	ListStorageClasses() ([]storagev1.StorageClass, error)
	GetStorageClass(name string) (*storagev1.StorageClass, error)
}
