package resources

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	typeddiscoveryv1 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	typednetworkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	typedstoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	"sigs.k8s.io/yaml"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/pagination"
)

type KubernetesClient interface {
	CoreV1() typedcorev1.CoreV1Interface
	AppsV1() typedappsv1.AppsV1Interface
	BatchV1() typedbatchv1.BatchV1Interface
	DiscoveryV1() typeddiscoveryv1.DiscoveryV1Interface
	NetworkingV1() typednetworkingv1.NetworkingV1Interface
	StorageV1() typedstoragev1.StorageV1Interface
}

type Service struct {
	client      KubernetesClient
	clusterName string
}

func NewService(client KubernetesClient, clusterName string) *Service {
	if clusterName == "" {
		clusterName = "in-cluster"
	}
	return &Service{client: client, clusterName: clusterName}
}

func (s *Service) Overview(ctx context.Context) (ClusterOverview, error) {
	if s.client == nil {
		return ClusterOverview{}, unavailable()
	}
	nodes, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ClusterOverview{}, mapKubernetesError(err, "list nodes failed")
	}
	namespaces, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ClusterOverview{}, mapKubernetesError(err, "list namespaces failed")
	}
	pods, err := s.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ClusterOverview{}, mapKubernetesError(err, "list pods failed")
	}

	overview := ClusterOverview{
		Cluster:        s.clusterName,
		NodeCount:      len(nodes.Items),
		NamespaceCount: len(namespaces.Items),
		PodCount:       len(pods.Items),
		Nodes:          make([]NodeSummary, 0, len(nodes.Items)),
	}
	for _, node := range nodes.Items {
		summary := nodeSummary(node)
		if summary.Status == "Ready" {
			overview.ReadyNodeCount++
		}
		overview.Nodes = append(overview.Nodes, summary)
	}
	for _, pod := range pods.Items {
		if isAbnormalPod(pod) {
			overview.AbnormalPodCount++
		}
	}
	return overview, nil
}

func (s *Service) ListNamespaces(ctx context.Context) ([]NamespaceSummary, error) {
	if s.client == nil {
		return nil, unavailable()
	}
	items, err := s.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, mapKubernetesError(err, "list namespaces failed")
	}
	namespaces := make([]NamespaceSummary, 0, len(items.Items))
	for _, ns := range items.Items {
		namespaces = append(namespaces, NamespaceSummary{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			CreatedAt: ns.CreationTimestamp.Time,
		})
	}
	sort.Slice(namespaces, func(i, j int) bool {
		return namespaces[i].Name < namespaces[j].Name
	})
	return namespaces, nil
}

func (s *Service) GetNamespace(ctx context.Context, name string) (NamespaceDetail, error) {
	if s.client == nil {
		return NamespaceDetail{}, unavailable()
	}
	namespace, err := s.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "get namespace failed")
	}
	pods, err := s.client.CoreV1().Pods(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace pods failed")
	}
	deployments, err := s.client.AppsV1().Deployments(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace deployments failed")
	}
	statefulSets, err := s.client.AppsV1().StatefulSets(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace statefulsets failed")
	}
	daemonSets, err := s.client.AppsV1().DaemonSets(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace daemonsets failed")
	}
	replicaSets, err := s.client.AppsV1().ReplicaSets(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace replicasets failed")
	}
	jobs, err := s.client.BatchV1().Jobs(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace jobs failed")
	}
	cronJobs, err := s.client.BatchV1().CronJobs(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace cronjobs failed")
	}
	services, err := s.client.CoreV1().Services(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace services failed")
	}
	ingresses, err := s.client.NetworkingV1().Ingresses(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace ingresses failed")
	}
	pvcs, err := s.client.CoreV1().PersistentVolumeClaims(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace persistent volume claims failed")
	}
	configMaps, err := s.client.CoreV1().ConfigMaps(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace configmaps failed")
	}
	events, err := s.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceDetail{}, mapKubernetesError(err, "list namespace events failed")
	}

	return s.namespaceDetail(ctx, *namespace, pods.Items, deployments.Items, statefulSets.Items, daemonSets.Items, replicaSets.Items, jobs.Items, cronJobs.Items, services.Items, ingresses.Items, pvcs.Items, configMaps.Items, events.Items), nil
}

func (s *Service) ListNodes(ctx context.Context, opts ListOptions) (pagination.Result[NodeSummary], error) {
	if s.client == nil {
		return pagination.Result[NodeSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[NodeSummary]{}, mapKubernetesError(err, "list nodes failed")
	}
	nodes := make([]NodeSummary, 0, len(items.Items))
	for _, node := range items.Items {
		nodes = append(nodes, nodeSummary(node))
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	return paginate(nodes, opts), nil
}

func (s *Service) GetNode(ctx context.Context, name string) (NodeDetail, error) {
	if s.client == nil {
		return NodeDetail{}, unavailable()
	}
	node, err := s.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return NodeDetail{}, mapKubernetesError(err, "get node failed")
	}
	pods, err := s.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return NodeDetail{}, mapKubernetesError(err, "list node pods failed")
	}
	events, err := s.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return NodeDetail{}, mapKubernetesError(err, "list node events failed")
	}
	return s.nodeDetail(ctx, *node, pods.Items, events.Items), nil
}

func (s *Service) ListPods(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[PodSummary], error) {
	if s.client == nil {
		return pagination.Result[PodSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[PodSummary]{}, mapKubernetesError(err, "list pods failed")
	}
	pods := make([]PodSummary, 0, len(items.Items))
	for _, pod := range items.Items {
		pods = append(pods, podSummary(pod))
	}
	sort.Slice(pods, func(i, j int) bool {
		if pods[i].Namespace == pods[j].Namespace {
			return pods[i].Name < pods[j].Name
		}
		return pods[i].Namespace < pods[j].Namespace
	})
	return paginate(pods, opts), nil
}

func (s *Service) GetPod(ctx context.Context, namespace, name string) (PodDetail, error) {
	if s.client == nil {
		return PodDetail{}, unavailable()
	}
	pod, err := s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return PodDetail{}, mapKubernetesError(err, "get pod failed")
	}
	return podDetail(*pod, s.podControllerChain(ctx, *pod)), nil
}

func (s *Service) PodYAML(ctx context.Context, namespace, name string) (string, error) {
	if s.client == nil {
		return "", unavailable()
	}
	pod, err := s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", mapKubernetesError(err, "get pod yaml failed")
	}
	pod.ManagedFields = nil
	raw, err := yaml.Marshal(pod)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Service) ListEvents(ctx context.Context, namespace, involvedKind, involvedName string) ([]EventSummary, error) {
	if s.client == nil {
		return nil, unavailable()
	}
	items, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, mapKubernetesError(err, "list events failed")
	}
	events := make([]EventSummary, 0, len(items.Items))
	for _, event := range items.Items {
		if involvedKind != "" && !strings.EqualFold(event.InvolvedObject.Kind, involvedKind) {
			continue
		}
		if involvedName != "" && event.InvolvedObject.Name != involvedName {
			continue
		}
		events = append(events, eventSummary(event))
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp)
	})
	return events, nil
}

func (s *Service) podControllerChain(ctx context.Context, pod corev1.Pod) []OwnerReference {
	controller, ok := controllerOwnerReference(pod.OwnerReferences)
	if !ok {
		return nil
	}
	chain := []OwnerReference{ownerReference(controller)}

	switch strings.ToLower(controller.Kind) {
	case "replicaset":
		item, err := s.client.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err == nil {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				chain = append(chain, ownerReference(parent))
			}
		}
	case "job":
		item, err := s.client.BatchV1().Jobs(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err == nil {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				chain = append(chain, ownerReference(parent))
			}
		}
	}
	return chain
}

func (s *Service) ListDeployments(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[DeploymentSummary], error) {
	if s.client == nil {
		return pagination.Result[DeploymentSummary]{}, unavailable()
	}
	items, err := s.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[DeploymentSummary]{}, mapKubernetesError(err, "list deployments failed")
	}
	deployments := make([]DeploymentSummary, 0, len(items.Items))
	for _, deployment := range items.Items {
		deployments = append(deployments, DeploymentSummaryFrom(deployment))
	}
	sort.Slice(deployments, func(i, j int) bool {
		if deployments[i].Namespace == deployments[j].Namespace {
			return deployments[i].Name < deployments[j].Name
		}
		return deployments[i].Namespace < deployments[j].Namespace
	})
	return paginate(deployments, opts), nil
}

func (s *Service) GetDeployment(ctx context.Context, namespace, name string) (DeploymentDetail, error) {
	if s.client == nil {
		return DeploymentDetail{}, unavailable()
	}
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return DeploymentDetail{}, mapKubernetesError(err, "get deployment failed")
	}
	replicaSetList, err := s.client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return DeploymentDetail{}, mapKubernetesError(err, "list deployment replicasets failed")
	}
	podList, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return DeploymentDetail{}, mapKubernetesError(err, "list deployment pods failed")
	}
	eventList, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return DeploymentDetail{}, mapKubernetesError(err, "list deployment events failed")
	}

	return deploymentDetail(*deployment, replicaSetList.Items, podList.Items, eventList.Items), nil
}

func (s *Service) DeploymentYAML(ctx context.Context, namespace, name string) (string, error) {
	if s.client == nil {
		return "", unavailable()
	}
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", mapKubernetesError(err, "get deployment yaml failed")
	}
	deployment.ManagedFields = nil
	raw, err := yaml.Marshal(deployment)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Service) ListStatefulSets(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[WorkloadSummary], error) {
	if s.client == nil {
		return pagination.Result[WorkloadSummary]{}, unavailable()
	}
	items, err := s.client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[WorkloadSummary]{}, mapKubernetesError(err, "list statefulsets failed")
	}
	workloads := make([]WorkloadSummary, 0, len(items.Items))
	for _, item := range items.Items {
		workloads = append(workloads, StatefulSetSummaryFrom(item))
	}
	sortWorkloads(workloads)
	return paginate(workloads, opts), nil
}

func (s *Service) GetStatefulSet(ctx context.Context, namespace, name string) (StatefulSetDetail, error) {
	if s.client == nil {
		return StatefulSetDetail{}, unavailable()
	}
	item, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return StatefulSetDetail{}, mapKubernetesError(err, "get statefulset failed")
	}
	return StatefulSetDetailFrom(*item), nil
}

func (s *Service) StatefulSetYAML(ctx context.Context, namespace, name string) (string, error) {
	if s.client == nil {
		return "", unavailable()
	}
	item, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", mapKubernetesError(err, "get statefulset yaml failed")
	}
	return marshalResourceYAML(item)
}

func (s *Service) ListReplicaSets(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[WorkloadSummary], error) {
	if s.client == nil {
		return pagination.Result[WorkloadSummary]{}, unavailable()
	}
	items, err := s.client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[WorkloadSummary]{}, mapKubernetesError(err, "list replicasets failed")
	}
	workloads := make([]WorkloadSummary, 0, len(items.Items))
	for _, item := range items.Items {
		workloads = append(workloads, replicaSetSummary(item))
	}
	sortWorkloads(workloads)
	return paginate(workloads, opts), nil
}

func (s *Service) GetReplicaSet(ctx context.Context, namespace, name string) (ReplicaSetDetail, error) {
	if s.client == nil {
		return ReplicaSetDetail{}, unavailable()
	}
	item, err := s.client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return ReplicaSetDetail{}, mapKubernetesError(err, "get replicaset failed")
	}
	podList, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ReplicaSetDetail{}, mapKubernetesError(err, "list replicaset pods failed")
	}
	eventList, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ReplicaSetDetail{}, mapKubernetesError(err, "list replicaset events failed")
	}
	return replicaSetDiagnosticDetail(*item, podList.Items, eventList.Items), nil
}

func (s *Service) ListDaemonSets(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[DaemonSetSummary], error) {
	if s.client == nil {
		return pagination.Result[DaemonSetSummary]{}, unavailable()
	}
	items, err := s.client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[DaemonSetSummary]{}, mapKubernetesError(err, "list daemonsets failed")
	}
	daemonSets := make([]DaemonSetSummary, 0, len(items.Items))
	for _, item := range items.Items {
		daemonSets = append(daemonSets, DaemonSetSummaryFrom(item))
	}
	sort.Slice(daemonSets, func(i, j int) bool {
		if daemonSets[i].Namespace == daemonSets[j].Namespace {
			return daemonSets[i].Name < daemonSets[j].Name
		}
		return daemonSets[i].Namespace < daemonSets[j].Namespace
	})
	return paginate(daemonSets, opts), nil
}

func (s *Service) GetDaemonSet(ctx context.Context, namespace, name string) (DaemonSetDetail, error) {
	if s.client == nil {
		return DaemonSetDetail{}, unavailable()
	}
	item, err := s.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return DaemonSetDetail{}, mapKubernetesError(err, "get daemonset failed")
	}
	return daemonSetDetail(*item), nil
}

func (s *Service) DaemonSetYAML(ctx context.Context, namespace, name string) (string, error) {
	if s.client == nil {
		return "", unavailable()
	}
	item, err := s.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", mapKubernetesError(err, "get daemonset yaml failed")
	}
	return marshalResourceYAML(item)
}

func (s *Service) ListJobs(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[JobSummary], error) {
	if s.client == nil {
		return pagination.Result[JobSummary]{}, unavailable()
	}
	items, err := s.client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[JobSummary]{}, mapKubernetesError(err, "list jobs failed")
	}
	jobs := make([]JobSummary, 0, len(items.Items))
	for _, item := range items.Items {
		jobs = append(jobs, jobSummary(item))
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Namespace == jobs[j].Namespace {
			return jobs[i].Name < jobs[j].Name
		}
		return jobs[i].Namespace < jobs[j].Namespace
	})
	return paginate(jobs, opts), nil
}

func (s *Service) GetJob(ctx context.Context, namespace, name string) (JobDetail, error) {
	if s.client == nil {
		return JobDetail{}, unavailable()
	}
	item, err := s.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return JobDetail{}, mapKubernetesError(err, "get job failed")
	}
	podList, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return JobDetail{}, mapKubernetesError(err, "list job pods failed")
	}
	eventList, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return JobDetail{}, mapKubernetesError(err, "list job events failed")
	}
	return jobDetail(*item, podList.Items, eventList.Items), nil
}

func (s *Service) ListCronJobs(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[CronJobSummary], error) {
	if s.client == nil {
		return pagination.Result[CronJobSummary]{}, unavailable()
	}
	items, err := s.client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[CronJobSummary]{}, mapKubernetesError(err, "list cronjobs failed")
	}
	cronJobs := make([]CronJobSummary, 0, len(items.Items))
	for _, item := range items.Items {
		cronJobs = append(cronJobs, CronJobSummaryFrom(item))
	}
	sort.Slice(cronJobs, func(i, j int) bool {
		if cronJobs[i].Namespace == cronJobs[j].Namespace {
			return cronJobs[i].Name < cronJobs[j].Name
		}
		return cronJobs[i].Namespace < cronJobs[j].Namespace
	})
	return paginate(cronJobs, opts), nil
}

func (s *Service) GetCronJob(ctx context.Context, namespace, name string) (CronJobDetail, error) {
	if s.client == nil {
		return CronJobDetail{}, unavailable()
	}
	item, err := s.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return CronJobDetail{}, mapKubernetesError(err, "get cronjob failed")
	}
	jobList, err := s.client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return CronJobDetail{}, mapKubernetesError(err, "list cronjob jobs failed")
	}
	podList, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return CronJobDetail{}, mapKubernetesError(err, "list cronjob pods failed")
	}
	eventList, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return CronJobDetail{}, mapKubernetesError(err, "list cronjob events failed")
	}
	return cronJobDetail(*item, jobList.Items, podList.Items, eventList.Items), nil
}

func (s *Service) ListServices(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[ServiceSummary], error) {
	if s.client == nil {
		return pagination.Result[ServiceSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[ServiceSummary]{}, mapKubernetesError(err, "list services failed")
	}
	services := make([]ServiceSummary, 0, len(items.Items))
	for _, item := range items.Items {
		services = append(services, serviceSummary(item))
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].Namespace == services[j].Namespace {
			return services[i].Name < services[j].Name
		}
		return services[i].Namespace < services[j].Namespace
	})
	return paginate(services, opts), nil
}

func (s *Service) GetService(ctx context.Context, namespace, name string) (ServiceDetail, error) {
	if s.client == nil {
		return ServiceDetail{}, unavailable()
	}
	item, err := s.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return ServiceDetail{}, mapKubernetesError(err, "get service failed")
	}
	slices, err := s.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: klabels.Set{discoveryv1.LabelServiceName: name}.AsSelector().String(),
	})
	if err != nil {
		return ServiceDetail{}, mapKubernetesError(err, "list service endpoint slices failed")
	}
	legacyEndpoints, err := s.client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return ServiceDetail{}, mapKubernetesError(err, "get service endpoints failed")
	}
	if apierrors.IsNotFound(err) {
		legacyEndpoints = nil
	}
	pods, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ServiceDetail{}, mapKubernetesError(err, "list service pods failed")
	}
	events, err := s.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ServiceDetail{}, mapKubernetesError(err, "list service events failed")
	}
	return serviceDetail(*item, slices.Items, legacyEndpoints, pods.Items, events.Items), nil
}

func (s *Service) ListIngresses(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[IngressSummary], error) {
	if s.client == nil {
		return pagination.Result[IngressSummary]{}, unavailable()
	}
	items, err := s.client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[IngressSummary]{}, mapKubernetesError(err, "list ingresses failed")
	}
	ingresses := make([]IngressSummary, 0, len(items.Items))
	for _, item := range items.Items {
		ingresses = append(ingresses, ingressSummary(item))
	}
	sort.Slice(ingresses, func(i, j int) bool {
		if ingresses[i].Namespace == ingresses[j].Namespace {
			return ingresses[i].Name < ingresses[j].Name
		}
		return ingresses[i].Namespace < ingresses[j].Namespace
	})
	return paginate(ingresses, opts), nil
}

func (s *Service) GetIngress(ctx context.Context, namespace, name string) (IngressDetail, error) {
	if s.client == nil {
		return IngressDetail{}, unavailable()
	}
	item, err := s.client.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "get ingress failed")
	}
	services, err := s.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "list ingress backend services failed")
	}
	slices, err := s.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "list ingress backend endpoint slices failed")
	}
	endpoints, err := s.client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "list ingress backend endpoints failed")
	}
	pods, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "list ingress backend pods failed")
	}
	events, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return IngressDetail{}, mapKubernetesError(err, "list ingress events failed")
	}
	return ingressDetail(*item, services.Items, slices.Items, endpoints.Items, pods.Items, events.Items), nil
}

func (s *Service) ListConfigMaps(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[ConfigMapSummary], error) {
	if s.client == nil {
		return pagination.Result[ConfigMapSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[ConfigMapSummary]{}, mapKubernetesError(err, "list configmaps failed")
	}
	configMaps := make([]ConfigMapSummary, 0, len(items.Items))
	for _, item := range items.Items {
		configMaps = append(configMaps, configMapSummary(item))
	}
	sort.Slice(configMaps, func(i, j int) bool {
		if configMaps[i].Namespace == configMaps[j].Namespace {
			return configMaps[i].Name < configMaps[j].Name
		}
		return configMaps[i].Namespace < configMaps[j].Namespace
	})
	return paginate(configMaps, opts), nil
}

func (s *Service) ListPVCs(ctx context.Context, namespace string, opts ListOptions) (pagination.Result[PVCSummary], error) {
	if s.client == nil {
		return pagination.Result[PVCSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[PVCSummary]{}, mapKubernetesError(err, "list persistentvolumeclaims failed")
	}
	pvcs := make([]PVCSummary, 0, len(items.Items))
	for _, item := range items.Items {
		pvcs = append(pvcs, pvcSummary(item))
	}
	sort.Slice(pvcs, func(i, j int) bool {
		if pvcs[i].Namespace == pvcs[j].Namespace {
			return pvcs[i].Name < pvcs[j].Name
		}
		return pvcs[i].Namespace < pvcs[j].Namespace
	})
	return paginate(pvcs, opts), nil
}

func (s *Service) GetPVC(ctx context.Context, namespace, name string) (PVCDetail, error) {
	if s.client == nil {
		return PVCDetail{}, unavailable()
	}
	pvc, err := s.client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return PVCDetail{}, mapKubernetesError(err, "get persistent volume claim failed")
	}

	var pv *corev1.PersistentVolume
	if pvc.Spec.VolumeName != "" {
		candidate, getErr := s.client.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
		switch {
		case getErr == nil && persistentVolumeClaims(candidate, pvc):
			pv = candidate
		case getErr != nil && !apierrors.IsNotFound(getErr):
			return PVCDetail{}, mapKubernetesError(getErr, "get bound persistent volume failed")
		}
	}

	pods, err := s.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return PVCDetail{}, mapKubernetesError(err, "list persistent volume claim pods failed")
	}
	events, err := s.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return PVCDetail{}, mapKubernetesError(err, "list persistent volume claim events failed")
	}
	return s.pvcDetail(ctx, *pvc, pv, pods.Items, events.Items), nil
}

func (s *Service) ListPVs(ctx context.Context, opts ListOptions) (pagination.Result[PVSummary], error) {
	if s.client == nil {
		return pagination.Result[PVSummary]{}, unavailable()
	}
	items, err := s.client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[PVSummary]{}, mapKubernetesError(err, "list persistentvolumes failed")
	}
	pvs := make([]PVSummary, 0, len(items.Items))
	for _, item := range items.Items {
		pvs = append(pvs, pvSummary(item))
	}
	sort.Slice(pvs, func(i, j int) bool {
		return pvs[i].Name < pvs[j].Name
	})
	return paginate(pvs, opts), nil
}

func (s *Service) GetPV(ctx context.Context, name string) (PVDetail, error) {
	if s.client == nil {
		return PVDetail{}, unavailable()
	}
	pv, err := s.client.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return PVDetail{}, mapKubernetesError(err, "get persistent volume failed")
	}

	var pvc *corev1.PersistentVolumeClaim
	if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace != "" && pv.Spec.ClaimRef.Name != "" {
		candidate, getErr := s.client.CoreV1().PersistentVolumeClaims(pv.Spec.ClaimRef.Namespace).Get(ctx, pv.Spec.ClaimRef.Name, metav1.GetOptions{})
		switch {
		case getErr == nil && persistentVolumeClaims(pv, candidate):
			pvc = candidate
		case getErr != nil && !apierrors.IsNotFound(getErr):
			return PVDetail{}, mapKubernetesError(getErr, "get bound persistent volume claim failed")
		}
	}

	pods := []corev1.Pod{}
	if pvc != nil {
		items, listErr := s.client.CoreV1().Pods(pvc.Namespace).List(ctx, metav1.ListOptions{})
		if listErr != nil {
			return PVDetail{}, mapKubernetesError(listErr, "list persistent volume pods failed")
		}
		pods = items.Items
	}
	events, err := s.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return PVDetail{}, mapKubernetesError(err, "list persistent volume events failed")
	}
	return s.pvDetail(ctx, *pv, pvc, pods, events.Items), nil
}

func (s *Service) ListStorageClasses(ctx context.Context, opts ListOptions) (pagination.Result[StorageClassSummary], error) {
	if s.client == nil {
		return pagination.Result[StorageClassSummary]{}, unavailable()
	}
	items, err := s.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return pagination.Result[StorageClassSummary]{}, mapKubernetesError(err, "list storageclasses failed")
	}
	storageClasses := make([]StorageClassSummary, 0, len(items.Items))
	for _, item := range items.Items {
		storageClasses = append(storageClasses, storageClassSummary(item))
	}
	sort.Slice(storageClasses, func(i, j int) bool {
		return storageClasses[i].Name < storageClasses[j].Name
	})
	return paginate(storageClasses, opts), nil
}

func (s *Service) ResourceYAML(ctx context.Context, kind, namespace, name string) (string, error) {
	if s.client == nil {
		return "", unavailable()
	}
	kind = normalizeResourceKind(kind)
	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)
	if kind == "" || name == "" {
		return "", apperrors.New(apperrors.CodeInvalidArgument, "kind and name are required", http.StatusBadRequest)
	}
	var obj any
	var err error

	switch kind {
	case "namespace":
		obj, err = s.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	case "node":
		obj, err = s.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	case "pod":
		obj, err = s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		obj, err = s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset":
		obj, err = s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "daemonset":
		obj, err = s.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "replicaset":
		obj, err = s.client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "job":
		obj, err = s.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "cronjob":
		obj, err = s.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "service":
		obj, err = s.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	case "ingress":
		obj, err = s.client.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	case "configmap":
		obj, err = s.client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	case "pvc":
		obj, err = s.client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	case "pv":
		obj, err = s.client.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	case "storageclass":
		obj, err = s.client.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	default:
		return "", apperrors.New(apperrors.CodeInvalidArgument, fmt.Sprintf("unsupported resource kind %q", kind), http.StatusBadRequest)
	}
	if err != nil {
		return "", mapKubernetesError(err, "get resource yaml failed")
	}
	return marshalResourceYAML(obj)
}

func (s *Service) UpdateResourceYAML(ctx context.Context, req ResourceYAMLUpdateRequest) (ResourceYAMLUpdateResult, error) {
	if s.client == nil {
		return ResourceYAMLUpdateResult{}, unavailable()
	}
	kind := normalizeResourceKind(req.Kind)
	namespace := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	rawYAML := strings.TrimSpace(req.YAML)
	if kind == "" || name == "" || rawYAML == "" {
		return ResourceYAMLUpdateResult{}, apperrors.New(apperrors.CodeInvalidArgument, "kind, name and yaml are required", http.StatusBadRequest)
	}
	if namespace == "" {
		return ResourceYAMLUpdateResult{}, apperrors.New(apperrors.CodeInvalidArgument, "namespace is required for yaml update", http.StatusBadRequest)
	}

	var updated any
	var err error
	switch kind {
	case "deployment":
		var obj appsv1.Deployment
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("Deployment", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.AppsV1().Deployments(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "statefulset":
		var obj appsv1.StatefulSet
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("StatefulSet", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.AppsV1().StatefulSets(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "daemonset":
		var obj appsv1.DaemonSet
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("DaemonSet", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.AppsV1().DaemonSets(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "job":
		var obj batchv1.Job
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("Job", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.BatchV1().Jobs(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "cronjob":
		var obj batchv1.CronJob
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("CronJob", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.BatchV1().CronJobs(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "service":
		var obj corev1.Service
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("Service", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.CoreV1().Services(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	case "ingress":
		var obj networkingv1.Ingress
		if err := decodeResourceYAML(rawYAML, &obj); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		if err := validateYAMLObject("Ingress", namespace, name, &obj.ObjectMeta, obj.Kind); err != nil {
			return ResourceYAMLUpdateResult{}, err
		}
		updated, err = s.client.NetworkingV1().Ingresses(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	default:
		return ResourceYAMLUpdateResult{}, apperrors.New(apperrors.CodeInvalidArgument, fmt.Sprintf("yaml update does not support resource kind %q", kind), http.StatusBadRequest)
	}
	if err != nil {
		return ResourceYAMLUpdateResult{}, mapKubernetesError(err, "update resource yaml failed")
	}
	rendered, err := marshalResourceYAML(updated)
	if err != nil {
		return ResourceYAMLUpdateResult{}, err
	}
	return ResourceYAMLUpdateResult{
		Kind:      resourceKindDisplay(kind),
		Namespace: namespace,
		Name:      name,
		Operation: "update",
		UpdatedAt: time.Now().UTC(),
		YAML:      rendered,
	}, nil
}

func normalizeResourceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "namespace", "namespaces":
		return "namespace"
	case "node", "nodes":
		return "node"
	case "pod", "pods":
		return "pod"
	case "deployment", "deployments":
		return "deployment"
	case "statefulset", "statefulsets":
		return "statefulset"
	case "daemonset", "daemonsets":
		return "daemonset"
	case "replicaset", "replicasets":
		return "replicaset"
	case "job", "jobs":
		return "job"
	case "cronjob", "cronjobs":
		return "cronjob"
	case "service", "services":
		return "service"
	case "ingress", "ingresses":
		return "ingress"
	case "configmap", "configmaps":
		return "configmap"
	case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
		return "pvc"
	case "pv", "persistentvolume", "persistentvolumes":
		return "pv"
	case "storageclass", "storageclasses":
		return "storageclass"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func resourceKindDisplay(kind string) string {
	switch kind {
	case "deployment":
		return "Deployment"
	case "statefulset":
		return "StatefulSet"
	case "daemonset":
		return "DaemonSet"
	case "job":
		return "Job"
	case "cronjob":
		return "CronJob"
	case "service":
		return "Service"
	case "ingress":
		return "Ingress"
	default:
		return kind
	}
}

func decodeResourceYAML(raw string, obj any) error {
	if err := yaml.Unmarshal([]byte(raw), obj); err != nil {
		return apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid resource yaml", http.StatusBadRequest)
	}
	return nil
}

func validateYAMLObject(expectedKind, namespace, name string, meta metav1.Object, actualKind string) error {
	if !strings.EqualFold(actualKind, expectedKind) {
		return apperrors.New(apperrors.CodeInvalidArgument, fmt.Sprintf("yaml kind must be %s", expectedKind), http.StatusBadRequest)
	}
	if meta.GetName() != name {
		return apperrors.New(apperrors.CodeInvalidArgument, "yaml metadata.name must match request name", http.StatusBadRequest)
	}
	if meta.GetNamespace() != namespace {
		return apperrors.New(apperrors.CodeInvalidArgument, "yaml metadata.namespace must match request namespace", http.StatusBadRequest)
	}
	return nil
}

func marshalResourceYAML(obj any) (string, error) {
	switch item := obj.(type) {
	case *corev1.Namespace:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.Node:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Node"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.Pod:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *appsv1.Deployment:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *appsv1.StatefulSet:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *appsv1.DaemonSet:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *appsv1.ReplicaSet:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSet"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *batchv1.Job:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *batchv1.CronJob:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.Service:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *networkingv1.Ingress:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.ConfigMap:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.PersistentVolumeClaim:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *corev1.PersistentVolume:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolume"}
		out.ManagedFields = nil
		return marshalYAML(out)
	case *storagev1.StorageClass:
		out := item.DeepCopy()
		out.TypeMeta = metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"}
		out.ManagedFields = nil
		return marshalYAML(out)
	default:
		return marshalYAML(obj)
	}
}

func marshalYAML(obj any) (string, error) {
	raw, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func paginate[T any](items []T, opts ListOptions) pagination.Result[T] {
	params := pagination.Normalize(opts.Page, opts.PageSize)
	start := params.Offset()
	if start > len(items) {
		start = len(items)
	}
	end := start + params.PageSize
	if end > len(items) {
		end = len(items)
	}
	return pagination.Result[T]{
		Items:    items[start:end],
		Total:    int64(len(items)),
		Page:     params.Page,
		PageSize: params.PageSize,
	}
}

func nodeSummary(node corev1.Node) NodeSummary {
	status := "NotReady"
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			status = "Ready"
			break
		}
	}
	return NodeSummary{
		Name:            node.Name,
		Status:          status,
		CPUAllocatable:  node.Status.Allocatable.Cpu().String(),
		MemAllocatable:  node.Status.Allocatable.Memory().String(),
		PodsAllocatable: node.Status.Allocatable.Pods().String(),
		CreatedAt:       node.CreationTimestamp.Time,
	}
}

type declaredResourceQuantities struct {
	cpu              resource.Quantity
	memory           resource.Quantity
	ephemeralStorage resource.Quantity
	pods             int64
}

func newDeclaredResourceQuantities() declaredResourceQuantities {
	return declaredResourceQuantities{
		cpu:              *resource.NewMilliQuantity(0, resource.DecimalSI),
		memory:           *resource.NewQuantity(0, resource.BinarySI),
		ephemeralStorage: *resource.NewQuantity(0, resource.BinarySI),
	}
}

func (s *Service) namespaceDetail(
	ctx context.Context,
	namespace corev1.Namespace,
	pods []corev1.Pod,
	deployments []appsv1.Deployment,
	statefulSets []appsv1.StatefulSet,
	daemonSets []appsv1.DaemonSet,
	replicaSets []appsv1.ReplicaSet,
	jobs []batchv1.Job,
	cronJobs []batchv1.CronJob,
	services []corev1.Service,
	ingresses []networkingv1.Ingress,
	pvcs []corev1.PersistentVolumeClaim,
	configMaps []corev1.ConfigMap,
	events []corev1.Event,
) NamespaceDetail {
	detail := NamespaceDetail{
		NamespaceSummary: NamespaceSummary{
			Name:      namespace.Name,
			Status:    string(namespace.Status.Phase),
			CreatedAt: namespace.CreationTimestamp.Time,
		},
		Labels:     copyStringMap(namespace.Labels),
		Finalizers: make([]string, 0, len(namespace.Spec.Finalizers)),
		Conditions: make([]Condition, 0, len(namespace.Status.Conditions)),
		Counts: NamespaceResourceCounts{
			Pods:                   len(pods),
			Deployments:            len(deployments),
			StatefulSets:           len(statefulSets),
			DaemonSets:             len(daemonSets),
			ReplicaSets:            len(replicaSets),
			Jobs:                   len(jobs),
			CronJobs:               len(cronJobs),
			Services:               len(services),
			Ingresses:              len(ingresses),
			PersistentVolumeClaims: len(pvcs),
			ConfigMaps:             len(configMaps),
		},
		Pods:                   make([]PodSummary, 0, len(pods)),
		Services:               make([]ServiceSummary, 0, len(services)),
		Ingresses:              make([]IngressSummary, 0, len(ingresses)),
		PersistentVolumeClaims: make([]PVCSummary, 0, len(pvcs)),
	}
	for _, finalizer := range namespace.Spec.Finalizers {
		detail.Finalizers = append(detail.Finalizers, string(finalizer))
	}
	for _, condition := range namespace.Status.Conditions {
		detail.Conditions = append(detail.Conditions, Condition{
			Type:    string(condition.Type),
			Status:  string(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}

	requests := newDeclaredResourceQuantities()
	limits := newDeclaredResourceQuantities()
	workloads := make(map[string]ResourceReference)
	for _, pod := range pods {
		detail.Pods = append(detail.Pods, podSummary(pod))
		if podReady(pod) {
			detail.Counts.ReadyPods++
		}
		if isAbnormalPod(pod) {
			detail.Counts.AbnormalPods++
		}
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			podRequests, podLimits := podDeclaredResources(pod)
			requests.add(podRequests)
			limits.add(podLimits)
		}
		addTopWorkloadReference(workloads, namespace.Name, s.podControllerChain(ctx, pod))
	}
	detail.Allocated = ResourceAllocation{Requests: requests.summary(), Limits: limits.summary()}
	detail.Workloads = sortedResourceReferences(workloads)
	sort.Slice(detail.Pods, func(i, j int) bool { return detail.Pods[i].Name < detail.Pods[j].Name })

	for _, item := range services {
		detail.Services = append(detail.Services, serviceSummary(item))
	}
	for _, item := range ingresses {
		detail.Ingresses = append(detail.Ingresses, ingressSummary(item))
	}
	for _, item := range pvcs {
		detail.PersistentVolumeClaims = append(detail.PersistentVolumeClaims, pvcSummary(item))
	}
	sort.Slice(detail.Services, func(i, j int) bool { return detail.Services[i].Name < detail.Services[j].Name })
	sort.Slice(detail.Ingresses, func(i, j int) bool { return detail.Ingresses[i].Name < detail.Ingresses[j].Name })
	sort.Slice(detail.PersistentVolumeClaims, func(i, j int) bool {
		return detail.PersistentVolumeClaims[i].Name < detail.PersistentVolumeClaims[j].Name
	})

	for _, event := range events {
		if strings.EqualFold(event.InvolvedObject.Kind, "Namespace") && involvedObjectMatches(event.InvolvedObject, namespace.Name, string(namespace.UID)) {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func (s *Service) nodeDetail(ctx context.Context, node corev1.Node, pods []corev1.Pod, events []corev1.Event) NodeDetail {
	detail := NodeDetail{
		NodeSummary:   nodeSummary(node),
		Roles:         nodeRoles(node.Labels),
		Unschedulable: node.Spec.Unschedulable,
		PodCIDRs:      append([]string(nil), node.Spec.PodCIDRs...),
		Labels:        copyStringMap(node.Labels),
		Capacity:      declaredResourcesFromList(node.Status.Capacity),
		Allocatable:   declaredResourcesFromList(node.Status.Allocatable),
		SystemInfo: NodeSystemInfo{
			MachineID:               node.Status.NodeInfo.MachineID,
			SystemUUID:              node.Status.NodeInfo.SystemUUID,
			BootID:                  node.Status.NodeInfo.BootID,
			KernelVersion:           node.Status.NodeInfo.KernelVersion,
			OSImage:                 node.Status.NodeInfo.OSImage,
			ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        node.Status.NodeInfo.KubeProxyVersion,
			OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
			Architecture:            node.Status.NodeInfo.Architecture,
		},
	}
	if len(detail.PodCIDRs) == 0 && node.Spec.PodCIDR != "" {
		detail.PodCIDRs = []string{node.Spec.PodCIDR}
	}
	for _, address := range node.Status.Addresses {
		detail.Addresses = append(detail.Addresses, NodeAddressDetail{Type: string(address.Type), Address: address.Address})
	}
	for _, taint := range node.Spec.Taints {
		item := NodeTaintDetail{Key: taint.Key, Value: taint.Value, Effect: string(taint.Effect)}
		if taint.TimeAdded != nil {
			item.TimeAdded = taint.TimeAdded.Time
		}
		detail.Taints = append(detail.Taints, item)
	}
	for _, condition := range node.Status.Conditions {
		detail.Conditions = append(detail.Conditions, NodeConditionDetail{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastHeartbeatTime:  condition.LastHeartbeatTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}

	requests := newDeclaredResourceQuantities()
	limits := newDeclaredResourceQuantities()
	workloads := make(map[string]ResourceReference)
	for _, pod := range pods {
		if pod.Spec.NodeName != node.Name {
			continue
		}
		detail.Pods = append(detail.Pods, podSummary(pod))
		addTopWorkloadReference(workloads, pod.Namespace, s.podControllerChain(ctx, pod))
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		podRequests, podLimits := podDeclaredResources(pod)
		requests.add(podRequests)
		limits.add(podLimits)
	}
	detail.Allocated = ResourceAllocation{
		Requests:             requests.summary(),
		Limits:               limits.summary(),
		CPURequestPercent:    quantityPercent(requests.cpu, node.Status.Allocatable[corev1.ResourceCPU], true),
		MemoryRequestPercent: quantityPercent(requests.memory, node.Status.Allocatable[corev1.ResourceMemory], false),
		PodPercent:           countPercent(requests.pods, resourceListValue(node.Status.Allocatable, corev1.ResourcePods)),
	}
	detail.Workloads = sortedResourceReferences(workloads)
	sort.Slice(detail.Pods, func(i, j int) bool {
		if detail.Pods[i].Namespace == detail.Pods[j].Namespace {
			return detail.Pods[i].Name < detail.Pods[j].Name
		}
		return detail.Pods[i].Namespace < detail.Pods[j].Namespace
	})
	for _, event := range events {
		if strings.EqualFold(event.InvolvedObject.Kind, "Node") && involvedObjectMatches(event.InvolvedObject, node.Name, string(node.UID)) {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func podDeclaredResources(pod corev1.Pod) (declaredResourceQuantities, declaredResourceQuantities) {
	requests := newDeclaredResourceQuantities()
	limits := newDeclaredResourceQuantities()
	for _, container := range pod.Spec.Containers {
		requests.addResourceList(container.Resources.Requests)
		limits.addResourceList(container.Resources.Limits)
	}
	initRequests := newDeclaredResourceQuantities()
	initLimits := newDeclaredResourceQuantities()
	for _, container := range pod.Spec.InitContainers {
		initRequests.maxResourceList(container.Resources.Requests)
		initLimits.maxResourceList(container.Resources.Limits)
	}
	requests.max(initRequests)
	limits.max(initLimits)
	requests.addResourceList(pod.Spec.Overhead)
	limits.addResourceList(pod.Spec.Overhead)
	requests.pods = 1
	limits.pods = 1
	return requests, limits
}

func (items *declaredResourceQuantities) add(other declaredResourceQuantities) {
	items.cpu.Add(other.cpu)
	items.memory.Add(other.memory)
	items.ephemeralStorage.Add(other.ephemeralStorage)
	items.pods += other.pods
}

func (items *declaredResourceQuantities) addResourceList(list corev1.ResourceList) {
	if quantity, found := list[corev1.ResourceCPU]; found {
		items.cpu.Add(quantity.DeepCopy())
	}
	if quantity, found := list[corev1.ResourceMemory]; found {
		items.memory.Add(quantity.DeepCopy())
	}
	if quantity, found := list[corev1.ResourceEphemeralStorage]; found {
		items.ephemeralStorage.Add(quantity.DeepCopy())
	}
}

func (items *declaredResourceQuantities) max(other declaredResourceQuantities) {
	if other.cpu.Cmp(items.cpu) > 0 {
		items.cpu = other.cpu.DeepCopy()
	}
	if other.memory.Cmp(items.memory) > 0 {
		items.memory = other.memory.DeepCopy()
	}
	if other.ephemeralStorage.Cmp(items.ephemeralStorage) > 0 {
		items.ephemeralStorage = other.ephemeralStorage.DeepCopy()
	}
}

func (items *declaredResourceQuantities) maxResourceList(list corev1.ResourceList) {
	other := newDeclaredResourceQuantities()
	other.addResourceList(list)
	items.max(other)
}

func (items declaredResourceQuantities) summary() DeclaredResources {
	return DeclaredResources{
		CPU:              items.cpu.String(),
		Memory:           items.memory.String(),
		EphemeralStorage: items.ephemeralStorage.String(),
		Pods:             items.pods,
	}
}

func declaredResourcesFromList(list corev1.ResourceList) DeclaredResources {
	items := newDeclaredResourceQuantities()
	items.addResourceList(list)
	if pods, found := list[corev1.ResourcePods]; found {
		items.pods = pods.Value()
	}
	return items.summary()
}

func resourceListValue(list corev1.ResourceList, name corev1.ResourceName) int64 {
	quantity, found := list[name]
	if !found {
		return 0
	}
	return quantity.Value()
}

func quantityPercent(used, total resource.Quantity, milli bool) float64 {
	var numerator, denominator int64
	if milli {
		numerator, denominator = used.MilliValue(), total.MilliValue()
	} else {
		numerator, denominator = used.Value(), total.Value()
	}
	return countPercent(numerator, denominator)
}

func countPercent(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round((float64(used)/float64(total))*10000) / 100
}

func addTopWorkloadReference(items map[string]ResourceReference, namespace string, chain []OwnerReference) {
	if len(chain) == 0 {
		return
	}
	top := chain[len(chain)-1]
	key := resourceReferenceKey(top.Kind, namespace+"/"+top.Name)
	items[key] = ResourceReference{Kind: top.Kind, Namespace: namespace, Name: top.Name}
}

func sortedResourceReferences(items map[string]ResourceReference) []ResourceReference {
	result := make([]ResourceReference, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func nodeRoles(labels map[string]string) []string {
	const prefix = "node-role.kubernetes.io/"
	roles := make([]string, 0)
	for key := range labels {
		if strings.HasPrefix(key, prefix) && strings.TrimPrefix(key, prefix) != "" {
			roles = append(roles, strings.TrimPrefix(key, prefix))
		}
	}
	sort.Strings(roles)
	return roles
}

func copyStringMap(items map[string]string) map[string]string {
	result := make(map[string]string, len(items))
	for key, value := range items {
		result[key] = value
	}
	return result
}

func podSummary(pod corev1.Pod) PodSummary {
	containers := make([]ContainerSummary, 0, len(pod.Status.ContainerStatuses))
	var restarts int32
	for _, status := range pod.Status.ContainerStatuses {
		restarts += status.RestartCount
		containers = append(containers, containerSummary(status))
	}
	return PodSummary{
		Namespace:    pod.Namespace,
		Name:         pod.Name,
		Phase:        string(pod.Status.Phase),
		Status:       podStatus(pod),
		NodeName:     pod.Spec.NodeName,
		Ready:        podReady(pod),
		RestartCount: restarts,
		Containers:   containers,
		CreatedAt:    pod.CreationTimestamp.Time,
	}
}

func podDetail(pod corev1.Pod, controllerChain []OwnerReference) PodDetail {
	summary := podSummary(pod)
	owners := make([]OwnerReference, 0, len(pod.OwnerReferences))
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, ownerReference(owner))
	}
	initContainers := make([]ContainerSummary, 0, len(pod.Status.InitContainerStatuses))
	for _, status := range pod.Status.InitContainerStatuses {
		initContainers = append(initContainers, containerSummary(status))
	}
	conditions := make([]Condition, 0, len(pod.Status.Conditions))
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, Condition{
			Type:    string(condition.Type),
			Status:  string(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}
	var startTime time.Time
	if pod.Status.StartTime != nil {
		startTime = pod.Status.StartTime.Time
	}
	return PodDetail{
		PodSummary:      summary,
		PodIP:           pod.Status.PodIP,
		HostIP:          pod.Status.HostIP,
		QoSClass:        string(pod.Status.QOSClass),
		ServiceAccount:  pod.Spec.ServiceAccountName,
		RestartPolicy:   string(pod.Spec.RestartPolicy),
		StartTime:       startTime,
		Labels:          pod.Labels,
		Annotations:     pod.Annotations,
		OwnerRefs:       owners,
		ControllerChain: controllerChain,
		InitContainers:  initContainers,
		Conditions:      conditions,
	}
}

func containerSummary(status corev1.ContainerStatus) ContainerSummary {
	state := "Unknown"
	reason := ""
	var exitCode int32
	var startedAt time.Time
	var finishedAt time.Time
	switch {
	case status.State.Running != nil:
		state = "Running"
		startedAt = status.State.Running.StartedAt.Time
	case status.State.Waiting != nil:
		state = "Waiting"
		reason = status.State.Waiting.Reason
	case status.State.Terminated != nil:
		state = "Terminated"
		reason = status.State.Terminated.Reason
		exitCode = status.State.Terminated.ExitCode
		startedAt = status.State.Terminated.StartedAt.Time
		finishedAt = status.State.Terminated.FinishedAt.Time
	}
	lastState := ""
	lastReason := ""
	var lastExitCode int32
	var lastFinishedAt time.Time
	switch {
	case status.LastTerminationState.Running != nil:
		lastState = "Running"
	case status.LastTerminationState.Waiting != nil:
		lastState = "Waiting"
		lastReason = status.LastTerminationState.Waiting.Reason
	case status.LastTerminationState.Terminated != nil:
		lastState = "Terminated"
		lastReason = status.LastTerminationState.Terminated.Reason
		lastExitCode = status.LastTerminationState.Terminated.ExitCode
		lastFinishedAt = status.LastTerminationState.Terminated.FinishedAt.Time
	}
	return ContainerSummary{
		Name:           status.Name,
		Image:          status.Image,
		Ready:          status.Ready,
		RestartCount:   status.RestartCount,
		State:          state,
		Reason:         reason,
		ExitCode:       exitCode,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		LastState:      lastState,
		LastReason:     lastReason,
		LastExitCode:   lastExitCode,
		LastFinishedAt: lastFinishedAt,
	}
}

func controllerOwnerReference(items []metav1.OwnerReference) (metav1.OwnerReference, bool) {
	for _, item := range items {
		if item.Controller != nil && *item.Controller {
			return item, true
		}
	}
	return metav1.OwnerReference{}, false
}

func ownerReference(item metav1.OwnerReference) OwnerReference {
	return OwnerReference{
		Kind:       item.Kind,
		Name:       item.Name,
		Controller: item.Controller != nil && *item.Controller,
	}
}

func podReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func podStatus(pod corev1.Pod) string {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return "Init:" + status.State.Waiting.Reason
		}
		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			reason := status.State.Terminated.Reason
			if reason == "" {
				reason = fmt.Sprintf("ExitCode%d", status.State.Terminated.ExitCode)
			}
			return "Init:" + reason
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return status.State.Waiting.Reason
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" && status.State.Terminated.Reason != "Completed" {
			return status.State.Terminated.Reason
		}
	}
	return string(pod.Status.Phase)
}

func isAbnormalPod(pod corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
		return true
	}
	if pod.Status.Phase == corev1.PodSucceeded {
		return false
	}
	if !podReady(pod) {
		return true
	}
	statuses := append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...)
	statuses = append(statuses, pod.Status.ContainerStatuses...)
	for _, status := range statuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return true
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" && status.State.Terminated.Reason != "Completed" {
			return true
		}
	}
	return false
}

func eventSummary(event corev1.Event) EventSummary {
	source := event.Source.Component
	if event.Source.Host != "" {
		if source == "" {
			source = event.Source.Host
		} else {
			source += "/" + event.Source.Host
		}
	}
	firstTimestamp := event.FirstTimestamp.Time
	lastTimestamp := event.LastTimestamp.Time
	if firstTimestamp.IsZero() && !event.EventTime.IsZero() {
		firstTimestamp = event.EventTime.Time
	}
	if lastTimestamp.IsZero() {
		switch {
		case event.Series != nil && !event.Series.LastObservedTime.IsZero():
			lastTimestamp = event.Series.LastObservedTime.Time
		case !event.EventTime.IsZero():
			lastTimestamp = event.EventTime.Time
		case !event.CreationTimestamp.IsZero():
			lastTimestamp = event.CreationTimestamp.Time
		}
	}
	if firstTimestamp.IsZero() {
		firstTimestamp = lastTimestamp
	}
	return EventSummary{
		Type:           event.Type,
		Reason:         event.Reason,
		Message:        event.Message,
		Count:          event.Count,
		Namespace:      event.Namespace,
		InvolvedKind:   event.InvolvedObject.Kind,
		InvolvedName:   event.InvolvedObject.Name,
		Source:         source,
		FirstTimestamp: firstTimestamp,
		LastTimestamp:  lastTimestamp,
	}
}

func DeploymentSummaryFrom(deployment appsv1.Deployment) DeploymentSummary {
	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	images := make([]string, 0, len(deployment.Spec.Template.Spec.Containers))
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}
	return DeploymentSummary{
		Namespace:           deployment.Namespace,
		Name:                deployment.Name,
		Replicas:            replicas,
		ReadyReplicas:       deployment.Status.ReadyReplicas,
		UpdatedReplicas:     deployment.Status.UpdatedReplicas,
		AvailableReplicas:   deployment.Status.AvailableReplicas,
		UnavailableReplicas: deployment.Status.UnavailableReplicas,
		Labels:              deployment.Labels,
		Images:              images,
		CreatedAt:           deployment.CreationTimestamp.Time,
	}
}

func deploymentDetail(deployment appsv1.Deployment, replicaSets []appsv1.ReplicaSet, pods []corev1.Pod, events []corev1.Event) DeploymentDetail {
	ownedReplicaSets := make([]ReplicaSetDetail, 0)
	replicaSetUIDs := make(map[string]string)
	for _, item := range replicaSets {
		if !hasControllerOwner(item.OwnerReferences, "Deployment", deployment.Name, string(deployment.UID)) {
			continue
		}
		ownedReplicaSets = append(ownedReplicaSets, replicaSetDetail(item))
		replicaSetUIDs[item.Name] = string(item.UID)
	}
	sort.Slice(ownedReplicaSets, func(i, j int) bool {
		if ownedReplicaSets[i].CreatedAt.Equal(ownedReplicaSets[j].CreatedAt) {
			return ownedReplicaSets[i].Name < ownedReplicaSets[j].Name
		}
		return ownedReplicaSets[i].CreatedAt.After(ownedReplicaSets[j].CreatedAt)
	})

	ownedPods := make([]PodSummary, 0)
	podUIDs := make(map[string]string)
	for _, pod := range pods {
		controller, ok := controllerOwnerReference(pod.OwnerReferences)
		if !ok || !strings.EqualFold(controller.Kind, "ReplicaSet") {
			continue
		}
		replicaSetUID, found := replicaSetUIDs[controller.Name]
		if !found || (controller.UID != "" && replicaSetUID != "" && string(controller.UID) != replicaSetUID) {
			continue
		}
		ownedPods = append(ownedPods, podSummary(pod))
		podUIDs[pod.Name] = string(pod.UID)
	}
	sort.Slice(ownedPods, func(i, j int) bool {
		return ownedPods[i].Name < ownedPods[j].Name
	})

	relatedEvents := make([]EventSummary, 0)
	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "deployment":
			related = involvedObjectMatches(event.InvolvedObject, deployment.Name, string(deployment.UID))
		case "replicaset":
			if uid, found := replicaSetUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			relatedEvents = append(relatedEvents, eventSummary(event))
		}
	}
	sort.Slice(relatedEvents, func(i, j int) bool {
		return relatedEvents[i].LastTimestamp.After(relatedEvents[j].LastTimestamp)
	})

	conditions := make([]WorkloadCondition, 0, len(deployment.Status.Conditions))
	for _, condition := range deployment.Status.Conditions {
		conditions = append(conditions, WorkloadCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastUpdateTime:     condition.LastUpdateTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}

	selector := map[string]string{}
	if deployment.Spec.Selector != nil {
		for key, value := range deployment.Spec.Selector.MatchLabels {
			selector[key] = value
		}
	}
	maxSurge := ""
	maxUnavailable := ""
	if deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = deployment.Spec.Strategy.RollingUpdate.MaxSurge.String()
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
	}
	progressDeadlineSeconds := int32(0)
	if deployment.Spec.ProgressDeadlineSeconds != nil {
		progressDeadlineSeconds = *deployment.Spec.ProgressDeadlineSeconds
	}
	revisionHistoryLimit := int32(0)
	if deployment.Spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = *deployment.Spec.RevisionHistoryLimit
	}

	return DeploymentDetail{
		DeploymentSummary:       DeploymentSummaryFrom(deployment),
		Generation:              deployment.Generation,
		ObservedGeneration:      deployment.Status.ObservedGeneration,
		Paused:                  deployment.Spec.Paused,
		Strategy:                string(deployment.Spec.Strategy.Type),
		MaxSurge:                maxSurge,
		MaxUnavailable:          maxUnavailable,
		MinReadySeconds:         deployment.Spec.MinReadySeconds,
		ProgressDeadlineSeconds: progressDeadlineSeconds,
		RevisionHistoryLimit:    revisionHistoryLimit,
		Selector:                selector,
		Conditions:              conditions,
		ReplicaSets:             ownedReplicaSets,
		Pods:                    ownedPods,
		Events:                  relatedEvents,
	}
}

func replicaSetDetail(item appsv1.ReplicaSet) ReplicaSetDetail {
	selector := map[string]string{}
	if item.Spec.Selector != nil {
		for key, value := range item.Spec.Selector.MatchLabels {
			selector[key] = value
		}
	}
	conditions := make([]WorkloadCondition, 0, len(item.Status.Conditions))
	for _, condition := range item.Status.Conditions {
		conditions = append(conditions, WorkloadCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}
	var owner *OwnerReference
	if controller, ok := controllerOwnerReference(item.OwnerReferences); ok {
		value := ownerReference(controller)
		owner = &value
	}
	return ReplicaSetDetail{
		WorkloadSummary:      replicaSetSummary(item),
		Revision:             item.Annotations["deployment.kubernetes.io/revision"],
		CurrentReplicas:      item.Status.Replicas,
		FullyLabeledReplicas: item.Status.FullyLabeledReplicas,
		ObservedGeneration:   item.Status.ObservedGeneration,
		MinReadySeconds:      item.Spec.MinReadySeconds,
		Selector:             selector,
		Owner:                owner,
		Conditions:           conditions,
	}
}

func replicaSetDiagnosticDetail(item appsv1.ReplicaSet, pods []corev1.Pod, events []corev1.Event) ReplicaSetDetail {
	detail := replicaSetDetail(item)
	podUIDs := make(map[string]string)
	for _, pod := range pods {
		if !hasControllerOwner(pod.OwnerReferences, "ReplicaSet", item.Name, string(item.UID)) {
			continue
		}
		detail.Pods = append(detail.Pods, podSummary(pod))
		podUIDs[pod.Name] = string(pod.UID)
	}
	sort.Slice(detail.Pods, func(i, j int) bool {
		return detail.Pods[i].Name < detail.Pods[j].Name
	})

	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "replicaset":
			related = involvedObjectMatches(event.InvolvedObject, item.Name, string(item.UID))
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func hasControllerOwner(items []metav1.OwnerReference, kind, name, uid string) bool {
	controller, ok := controllerOwnerReference(items)
	if !ok || !strings.EqualFold(controller.Kind, kind) || controller.Name != name {
		return false
	}
	return controller.UID == "" || uid == "" || string(controller.UID) == uid
}

func involvedObjectMatches(reference corev1.ObjectReference, name, uid string) bool {
	if reference.Name != name {
		return false
	}
	return reference.UID == "" || uid == "" || string(reference.UID) == uid
}

func StatefulSetSummaryFrom(item appsv1.StatefulSet) WorkloadSummary {
	replicas := int32(0)
	if item.Spec.Replicas != nil {
		replicas = *item.Spec.Replicas
	}
	unavailableReplicas := replicas - item.Status.ReadyReplicas
	if unavailableReplicas < 0 {
		unavailableReplicas = 0
	}
	return WorkloadSummary{
		Kind:                "StatefulSet",
		Namespace:           item.Namespace,
		Name:                item.Name,
		Replicas:            replicas,
		ReadyReplicas:       item.Status.ReadyReplicas,
		AvailableReplicas:   item.Status.AvailableReplicas,
		UnavailableReplicas: unavailableReplicas,
		Labels:              item.Labels,
		Images:              podTemplateImages(item.Spec.Template),
		CreatedAt:           item.CreationTimestamp.Time,
	}
}

func StatefulSetDetailFrom(item appsv1.StatefulSet) StatefulSetDetail {
	claims := make([]StatefulSetVolumeClaim, 0, len(item.Spec.VolumeClaimTemplates))
	for _, claim := range item.Spec.VolumeClaimTemplates {
		storageClass := ""
		if claim.Spec.StorageClassName != nil {
			storageClass = *claim.Spec.StorageClassName
		}
		requestedStorage := ""
		if quantity, ok := claim.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			requestedStorage = quantity.String()
		}
		accessModes := make([]string, 0, len(claim.Spec.AccessModes))
		for _, mode := range claim.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}
		claims = append(claims, StatefulSetVolumeClaim{
			Name:             claim.Name,
			StorageClass:     storageClass,
			RequestedStorage: requestedStorage,
			AccessModes:      accessModes,
		})
	}

	selector := map[string]string{}
	if item.Spec.Selector != nil {
		for key, value := range item.Spec.Selector.MatchLabels {
			selector[key] = value
		}
	}
	return StatefulSetDetail{
		WorkloadSummary:     StatefulSetSummaryFrom(item),
		ServiceName:         item.Spec.ServiceName,
		PodManagementPolicy: string(item.Spec.PodManagementPolicy),
		UpdateStrategy:      string(item.Spec.UpdateStrategy.Type),
		CurrentRevision:     item.Status.CurrentRevision,
		UpdateRevision:      item.Status.UpdateRevision,
		CurrentReplicas:     item.Status.CurrentReplicas,
		UpdatedReplicas:     item.Status.UpdatedReplicas,
		Selector:            selector,
		VolumeClaims:        claims,
	}
}

func replicaSetSummary(item appsv1.ReplicaSet) WorkloadSummary {
	replicas := int32(0)
	if item.Spec.Replicas != nil {
		replicas = *item.Spec.Replicas
	}
	unavailableReplicas := replicas - item.Status.ReadyReplicas
	if unavailableReplicas < 0 {
		unavailableReplicas = 0
	}
	return WorkloadSummary{
		Kind:                "ReplicaSet",
		Namespace:           item.Namespace,
		Name:                item.Name,
		Replicas:            replicas,
		ReadyReplicas:       item.Status.ReadyReplicas,
		AvailableReplicas:   item.Status.AvailableReplicas,
		UnavailableReplicas: unavailableReplicas,
		Labels:              item.Labels,
		Images:              podTemplateImages(item.Spec.Template),
		CreatedAt:           item.CreationTimestamp.Time,
	}
}

func DaemonSetSummaryFrom(item appsv1.DaemonSet) DaemonSetSummary {
	return DaemonSetSummary{
		Namespace:          item.Namespace,
		Name:               item.Name,
		DesiredNumber:      item.Status.DesiredNumberScheduled,
		CurrentNumber:      item.Status.CurrentNumberScheduled,
		ReadyNumber:        item.Status.NumberReady,
		UpdatedNumber:      item.Status.UpdatedNumberScheduled,
		AvailableNumber:    item.Status.NumberAvailable,
		UnavailableNumber:  item.Status.NumberUnavailable,
		MisscheduledNumber: item.Status.NumberMisscheduled,
		Labels:             item.Labels,
		Images:             podTemplateImages(item.Spec.Template),
		CreatedAt:          item.CreationTimestamp.Time,
	}
}

func daemonSetDetail(item appsv1.DaemonSet) DaemonSetDetail {
	selector := map[string]string{}
	if item.Spec.Selector != nil {
		for key, value := range item.Spec.Selector.MatchLabels {
			selector[key] = value
		}
	}
	nodeSelector := map[string]string{}
	for key, value := range item.Spec.Template.Spec.NodeSelector {
		nodeSelector[key] = value
	}
	tolerations := make([]string, 0, len(item.Spec.Template.Spec.Tolerations))
	for _, toleration := range item.Spec.Template.Spec.Tolerations {
		tolerations = append(tolerations, formatToleration(toleration))
	}
	return DaemonSetDetail{
		DaemonSetSummary: DaemonSetSummaryFrom(item),
		UpdateStrategy:   string(item.Spec.UpdateStrategy.Type),
		Selector:         selector,
		NodeSelector:     nodeSelector,
		Tolerations:      tolerations,
	}
}

func formatToleration(item corev1.Toleration) string {
	operator := item.Operator
	if operator == "" {
		operator = corev1.TolerationOpEqual
	}
	value := item.Key
	if operator == corev1.TolerationOpExists {
		value += ":Exists"
	} else {
		value += "=" + item.Value
	}
	if item.Effect != "" {
		value += ":" + string(item.Effect)
	}
	return value
}

func jobSummary(item batchv1.Job) JobSummary {
	completions := int32(0)
	if item.Spec.Completions != nil {
		completions = *item.Spec.Completions
	}
	var startTime time.Time
	if item.Status.StartTime != nil {
		startTime = item.Status.StartTime.Time
	}
	var completionTime time.Time
	if item.Status.CompletionTime != nil {
		completionTime = item.Status.CompletionTime.Time
	}
	return JobSummary{
		Namespace:      item.Namespace,
		Name:           item.Name,
		Completions:    completions,
		Succeeded:      item.Status.Succeeded,
		Failed:         item.Status.Failed,
		Active:         item.Status.Active,
		StartTime:      startTime,
		CompletionTime: completionTime,
		CreatedAt:      item.CreationTimestamp.Time,
	}
}

func jobDetail(item batchv1.Job, pods []corev1.Pod, events []corev1.Event) JobDetail {
	detail := JobDetail{
		JobSummary:     jobSummary(item),
		CompletionMode: string(batchv1.NonIndexedCompletion),
		Selector:       map[string]string{},
		Images:         podTemplateImages(item.Spec.Template),
	}
	if item.Spec.Parallelism != nil {
		detail.Parallelism = *item.Spec.Parallelism
	}
	if item.Spec.BackoffLimit != nil {
		detail.BackoffLimit = *item.Spec.BackoffLimit
	}
	if item.Spec.ActiveDeadlineSeconds != nil {
		detail.ActiveDeadlineSeconds = *item.Spec.ActiveDeadlineSeconds
	}
	if item.Spec.TTLSecondsAfterFinished != nil {
		detail.TTLSecondsAfterFinished = *item.Spec.TTLSecondsAfterFinished
	}
	if item.Spec.CompletionMode != nil {
		detail.CompletionMode = string(*item.Spec.CompletionMode)
	}
	if item.Spec.Suspend != nil {
		detail.Suspend = *item.Spec.Suspend
	}
	if item.Spec.ManualSelector != nil {
		detail.ManualSelector = *item.Spec.ManualSelector
	}
	if item.Spec.Selector != nil {
		for key, value := range item.Spec.Selector.MatchLabels {
			detail.Selector[key] = value
		}
	}
	if controller, ok := controllerOwnerReference(item.OwnerReferences); ok {
		value := ownerReference(controller)
		detail.Owner = &value
	}
	for _, condition := range item.Status.Conditions {
		detail.Conditions = append(detail.Conditions, WorkloadCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}

	podUIDs := make(map[string]string)
	for _, pod := range pods {
		if !hasControllerOwner(pod.OwnerReferences, "Job", item.Name, string(item.UID)) {
			continue
		}
		detail.Pods = append(detail.Pods, podSummary(pod))
		podUIDs[pod.Name] = string(pod.UID)
	}
	sort.Slice(detail.Pods, func(i, j int) bool {
		return detail.Pods[i].CreatedAt.After(detail.Pods[j].CreatedAt)
	})

	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "job":
			related = involvedObjectMatches(event.InvolvedObject, item.Name, string(item.UID))
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func CronJobSummaryFrom(item batchv1.CronJob) CronJobSummary {
	suspend := false
	if item.Spec.Suspend != nil {
		suspend = *item.Spec.Suspend
	}
	var lastScheduleTime time.Time
	if item.Status.LastScheduleTime != nil {
		lastScheduleTime = item.Status.LastScheduleTime.Time
	}
	return CronJobSummary{
		Namespace:        item.Namespace,
		Name:             item.Name,
		Schedule:         item.Spec.Schedule,
		Suspend:          suspend,
		Active:           len(item.Status.Active),
		LastScheduleTime: lastScheduleTime,
		CreatedAt:        item.CreationTimestamp.Time,
	}
}

func cronJobDetail(item batchv1.CronJob, jobs []batchv1.Job, pods []corev1.Pod, events []corev1.Event) CronJobDetail {
	detail := CronJobDetail{
		CronJobSummary:             CronJobSummaryFrom(item),
		ConcurrencyPolicy:          string(batchv1.AllowConcurrent),
		SuccessfulJobsHistoryLimit: 3,
		FailedJobsHistoryLimit:     1,
		JobTemplate:                jobTemplatePolicy(item.Spec.JobTemplate.Spec),
	}
	if item.Spec.TimeZone != nil {
		detail.TimeZone = *item.Spec.TimeZone
	}
	if item.Spec.ConcurrencyPolicy != "" {
		detail.ConcurrencyPolicy = string(item.Spec.ConcurrencyPolicy)
	}
	if item.Spec.StartingDeadlineSeconds != nil {
		detail.StartingDeadlineSeconds = *item.Spec.StartingDeadlineSeconds
	}
	if item.Spec.SuccessfulJobsHistoryLimit != nil {
		detail.SuccessfulJobsHistoryLimit = *item.Spec.SuccessfulJobsHistoryLimit
	}
	if item.Spec.FailedJobsHistoryLimit != nil {
		detail.FailedJobsHistoryLimit = *item.Spec.FailedJobsHistoryLimit
	}
	if item.Status.LastSuccessfulTime != nil {
		detail.LastSuccessfulTime = item.Status.LastSuccessfulTime.Time
	}

	jobUIDs := make(map[string]string)
	for _, job := range jobs {
		if !hasControllerOwner(job.OwnerReferences, "CronJob", item.Name, string(item.UID)) {
			continue
		}
		detail.Jobs = append(detail.Jobs, jobDetail(job, pods, events))
		jobUIDs[job.Name] = string(job.UID)
	}
	sort.Slice(detail.Jobs, func(i, j int) bool {
		return detail.Jobs[i].CreatedAt.After(detail.Jobs[j].CreatedAt)
	})

	podUIDs := make(map[string]string)
	for _, pod := range pods {
		controller, ok := controllerOwnerReference(pod.OwnerReferences)
		if !ok || !strings.EqualFold(controller.Kind, "Job") {
			continue
		}
		jobUID, found := jobUIDs[controller.Name]
		if !found || (controller.UID != "" && jobUID != "" && string(controller.UID) != jobUID) {
			continue
		}
		podUIDs[pod.Name] = string(pod.UID)
	}

	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "cronjob":
			related = involvedObjectMatches(event.InvolvedObject, item.Name, string(item.UID))
		case "job":
			if uid, found := jobUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func jobTemplatePolicy(spec batchv1.JobSpec) JobTemplatePolicy {
	policy := JobTemplatePolicy{
		Parallelism:    1,
		Completions:    1,
		BackoffLimit:   6,
		CompletionMode: string(batchv1.NonIndexedCompletion),
		RestartPolicy:  string(spec.Template.Spec.RestartPolicy),
		Images:         podTemplateImages(spec.Template),
	}
	if spec.Parallelism != nil {
		policy.Parallelism = *spec.Parallelism
	}
	if spec.Completions != nil {
		policy.Completions = *spec.Completions
	}
	if spec.BackoffLimit != nil {
		policy.BackoffLimit = *spec.BackoffLimit
	}
	if spec.ActiveDeadlineSeconds != nil {
		policy.ActiveDeadlineSeconds = *spec.ActiveDeadlineSeconds
	}
	if spec.TTLSecondsAfterFinished != nil {
		policy.TTLSecondsAfterFinished = *spec.TTLSecondsAfterFinished
	}
	if spec.CompletionMode != nil {
		policy.CompletionMode = string(*spec.CompletionMode)
	}
	if spec.Suspend != nil {
		policy.Suspend = *spec.Suspend
	}
	return policy
}

func serviceDetail(
	item corev1.Service,
	slices []discoveryv1.EndpointSlice,
	legacy *corev1.Endpoints,
	pods []corev1.Pod,
	events []corev1.Event,
) ServiceDetail {
	detail := ServiceDetail{
		ServiceSummary:           serviceSummary(item),
		ClusterIPs:               append([]string(nil), item.Spec.ClusterIPs...),
		ExternalName:             item.Spec.ExternalName,
		SessionAffinity:          string(item.Spec.SessionAffinity),
		ExternalTrafficPolicy:    string(item.Spec.ExternalTrafficPolicy),
		LoadBalancerSourceRanges: append([]string(nil), item.Spec.LoadBalancerSourceRanges...),
		PortDetails:              servicePortDetails(item),
	}
	if len(detail.ClusterIPs) == 0 && item.Spec.ClusterIP != "" {
		detail.ClusterIPs = []string{item.Spec.ClusterIP}
	}
	for _, family := range item.Spec.IPFamilies {
		detail.IPFamilies = append(detail.IPFamilies, string(family))
	}
	if item.Spec.IPFamilyPolicy != nil {
		detail.IPFamilyPolicy = string(*item.Spec.IPFamilyPolicy)
	}
	if item.Spec.InternalTrafficPolicy != nil {
		detail.InternalTrafficPolicy = string(*item.Spec.InternalTrafficPolicy)
	}
	detail.PublishNotReadyAddresses = item.Spec.PublishNotReadyAddresses

	if len(slices) > 0 {
		detail.EndpointSource = "EndpointSlice"
		detail.Endpoints = endpointSliceDetails(slices)
	} else if legacy != nil {
		detail.EndpointSource = "Endpoints"
		detail.Endpoints = legacyEndpointDetails(legacy)
	}
	detail.Pods = servicePods(item, detail.Endpoints, pods)

	sliceUIDs := make(map[string]string, len(slices))
	for _, slice := range slices {
		sliceUIDs[slice.Name] = string(slice.UID)
	}
	podUIDs := make(map[string]string, len(detail.Pods))
	for _, pod := range detail.Pods {
		for _, item := range pods {
			if item.Namespace == pod.Namespace && item.Name == pod.Name {
				podUIDs[pod.Name] = string(item.UID)
				break
			}
		}
	}
	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "service":
			related = involvedObjectMatches(event.InvolvedObject, item.Name, string(item.UID))
		case "endpointslice":
			if len(slices) > 0 {
				if uid, found := sliceUIDs[event.InvolvedObject.Name]; found {
					related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
				}
			}
		case "endpoints":
			if legacy != nil {
				related = involvedObjectMatches(event.InvolvedObject, legacy.Name, string(legacy.UID))
			}
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func servicePortDetails(item corev1.Service) []ServicePortDetail {
	ports := make([]ServicePortDetail, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		detail := ServicePortDetail{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort.String(),
			NodePort:   port.NodePort,
		}
		if port.AppProtocol != nil {
			detail.AppProtocol = *port.AppProtocol
		}
		ports = append(ports, detail)
	}
	return ports
}

func endpointSliceDetails(items []discoveryv1.EndpointSlice) []ServiceEndpoint {
	result := make([]ServiceEndpoint, 0)
	for _, item := range items {
		ports := make([]string, 0, len(item.Ports))
		for _, port := range item.Ports {
			ports = append(ports, endpointSlicePort(port))
		}
		for _, endpoint := range item.Endpoints {
			ready := endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready
			serving := ready
			if endpoint.Conditions.Serving != nil {
				serving = *endpoint.Conditions.Serving
			}
			terminating := endpoint.Conditions.Terminating != nil && *endpoint.Conditions.Terminating
			detail := ServiceEndpoint{
				Source:      "EndpointSlice",
				SourceName:  item.Name,
				Addresses:   append([]string(nil), endpoint.Addresses...),
				Ready:       ready,
				Serving:     serving,
				Terminating: terminating,
				Ports:       append([]string(nil), ports...),
			}
			if endpoint.Hostname != nil {
				detail.Hostname = *endpoint.Hostname
			}
			if endpoint.NodeName != nil {
				detail.NodeName = *endpoint.NodeName
			}
			if endpoint.Zone != nil {
				detail.Zone = *endpoint.Zone
			}
			if endpoint.TargetRef != nil {
				detail.TargetKind = endpoint.TargetRef.Kind
				detail.TargetName = endpoint.TargetRef.Name
			}
			result = append(result, detail)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceName != result[j].SourceName {
			return result[i].SourceName < result[j].SourceName
		}
		left := strings.Join(result[i].Addresses, ",")
		right := strings.Join(result[j].Addresses, ",")
		if left != right {
			return left < right
		}
		return result[i].TargetName < result[j].TargetName
	})
	return result
}

func endpointSlicePort(port discoveryv1.EndpointPort) string {
	name := ""
	if port.Name != nil {
		name = *port.Name
	}
	portNumber := "*"
	if port.Port != nil {
		portNumber = fmt.Sprintf("%d", *port.Port)
	}
	protocol := string(corev1.ProtocolTCP)
	if port.Protocol != nil {
		protocol = string(*port.Protocol)
	}
	value := fmt.Sprintf("%s/%s", portNumber, protocol)
	if name != "" {
		value = name + ":" + value
	}
	return value
}

func legacyEndpointDetails(item *corev1.Endpoints) []ServiceEndpoint {
	if item == nil {
		return nil
	}
	result := make([]ServiceEndpoint, 0)
	for _, subset := range item.Subsets {
		ports := make([]string, 0, len(subset.Ports))
		for _, port := range subset.Ports {
			value := fmt.Sprintf("%d/%s", port.Port, port.Protocol)
			if port.Name != "" {
				value = port.Name + ":" + value
			}
			ports = append(ports, value)
		}
		for _, address := range subset.Addresses {
			result = append(result, legacyEndpoint(item.Name, address, ports, true))
		}
		for _, address := range subset.NotReadyAddresses {
			result = append(result, legacyEndpoint(item.Name, address, ports, false))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.Join(result[i].Addresses, ",")
		right := strings.Join(result[j].Addresses, ",")
		if left != right {
			return left < right
		}
		return result[i].TargetName < result[j].TargetName
	})
	return result
}

func legacyEndpoint(sourceName string, address corev1.EndpointAddress, ports []string, ready bool) ServiceEndpoint {
	detail := ServiceEndpoint{
		Source:     "Endpoints",
		SourceName: sourceName,
		Addresses:  []string{address.IP},
		Ready:      ready,
		Serving:    ready,
		Hostname:   address.Hostname,
		Ports:      append([]string(nil), ports...),
	}
	if address.NodeName != nil {
		detail.NodeName = *address.NodeName
	}
	if address.TargetRef != nil {
		detail.TargetKind = address.TargetRef.Kind
		detail.TargetName = address.TargetRef.Name
	}
	return detail
}

func servicePods(item corev1.Service, endpoints []ServiceEndpoint, pods []corev1.Pod) []PodSummary {
	selector := klabels.SelectorFromSet(item.Spec.Selector)
	hasSelector := len(item.Spec.Selector) > 0
	targetPods := make(map[string]struct{})
	for _, endpoint := range endpoints {
		if strings.EqualFold(endpoint.TargetKind, "Pod") && endpoint.TargetName != "" {
			targetPods[endpoint.TargetName] = struct{}{}
		}
	}

	selected := make(map[string]struct{})
	result := make([]PodSummary, 0)
	for _, pod := range pods {
		if pod.Namespace != item.Namespace {
			continue
		}
		_, targeted := targetPods[pod.Name]
		if !targeted && (!hasSelector || !selector.Matches(klabels.Set(pod.Labels))) {
			continue
		}
		key := string(pod.UID)
		if key == "" {
			key = pod.Namespace + "/" + pod.Name
		}
		if _, found := selected[key]; found {
			continue
		}
		selected[key] = struct{}{}
		result = append(result, podSummary(pod))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func serviceSummary(item corev1.Service) ServiceSummary {
	ports := make([]string, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		entry := fmt.Sprintf("%d/%s", port.Port, port.Protocol)
		if port.NodePort > 0 {
			entry = fmt.Sprintf("%s:%d", entry, port.NodePort)
		}
		if port.Name != "" {
			entry = port.Name + ":" + entry
		}
		ports = append(ports, entry)
	}
	externalIP := strings.Join(item.Spec.ExternalIPs, ",")
	if item.Spec.LoadBalancerIP != "" {
		if externalIP != "" {
			externalIP += ","
		}
		externalIP += item.Spec.LoadBalancerIP
	}
	return ServiceSummary{
		Namespace:  item.Namespace,
		Name:       item.Name,
		Type:       string(item.Spec.Type),
		ClusterIP:  item.Spec.ClusterIP,
		ExternalIP: externalIP,
		Ports:      ports,
		Selector:   item.Spec.Selector,
		CreatedAt:  item.CreationTimestamp.Time,
	}
}

func ingressSummary(item networkingv1.Ingress) IngressSummary {
	hosts := make([]string, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	addresses := make([]string, 0)
	for _, ingress := range item.Status.LoadBalancer.Ingress {
		switch {
		case ingress.IP != "":
			addresses = append(addresses, ingress.IP)
		case ingress.Hostname != "":
			addresses = append(addresses, ingress.Hostname)
		}
	}
	className := ""
	if item.Spec.IngressClassName != nil {
		className = *item.Spec.IngressClassName
	}
	return IngressSummary{
		Namespace: item.Namespace,
		Name:      item.Name,
		ClassName: className,
		Hosts:     hosts,
		Addresses: addresses,
		TLS:       len(item.Spec.TLS) > 0,
		CreatedAt: item.CreationTimestamp.Time,
	}
}

func ingressDetail(
	item networkingv1.Ingress,
	services []corev1.Service,
	slices []discoveryv1.EndpointSlice,
	endpoints []corev1.Endpoints,
	pods []corev1.Pod,
	events []corev1.Event,
) IngressDetail {
	detail := IngressDetail{IngressSummary: ingressSummary(item)}
	detail.Backends = ingressBackends(item)
	for _, tls := range item.Spec.TLS {
		detail.TLSDetails = append(detail.TLSDetails, IngressTLSDetail{
			Hosts:      append([]string(nil), tls.Hosts...),
			SecretName: tls.SecretName,
		})
	}

	serviceByName := make(map[string]corev1.Service, len(services))
	for _, service := range services {
		serviceByName[service.Name] = service
	}
	slicesByService := make(map[string][]discoveryv1.EndpointSlice)
	for _, slice := range slices {
		serviceName := slice.Labels[discoveryv1.LabelServiceName]
		if serviceName != "" {
			slicesByService[serviceName] = append(slicesByService[serviceName], slice)
		}
	}
	endpointsByName := make(map[string]corev1.Endpoints, len(endpoints))
	for _, endpoint := range endpoints {
		endpointsByName[endpoint.Name] = endpoint
	}

	serviceUIDs := make(map[string]string)
	sliceUIDs := make(map[string]string)
	endpointUIDs := make(map[string]string)
	podUIDs := make(map[string]string)
	seenServices := make(map[string]struct{})
	for index := range detail.Backends {
		backend := &detail.Backends[index]
		if backend.BackendKind != "Service" || backend.BackendName == "" {
			continue
		}
		service, found := serviceByName[backend.BackendName]
		backend.ServiceFound = found
		if !found {
			continue
		}
		backend.ServicePortFound = ingressServicePortFound(service, backend.BackendPort)
		serviceUIDs[service.Name] = string(service.UID)
		for _, slice := range slicesByService[service.Name] {
			sliceUIDs[slice.Name] = string(slice.UID)
		}
		legacy, hasLegacy := endpointsByName[service.Name]
		var legacyPointer *corev1.Endpoints
		if hasLegacy {
			legacyPointer = &legacy
			endpointUIDs[legacy.Name] = string(legacy.UID)
		}
		if _, seen := seenServices[service.Name]; seen {
			continue
		}
		seenServices[service.Name] = struct{}{}
		serviceDetail := serviceDetail(service, slicesByService[service.Name], legacyPointer, pods, events)
		detail.Services = append(detail.Services, serviceDetail)
		for _, associatedPod := range serviceDetail.Pods {
			for _, pod := range pods {
				if pod.Namespace == associatedPod.Namespace && pod.Name == associatedPod.Name {
					podUIDs[pod.Name] = string(pod.UID)
					break
				}
			}
		}
	}

	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "ingress":
			related = involvedObjectMatches(event.InvolvedObject, item.Name, string(item.UID))
		case "service":
			if uid, found := serviceUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		case "endpointslice":
			if uid, found := sliceUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		case "endpoints":
			if uid, found := endpointUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		case "pod":
			if uid, found := podUIDs[event.InvolvedObject.Name]; found {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			detail.Events = append(detail.Events, eventSummary(event))
		}
	}
	sort.Slice(detail.Events, func(i, j int) bool {
		return detail.Events[i].LastTimestamp.After(detail.Events[j].LastTimestamp)
	})
	return detail
}

func ingressBackends(item networkingv1.Ingress) []IngressBackendDetail {
	result := make([]IngressBackendDetail, 0)
	if item.Spec.DefaultBackend != nil {
		result = append(result, ingressBackend("", "", "", true, *item.Spec.DefaultBackend))
	}
	for _, rule := range item.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			pathType := ""
			if path.PathType != nil {
				pathType = string(*path.PathType)
			}
			result = append(result, ingressBackend(rule.Host, path.Path, pathType, false, path.Backend))
		}
	}
	return result
}

func ingressBackend(host, path, pathType string, isDefault bool, backend networkingv1.IngressBackend) IngressBackendDetail {
	detail := IngressBackendDetail{
		Host:      host,
		Path:      path,
		PathType:  pathType,
		IsDefault: isDefault,
	}
	if backend.Service != nil {
		detail.BackendKind = "Service"
		detail.BackendName = backend.Service.Name
		if backend.Service.Port.Name != "" {
			detail.BackendPort = backend.Service.Port.Name
		} else if backend.Service.Port.Number != 0 {
			detail.BackendPort = fmt.Sprintf("%d", backend.Service.Port.Number)
		}
		return detail
	}
	if backend.Resource != nil {
		detail.BackendKind = backend.Resource.Kind
		detail.BackendName = backend.Resource.Name
		if backend.Resource.APIGroup != nil {
			detail.BackendAPIGroup = *backend.Resource.APIGroup
		}
	}
	return detail
}

func ingressServicePortFound(service corev1.Service, backendPort string) bool {
	for _, port := range service.Spec.Ports {
		if port.Name != "" && port.Name == backendPort {
			return true
		}
		if fmt.Sprintf("%d", port.Port) == backendPort {
			return true
		}
	}
	return false
}

func configMapSummary(item corev1.ConfigMap) ConfigMapSummary {
	keys := make([]string, 0, len(item.Data)+len(item.BinaryData))
	for key := range item.Data {
		keys = append(keys, key)
	}
	for key := range item.BinaryData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return ConfigMapSummary{
		Namespace:       item.Namespace,
		Name:            item.Name,
		KeyCount:        len(item.Data),
		BinaryDataCount: len(item.BinaryData),
		Keys:            keys,
		CreatedAt:       item.CreationTimestamp.Time,
	}
}

type pvcAssociations struct {
	pods         []PodSummary
	workloads    []OwnerReference
	mounts       []VolumeMountDetail
	podUIDs      map[string]string
	workloadUIDs map[string]string
}

func (s *Service) pvcDetail(ctx context.Context, item corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, pods []corev1.Pod, events []corev1.Event) PVCDetail {
	detail := PVCDetail{
		PVCSummary: pvcSummary(item),
		VolumeMode: string(corev1.PersistentVolumeFilesystem),
		Selector:   map[string]string{},
	}
	if item.Spec.VolumeMode != nil {
		detail.VolumeMode = string(*item.Spec.VolumeMode)
	}
	if item.Spec.Selector != nil {
		for key, value := range item.Spec.Selector.MatchLabels {
			detail.Selector[key] = value
		}
		detail.SelectorExpressions = selectorExpressions(item.Spec.Selector.MatchExpressions)
	}
	detail.DataSource = pvcDataSource(item)
	for _, condition := range item.Status.Conditions {
		detail.Conditions = append(detail.Conditions, Condition{
			Type:    string(condition.Type),
			Status:  string(condition.Status),
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}
	if pv != nil {
		summary := pvSummary(*pv)
		detail.PV = &summary
	}

	associations := s.findPVCAssociations(ctx, item, pods)
	detail.Pods = associations.pods
	detail.Workloads = associations.workloads
	detail.Mounts = associations.mounts
	detail.Events = storageEvents(item, pv, associations, events)
	return detail
}

func (s *Service) pvDetail(ctx context.Context, item corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim, pods []corev1.Pod, events []corev1.Event) PVDetail {
	sourceType, sourceInfo := persistentVolumeSourceSummary(item.Spec.PersistentVolumeSource)
	detail := PVDetail{
		PVSummary:        pvSummary(item),
		VolumeMode:       string(corev1.PersistentVolumeFilesystem),
		MountOptions:     append([]string(nil), item.Spec.MountOptions...),
		NodeAffinity:     volumeNodeAffinity(item.Spec.NodeAffinity),
		VolumeSourceType: sourceType,
		VolumeSourceInfo: sourceInfo,
	}
	if item.Spec.VolumeMode != nil {
		detail.VolumeMode = string(*item.Spec.VolumeMode)
	}

	associations := pvcAssociations{
		podUIDs:      map[string]string{},
		workloadUIDs: map[string]string{},
	}
	if pvc != nil {
		summary := pvcSummary(*pvc)
		detail.PVC = &summary
		associations = s.findPVCAssociations(ctx, *pvc, pods)
		detail.Pods = associations.pods
		detail.Workloads = associations.workloads
		detail.Mounts = associations.mounts
	}
	detail.Events = storageEventsForPV(item, pvc, associations, events)
	return detail
}

func (s *Service) findPVCAssociations(ctx context.Context, pvc corev1.PersistentVolumeClaim, pods []corev1.Pod) pvcAssociations {
	result := pvcAssociations{
		podUIDs:      map[string]string{},
		workloadUIDs: map[string]string{},
	}
	workloadSeen := map[string]struct{}{}
	for _, pod := range pods {
		volumeNames := podPVCVolumeNames(pod, pvc.Name)
		if len(volumeNames) == 0 {
			continue
		}
		result.pods = append(result.pods, podSummary(pod))
		result.podUIDs[pod.Name] = string(pod.UID)
		result.mounts = append(result.mounts, podVolumeMounts(pod, volumeNames)...)

		chain := s.podControllerChain(ctx, pod)
		if len(chain) == 0 {
			continue
		}
		workload := chain[len(chain)-1]
		key := resourceReferenceKey(workload.Kind, workload.Name)
		if _, found := workloadSeen[key]; !found {
			result.workloads = append(result.workloads, workload)
			workloadSeen[key] = struct{}{}
		}
		if uid := topControllerUID(ctx, s.client, pod); uid != "" {
			result.workloadUIDs[key] = uid
		}
	}
	sort.Slice(result.pods, func(i, j int) bool {
		return result.pods[i].Name < result.pods[j].Name
	})
	sort.Slice(result.workloads, func(i, j int) bool {
		if result.workloads[i].Kind == result.workloads[j].Kind {
			return result.workloads[i].Name < result.workloads[j].Name
		}
		return result.workloads[i].Kind < result.workloads[j].Kind
	})
	sort.Slice(result.mounts, func(i, j int) bool {
		left := result.mounts[i]
		right := result.mounts[j]
		if left.PodName != right.PodName {
			return left.PodName < right.PodName
		}
		if left.ContainerType != right.ContainerType {
			return left.ContainerType < right.ContainerType
		}
		if left.ContainerName != right.ContainerName {
			return left.ContainerName < right.ContainerName
		}
		return left.MountPath+left.DevicePath < right.MountPath+right.DevicePath
	})
	return result
}

func topControllerUID(ctx context.Context, client KubernetesClient, pod corev1.Pod) string {
	controller, ok := controllerOwnerReference(pod.OwnerReferences)
	if !ok {
		return ""
	}
	uid := string(controller.UID)
	switch strings.ToLower(controller.Kind) {
	case "replicaset":
		item, err := client.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err == nil {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				uid = string(parent.UID)
			}
		}
	case "job":
		item, err := client.BatchV1().Jobs(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err == nil {
			if parent, found := controllerOwnerReference(item.OwnerReferences); found {
				uid = string(parent.UID)
			}
		}
	}
	return uid
}

func podPVCVolumeNames(pod corev1.Pod, claimName string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == claimName {
			result[volume.Name] = struct{}{}
		}
	}
	return result
}

func podVolumeMounts(pod corev1.Pod, volumeNames map[string]struct{}) []VolumeMountDetail {
	result := make([]VolumeMountDetail, 0)
	appendContainer := func(containerType, containerName string, mounts []corev1.VolumeMount, devices []corev1.VolumeDevice) {
		for _, mount := range mounts {
			if _, found := volumeNames[mount.Name]; !found {
				continue
			}
			result = append(result, VolumeMountDetail{
				PodNamespace:  pod.Namespace,
				PodName:       pod.Name,
				VolumeName:    mount.Name,
				ContainerType: containerType,
				ContainerName: containerName,
				MountPath:     mount.MountPath,
				SubPath:       mount.SubPath,
				ReadOnly:      mount.ReadOnly,
			})
		}
		for _, device := range devices {
			if _, found := volumeNames[device.Name]; !found {
				continue
			}
			result = append(result, VolumeMountDetail{
				PodNamespace:  pod.Namespace,
				PodName:       pod.Name,
				VolumeName:    device.Name,
				ContainerType: containerType,
				ContainerName: containerName,
				DevicePath:    device.DevicePath,
			})
		}
	}
	for _, container := range pod.Spec.InitContainers {
		appendContainer("InitContainer", container.Name, container.VolumeMounts, container.VolumeDevices)
	}
	for _, container := range pod.Spec.Containers {
		appendContainer("Container", container.Name, container.VolumeMounts, container.VolumeDevices)
	}
	for _, container := range pod.Spec.EphemeralContainers {
		appendContainer("EphemeralContainer", container.Name, container.VolumeMounts, container.VolumeDevices)
	}
	return result
}

func storageEvents(pvc corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, associations pvcAssociations, events []corev1.Event) []EventSummary {
	result := make([]EventSummary, 0)
	for _, event := range events {
		related := false
		switch strings.ToLower(event.InvolvedObject.Kind) {
		case "persistentvolumeclaim":
			related = event.InvolvedObject.Namespace == pvc.Namespace && involvedObjectMatches(event.InvolvedObject, pvc.Name, string(pvc.UID))
		case "persistentvolume":
			related = pv != nil && involvedObjectMatches(event.InvolvedObject, pv.Name, string(pv.UID))
		case "pod":
			if uid, found := associations.podUIDs[event.InvolvedObject.Name]; found && event.InvolvedObject.Namespace == pvc.Namespace {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		default:
			key := resourceReferenceKey(event.InvolvedObject.Kind, event.InvolvedObject.Name)
			if uid, found := associations.workloadUIDs[key]; found && event.InvolvedObject.Namespace == pvc.Namespace {
				related = involvedObjectMatches(event.InvolvedObject, event.InvolvedObject.Name, uid)
			}
		}
		if related {
			result = append(result, eventSummary(event))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTimestamp.After(result[j].LastTimestamp)
	})
	return result
}

func storageEventsForPV(pv corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim, associations pvcAssociations, events []corev1.Event) []EventSummary {
	if pvc != nil {
		return storageEvents(*pvc, &pv, associations, events)
	}
	result := make([]EventSummary, 0)
	for _, event := range events {
		if strings.EqualFold(event.InvolvedObject.Kind, "PersistentVolume") && involvedObjectMatches(event.InvolvedObject, pv.Name, string(pv.UID)) {
			result = append(result, eventSummary(event))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTimestamp.After(result[j].LastTimestamp)
	})
	return result
}

func persistentVolumeClaims(pv *corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim) bool {
	if pv == nil || pvc == nil || pv.Spec.ClaimRef == nil {
		return false
	}
	if pvc.Spec.VolumeName != "" && pvc.Spec.VolumeName != pv.Name {
		return false
	}
	claim := pv.Spec.ClaimRef
	if claim.Namespace != pvc.Namespace || claim.Name != pvc.Name {
		return false
	}
	return claim.UID == "" || pvc.UID == "" || claim.UID == pvc.UID
}

func pvcDataSource(item corev1.PersistentVolumeClaim) *VolumeDataSource {
	if item.Spec.DataSourceRef != nil {
		source := &VolumeDataSource{Kind: item.Spec.DataSourceRef.Kind, Name: item.Spec.DataSourceRef.Name}
		if item.Spec.DataSourceRef.APIGroup != nil {
			source.APIGroup = *item.Spec.DataSourceRef.APIGroup
		}
		if item.Spec.DataSourceRef.Namespace != nil {
			source.Namespace = *item.Spec.DataSourceRef.Namespace
		}
		return source
	}
	if item.Spec.DataSource != nil {
		source := &VolumeDataSource{Kind: item.Spec.DataSource.Kind, Name: item.Spec.DataSource.Name}
		if item.Spec.DataSource.APIGroup != nil {
			source.APIGroup = *item.Spec.DataSource.APIGroup
		}
		return source
	}
	return nil
}

func selectorExpressions(expressions []metav1.LabelSelectorRequirement) []string {
	result := make([]string, 0, len(expressions))
	for _, expression := range expressions {
		value := expression.Key + " " + string(expression.Operator)
		if len(expression.Values) > 0 {
			value += " (" + strings.Join(expression.Values, ", ") + ")"
		}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func resourceReferenceKey(kind, name string) string {
	return strings.ToLower(kind) + "/" + name
}

func volumeNodeAffinity(affinity *corev1.VolumeNodeAffinity) []string {
	if affinity == nil || affinity.Required == nil {
		return nil
	}
	result := make([]string, 0)
	for _, term := range affinity.Required.NodeSelectorTerms {
		for _, expression := range term.MatchExpressions {
			result = append(result, "label:"+expression.Key+" "+string(expression.Operator)+" ("+strings.Join(expression.Values, ", ")+")")
		}
		for _, expression := range term.MatchFields {
			result = append(result, "field:"+expression.Key+" "+string(expression.Operator)+" ("+strings.Join(expression.Values, ", ")+")")
		}
	}
	sort.Strings(result)
	return result
}

func persistentVolumeSourceSummary(source corev1.PersistentVolumeSource) (string, map[string]string) {
	info := map[string]string{}
	switch {
	case source.CSI != nil:
		info["driver"] = source.CSI.Driver
		info["fs_type"] = source.CSI.FSType
		info["read_only"] = fmt.Sprintf("%t", source.CSI.ReadOnly)
		return "CSI", info
	case source.Local != nil:
		info["path"] = source.Local.Path
		if source.Local.FSType != nil {
			info["fs_type"] = *source.Local.FSType
		}
		return "Local", info
	case source.HostPath != nil:
		info["path"] = source.HostPath.Path
		if source.HostPath.Type != nil {
			info["type"] = string(*source.HostPath.Type)
		}
		return "HostPath", info
	case source.NFS != nil:
		info["server"] = source.NFS.Server
		info["path"] = source.NFS.Path
		info["read_only"] = fmt.Sprintf("%t", source.NFS.ReadOnly)
		return "NFS", info
	case source.GCEPersistentDisk != nil:
		info["fs_type"] = source.GCEPersistentDisk.FSType
		info["partition"] = fmt.Sprintf("%d", source.GCEPersistentDisk.Partition)
		info["read_only"] = fmt.Sprintf("%t", source.GCEPersistentDisk.ReadOnly)
		return "GCEPersistentDisk", info
	case source.AWSElasticBlockStore != nil:
		info["fs_type"] = source.AWSElasticBlockStore.FSType
		info["partition"] = fmt.Sprintf("%d", source.AWSElasticBlockStore.Partition)
		info["read_only"] = fmt.Sprintf("%t", source.AWSElasticBlockStore.ReadOnly)
		return "AWSElasticBlockStore", info
	case source.RBD != nil:
		return "RBD", info
	case source.ISCSI != nil:
		return "ISCSI", info
	case source.Cinder != nil:
		return "Cinder", info
	case source.CephFS != nil:
		return "CephFS", info
	case source.FC != nil:
		return "FibreChannel", info
	case source.Flocker != nil:
		return "Flocker", info
	case source.FlexVolume != nil:
		info["driver"] = source.FlexVolume.Driver
		return "FlexVolume", info
	case source.AzureFile != nil:
		return "AzureFile", info
	case source.VsphereVolume != nil:
		return "VsphereVolume", info
	case source.Quobyte != nil:
		return "Quobyte", info
	case source.AzureDisk != nil:
		return "AzureDisk", info
	case source.PhotonPersistentDisk != nil:
		return "PhotonPersistentDisk", info
	case source.PortworxVolume != nil:
		return "PortworxVolume", info
	case source.ScaleIO != nil:
		return "ScaleIO", info
	case source.StorageOS != nil:
		return "StorageOS", info
	default:
		return "Unknown", info
	}
}

func pvcSummary(item corev1.PersistentVolumeClaim) PVCSummary {
	storageClass := ""
	if item.Spec.StorageClassName != nil {
		storageClass = *item.Spec.StorageClassName
	}
	return PVCSummary{
		Namespace:    item.Namespace,
		Name:         item.Name,
		Status:       string(item.Status.Phase),
		StorageClass: storageClass,
		VolumeName:   item.Spec.VolumeName,
		Requested:    quantityString(item.Spec.Resources.Requests, corev1.ResourceStorage),
		Capacity:     quantityString(item.Status.Capacity, corev1.ResourceStorage),
		AccessModes:  accessModes(item.Spec.AccessModes),
		CreatedAt:    item.CreationTimestamp.Time,
	}
}

func pvSummary(item corev1.PersistentVolume) PVSummary {
	claimNamespace := ""
	claimName := ""
	if item.Spec.ClaimRef != nil {
		claimNamespace = item.Spec.ClaimRef.Namespace
		claimName = item.Spec.ClaimRef.Name
	}
	return PVSummary{
		Name:           item.Name,
		Status:         string(item.Status.Phase),
		StorageClass:   item.Spec.StorageClassName,
		Capacity:       quantityString(item.Spec.Capacity, corev1.ResourceStorage),
		ClaimNamespace: claimNamespace,
		ClaimName:      claimName,
		ReclaimPolicy:  string(item.Spec.PersistentVolumeReclaimPolicy),
		AccessModes:    accessModes(item.Spec.AccessModes),
		CreatedAt:      item.CreationTimestamp.Time,
	}
}

func storageClassSummary(item storagev1.StorageClass) StorageClassSummary {
	reclaimPolicy := ""
	if item.ReclaimPolicy != nil {
		reclaimPolicy = string(*item.ReclaimPolicy)
	}
	volumeBindingMode := ""
	if item.VolumeBindingMode != nil {
		volumeBindingMode = string(*item.VolumeBindingMode)
	}
	allowExpansion := false
	if item.AllowVolumeExpansion != nil {
		allowExpansion = *item.AllowVolumeExpansion
	}
	return StorageClassSummary{
		Name:                 item.Name,
		Provisioner:          item.Provisioner,
		ReclaimPolicy:        reclaimPolicy,
		VolumeBindingMode:    volumeBindingMode,
		AllowVolumeExpansion: allowExpansion,
		CreatedAt:            item.CreationTimestamp.Time,
	}
}

func podTemplateImages(template corev1.PodTemplateSpec) []string {
	images := make([]string, 0, len(template.Spec.Containers))
	for _, container := range template.Spec.Containers {
		images = append(images, container.Image)
	}
	return images
}

func sortWorkloads(items []WorkloadSummary) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Namespace == items[j].Namespace {
			return items[i].Name < items[j].Name
		}
		return items[i].Namespace < items[j].Namespace
	})
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	result := make([]string, 0, len(modes))
	for _, mode := range modes {
		result = append(result, string(mode))
	}
	return result
}

func quantityString(list corev1.ResourceList, name corev1.ResourceName) string {
	value, ok := list[name]
	if !ok {
		return ""
	}
	return value.String()
}
