package resources

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
	return podDetail(*pod), nil
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

func (s *Service) ListEvents(ctx context.Context, namespace string) ([]EventSummary, error) {
	if s.client == nil {
		return nil, unavailable()
	}
	items, err := s.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, mapKubernetesError(err, "list events failed")
	}
	events := make([]EventSummary, 0, len(items.Items))
	for _, event := range items.Items {
		events = append(events, eventSummary(event))
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp)
	})
	return events, nil
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

func (s *Service) GetDeployment(ctx context.Context, namespace, name string) (DeploymentSummary, error) {
	if s.client == nil {
		return DeploymentSummary{}, unavailable()
	}
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return DeploymentSummary{}, mapKubernetesError(err, "get deployment failed")
	}
	return DeploymentSummaryFrom(*deployment), nil
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
		workloads = append(workloads, statefulSetSummary(item))
	}
	sortWorkloads(workloads)
	return paginate(workloads, opts), nil
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
		daemonSets = append(daemonSets, daemonSetSummary(item))
	}
	sort.Slice(daemonSets, func(i, j int) bool {
		if daemonSets[i].Namespace == daemonSets[j].Namespace {
			return daemonSets[i].Name < daemonSets[j].Name
		}
		return daemonSets[i].Namespace < daemonSets[j].Namespace
	})
	return paginate(daemonSets, opts), nil
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
		cronJobs = append(cronJobs, cronJobSummary(item))
	}
	sort.Slice(cronJobs, func(i, j int) bool {
		if cronJobs[i].Namespace == cronJobs[j].Namespace {
			return cronJobs[i].Name < cronJobs[j].Name
		}
		return cronJobs[i].Namespace < cronJobs[j].Namespace
	})
	return paginate(cronJobs, opts), nil
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

func podDetail(pod corev1.Pod) PodDetail {
	summary := podSummary(pod)
	owners := make([]OwnerReference, 0, len(pod.OwnerReferences))
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, OwnerReference{Kind: owner.Kind, Name: owner.Name})
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
	return PodDetail{
		PodSummary:  summary,
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		OwnerRefs:   owners,
		Conditions:  conditions,
	}
}

func containerSummary(status corev1.ContainerStatus) ContainerSummary {
	state := "Unknown"
	reason := ""
	switch {
	case status.State.Running != nil:
		state = "Running"
	case status.State.Waiting != nil:
		state = "Waiting"
		reason = status.State.Waiting.Reason
	case status.State.Terminated != nil:
		state = "Terminated"
		reason = status.State.Terminated.Reason
	}
	return ContainerSummary{
		Name:         status.Name,
		Image:        status.Image,
		Ready:        status.Ready,
		RestartCount: status.RestartCount,
		State:        state,
		Reason:       reason,
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
	for _, status := range pod.Status.ContainerStatuses {
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
	return EventSummary{
		Type:           event.Type,
		Reason:         event.Reason,
		Message:        event.Message,
		Count:          event.Count,
		Namespace:      event.Namespace,
		InvolvedKind:   event.InvolvedObject.Kind,
		InvolvedName:   event.InvolvedObject.Name,
		Source:         source,
		FirstTimestamp: event.FirstTimestamp.Time,
		LastTimestamp:  event.LastTimestamp.Time,
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

func statefulSetSummary(item appsv1.StatefulSet) WorkloadSummary {
	replicas := int32(0)
	if item.Spec.Replicas != nil {
		replicas = *item.Spec.Replicas
	}
	return WorkloadSummary{
		Kind:                "StatefulSet",
		Namespace:           item.Namespace,
		Name:                item.Name,
		Replicas:            replicas,
		ReadyReplicas:       item.Status.ReadyReplicas,
		AvailableReplicas:   item.Status.AvailableReplicas,
		UnavailableReplicas: replicas - item.Status.ReadyReplicas,
		Labels:              item.Labels,
		Images:              podTemplateImages(item.Spec.Template),
		CreatedAt:           item.CreationTimestamp.Time,
	}
}

func replicaSetSummary(item appsv1.ReplicaSet) WorkloadSummary {
	replicas := int32(0)
	if item.Spec.Replicas != nil {
		replicas = *item.Spec.Replicas
	}
	return WorkloadSummary{
		Kind:                "ReplicaSet",
		Namespace:           item.Namespace,
		Name:                item.Name,
		Replicas:            replicas,
		ReadyReplicas:       item.Status.ReadyReplicas,
		AvailableReplicas:   item.Status.AvailableReplicas,
		UnavailableReplicas: replicas - item.Status.ReadyReplicas,
		Labels:              item.Labels,
		Images:              podTemplateImages(item.Spec.Template),
		CreatedAt:           item.CreationTimestamp.Time,
	}
}

func daemonSetSummary(item appsv1.DaemonSet) DaemonSetSummary {
	return DaemonSetSummary{
		Namespace:       item.Namespace,
		Name:            item.Name,
		DesiredNumber:   item.Status.DesiredNumberScheduled,
		CurrentNumber:   item.Status.CurrentNumberScheduled,
		ReadyNumber:     item.Status.NumberReady,
		AvailableNumber: item.Status.NumberAvailable,
		Labels:          item.Labels,
		Images:          podTemplateImages(item.Spec.Template),
		CreatedAt:       item.CreationTimestamp.Time,
	}
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

func cronJobSummary(item batchv1.CronJob) CronJobSummary {
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
