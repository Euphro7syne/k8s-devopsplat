package k8s

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	storageinformers "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	discoverylisters "k8s.io/client-go/listers/discovery/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	toolscache "k8s.io/client-go/tools/cache"
)

// ResourceCache keeps only the non-sensitive P0 read model in shared
// informers. Secret is deliberately absent: its values must never be retained
// in this process-wide cache.
type ResourceCache struct {
	informers []toolscache.SharedIndexInformer
	ready     atomic.Bool

	namespaces corelisters.NamespaceLister
	nodes      corelisters.NodeLister
	pods       corelisters.PodLister
	events     corelisters.EventLister
	services   corelisters.ServiceLister
	endpoints  corelisters.EndpointsLister
	pvcs       corelisters.PersistentVolumeClaimLister
	pvs        corelisters.PersistentVolumeLister
	configMaps corelisters.ConfigMapLister

	deployments  appslisters.DeploymentLister
	statefulSets appslisters.StatefulSetLister
	daemonSets   appslisters.DaemonSetLister
	replicaSets  appslisters.ReplicaSetLister

	jobs     batchlisters.JobLister
	cronJobs batchlisters.CronJobLister

	endpointSlices discoverylisters.EndpointSliceLister
	ingresses      networkinglisters.IngressLister
	storageClasses storagelisters.StorageClassLister
}

func NewResourceCache(client kubernetes.Interface, resync time.Duration) *ResourceCache {
	namespacedIndexers := toolscache.Indexers{toolscache.NamespaceIndex: toolscache.MetaNamespaceIndexFunc}
	clusterIndexers := toolscache.Indexers{}

	namespaceInformer := coreinformers.NewNamespaceInformer(client, resync, clusterIndexers)
	nodeInformer := coreinformers.NewNodeInformer(client, resync, clusterIndexers)
	podInformer := coreinformers.NewPodInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	eventInformer := coreinformers.NewEventInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	serviceInformer := coreinformers.NewServiceInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	endpointsInformer := coreinformers.NewEndpointsInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	pvcInformer := coreinformers.NewPersistentVolumeClaimInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	pvInformer := coreinformers.NewPersistentVolumeInformer(client, resync, clusterIndexers)
	configMapInformer := coreinformers.NewConfigMapInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)

	deploymentInformer := appsinformers.NewDeploymentInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	statefulSetInformer := appsinformers.NewStatefulSetInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	daemonSetInformer := appsinformers.NewDaemonSetInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	replicaSetInformer := appsinformers.NewReplicaSetInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)

	jobInformer := batchinformers.NewJobInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	cronJobInformer := batchinformers.NewCronJobInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)

	endpointSliceInformer := discoveryinformers.NewEndpointSliceInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	ingressInformer := networkinginformers.NewIngressInformer(client, metav1.NamespaceAll, resync, namespacedIndexers)
	storageClassInformer := storageinformers.NewStorageClassInformer(client, resync, clusterIndexers)

	return &ResourceCache{
		informers: []toolscache.SharedIndexInformer{
			namespaceInformer,
			nodeInformer,
			podInformer,
			eventInformer,
			serviceInformer,
			endpointsInformer,
			pvcInformer,
			pvInformer,
			configMapInformer,
			deploymentInformer,
			statefulSetInformer,
			daemonSetInformer,
			replicaSetInformer,
			jobInformer,
			cronJobInformer,
			endpointSliceInformer,
			ingressInformer,
			storageClassInformer,
		},

		namespaces: corelisters.NewNamespaceLister(namespaceInformer.GetIndexer()),
		nodes:      corelisters.NewNodeLister(nodeInformer.GetIndexer()),
		pods:       corelisters.NewPodLister(podInformer.GetIndexer()),
		events:     corelisters.NewEventLister(eventInformer.GetIndexer()),
		services:   corelisters.NewServiceLister(serviceInformer.GetIndexer()),
		endpoints:  corelisters.NewEndpointsLister(endpointsInformer.GetIndexer()),
		pvcs:       corelisters.NewPersistentVolumeClaimLister(pvcInformer.GetIndexer()),
		pvs:        corelisters.NewPersistentVolumeLister(pvInformer.GetIndexer()),
		configMaps: corelisters.NewConfigMapLister(configMapInformer.GetIndexer()),

		deployments:  appslisters.NewDeploymentLister(deploymentInformer.GetIndexer()),
		statefulSets: appslisters.NewStatefulSetLister(statefulSetInformer.GetIndexer()),
		daemonSets:   appslisters.NewDaemonSetLister(daemonSetInformer.GetIndexer()),
		replicaSets:  appslisters.NewReplicaSetLister(replicaSetInformer.GetIndexer()),

		jobs:     batchlisters.NewJobLister(jobInformer.GetIndexer()),
		cronJobs: batchlisters.NewCronJobLister(cronJobInformer.GetIndexer()),

		endpointSlices: discoverylisters.NewEndpointSliceLister(endpointSliceInformer.GetIndexer()),
		ingresses:      networkinglisters.NewIngressLister(ingressInformer.GetIndexer()),
		storageClasses: storagelisters.NewStorageClassLister(storageClassInformer.GetIndexer()),
	}
}

func (c *ResourceCache) Start(ctx context.Context, syncTimeout time.Duration) error {
	if c == nil {
		return fmt.Errorf("resource cache is nil")
	}
	if syncTimeout <= 0 {
		syncTimeout = 15 * time.Second
	}
	for _, informer := range c.informers {
		go informer.Run(ctx.Done())
	}

	syncCtx, cancel := context.WithTimeout(ctx, syncTimeout)
	defer cancel()
	syncs := make([]toolscache.InformerSynced, 0, len(c.informers))
	for _, informer := range c.informers {
		syncs = append(syncs, informer.HasSynced)
	}
	if !toolscache.WaitForCacheSync(syncCtx.Done(), syncs...) {
		return fmt.Errorf("kubernetes informer cache sync failed or timed out")
	}

	c.ready.Store(true)
	go func() {
		<-ctx.Done()
		c.ready.Store(false)
	}()
	return nil
}

func (c *ResourceCache) Shutdown() {
	if c != nil {
		c.ready.Store(false)
	}
}

func (c *ResourceCache) Ready() bool { return c != nil && c.ready.Load() }

func (c *ResourceCache) ListNamespaces() ([]corev1.Namespace, error) {
	items, err := c.namespaces.List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetNamespace(name string) (*corev1.Namespace, error) {
	return c.namespaces.Get(name)
}

func (c *ResourceCache) ListNodes() ([]corev1.Node, error) {
	items, err := c.nodes.List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetNode(name string) (*corev1.Node, error) { return c.nodes.Get(name) }

func (c *ResourceCache) ListPods(namespace string) ([]corev1.Pod, error) {
	var items []*corev1.Pod
	var err error
	if namespace == "" {
		items, err = c.pods.List(labels.Everything())
	} else {
		items, err = c.pods.Pods(namespace).List(labels.Everything())
	}
	return values(items), err
}

func (c *ResourceCache) GetPod(namespace, name string) (*corev1.Pod, error) {
	return c.pods.Pods(namespace).Get(name)
}

func (c *ResourceCache) ListEvents(namespace string) ([]corev1.Event, error) {
	var items []*corev1.Event
	var err error
	if namespace == "" {
		items, err = c.events.List(labels.Everything())
	} else {
		items, err = c.events.Events(namespace).List(labels.Everything())
	}
	return values(items), err
}

func (c *ResourceCache) ListServices(namespace string) ([]corev1.Service, error) {
	items, err := c.services.Services(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetService(namespace, name string) (*corev1.Service, error) {
	return c.services.Services(namespace).Get(name)
}

func (c *ResourceCache) ListEndpoints(namespace string) ([]corev1.Endpoints, error) {
	items, err := c.endpoints.Endpoints(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetEndpoints(namespace, name string) (*corev1.Endpoints, error) {
	return c.endpoints.Endpoints(namespace).Get(name)
}

func (c *ResourceCache) ListPVCs(namespace string) ([]corev1.PersistentVolumeClaim, error) {
	items, err := c.pvcs.PersistentVolumeClaims(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetPVC(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	return c.pvcs.PersistentVolumeClaims(namespace).Get(name)
}

func (c *ResourceCache) ListPVs() ([]corev1.PersistentVolume, error) {
	items, err := c.pvs.List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetPV(name string) (*corev1.PersistentVolume, error) { return c.pvs.Get(name) }

func (c *ResourceCache) ListConfigMaps(namespace string) ([]corev1.ConfigMap, error) {
	items, err := c.configMaps.ConfigMaps(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) ListDeployments(namespace string) ([]appsv1.Deployment, error) {
	items, err := c.deployments.Deployments(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	return c.deployments.Deployments(namespace).Get(name)
}

func (c *ResourceCache) ListStatefulSets(namespace string) ([]appsv1.StatefulSet, error) {
	items, err := c.statefulSets.StatefulSets(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	return c.statefulSets.StatefulSets(namespace).Get(name)
}

func (c *ResourceCache) ListDaemonSets(namespace string) ([]appsv1.DaemonSet, error) {
	items, err := c.daemonSets.DaemonSets(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetDaemonSet(namespace, name string) (*appsv1.DaemonSet, error) {
	return c.daemonSets.DaemonSets(namespace).Get(name)
}

func (c *ResourceCache) ListReplicaSets(namespace string) ([]appsv1.ReplicaSet, error) {
	items, err := c.replicaSets.ReplicaSets(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error) {
	return c.replicaSets.ReplicaSets(namespace).Get(name)
}

func (c *ResourceCache) ListJobs(namespace string) ([]batchv1.Job, error) {
	items, err := c.jobs.Jobs(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetJob(namespace, name string) (*batchv1.Job, error) {
	return c.jobs.Jobs(namespace).Get(name)
}

func (c *ResourceCache) ListCronJobs(namespace string) ([]batchv1.CronJob, error) {
	items, err := c.cronJobs.CronJobs(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetCronJob(namespace, name string) (*batchv1.CronJob, error) {
	return c.cronJobs.CronJobs(namespace).Get(name)
}

func (c *ResourceCache) ListEndpointSlices(namespace string, selector labels.Selector) ([]discoveryv1.EndpointSlice, error) {
	if selector == nil {
		selector = labels.Everything()
	}
	items, err := c.endpointSlices.EndpointSlices(namespace).List(selector)
	return values(items), err
}

func (c *ResourceCache) ListIngresses(namespace string) ([]networkingv1.Ingress, error) {
	items, err := c.ingresses.Ingresses(namespace).List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetIngress(namespace, name string) (*networkingv1.Ingress, error) {
	return c.ingresses.Ingresses(namespace).Get(name)
}

func (c *ResourceCache) ListStorageClasses() ([]storagev1.StorageClass, error) {
	items, err := c.storageClasses.List(labels.Everything())
	return values(items), err
}

func (c *ResourceCache) GetStorageClass(name string) (*storagev1.StorageClass, error) {
	return c.storageClasses.Get(name)
}

func values[T any](items []*T) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, *item)
		}
	}
	return result
}
