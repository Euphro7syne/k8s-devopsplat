package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"

	apperrors "ops-platform/internal/pkg/errors"
)

func TestSecretSafeReadDoesNotLeakValues(t *testing.T) {
	immutable := true
	client := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-secret",
				Namespace: "demo",
				Labels:    map[string]string{"app": "api"},
			},
			Type:      corev1.SecretTypeOpaque,
			Immutable: &immutable,
			Data: map[string][]byte{
				"password": []byte("value-that-must-not-leak"),
				"token":    []byte("second-value-that-must-not-leak"),
			},
		},
	)
	service := NewService(client, "test")

	list, err := service.ListSecrets(context.Background(), "demo", ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].KeyCount != 2 {
		t.Fatalf("unexpected Secret list: %#v", list)
	}
	if strings.Join(list.Items[0].Keys, ",") != "password,token" {
		t.Fatalf("expected sorted Secret keys, got %#v", list.Items[0].Keys)
	}

	detail, err := service.GetSecret(context.Background(), "demo", "app-secret")
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if !detail.Immutable || detail.Labels["app"] != "api" || len(detail.KeyDetails) != 2 {
		t.Fatalf("unexpected Secret detail: %#v", detail)
	}
	if detail.KeyDetails[0].Name != "password" || detail.KeyDetails[0].SizeBytes != int64(len("value-that-must-not-leak")) {
		t.Fatalf("unexpected Secret key metadata: %#v", detail.KeyDetails)
	}

	for name, value := range map[string]any{"list": list, "detail": detail} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s response: %v", name, err)
		}
		if strings.Contains(string(raw), "value-that-must-not-leak") || strings.Contains(string(raw), "second-value-that-must-not-leak") {
			t.Fatalf("Secret value leaked in %s response: %s", name, raw)
		}
	}
}

func TestReadSecretValueReturnsOnlyRequestedKey(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "demo"},
		Data: map[string][]byte{
			"password": []byte("requested-value"),
			"token":    []byte("must-not-be-returned"),
		},
	})
	service := NewService(client, "test")

	result, err := service.ReadSecretValue(context.Background(), "demo", "app-secret", "password")
	if err != nil {
		t.Fatalf("read Secret value: %v", err)
	}
	if result.Value != "requested-value" || result.Encoding != "utf-8" || result.Key != "password" {
		t.Fatalf("unexpected Secret value response: %#v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal Secret value response: %v", err)
	}
	if strings.Contains(string(raw), "must-not-be-returned") {
		t.Fatalf("unrequested Secret value leaked: %s", raw)
	}
}

func TestReadSecretValueUsesBase64ForBinaryData(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "binary-secret", Namespace: "demo"},
		Data:       map[string][]byte{"payload": {0xff, 0x00, 0x01}},
	})
	service := NewService(client, "test")

	result, err := service.ReadSecretValue(context.Background(), "demo", "binary-secret", "payload")
	if err != nil {
		t.Fatalf("read binary Secret value: %v", err)
	}
	if result.Encoding != "base64" || result.Value != "/wAB" {
		t.Fatalf("unexpected binary Secret value response: %#v", result)
	}
}

func TestReadSecretValueRejectsMissingKey(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "demo"},
		Data:       map[string][]byte{"password": []byte("value")},
	})
	service := NewService(client, "test")

	_, err := service.ReadSecretValue(context.Background(), "demo", "app-secret", "missing")
	if err == nil {
		t.Fatal("expected missing Secret key to return an error")
	}
	appErr := apperrors.From(err)
	if appErr.Code != apperrors.CodeNotFound || appErr.HTTPStatus != http.StatusNotFound {
		t.Fatalf("unexpected missing Secret key error: %#v", appErr)
	}
}

func TestResourceYAMLRejectsSecret(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "demo"},
		Data:       map[string][]byte{"password": []byte("must-not-leak")},
	})
	service := NewService(client, "test")

	result, err := service.ResourceYAML(context.Background(), "secret", "demo", "app-secret")
	if err == nil || result != "" {
		t.Fatalf("expected generic Secret YAML export to remain disabled: result=%q err=%v", result, err)
	}
}

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

func TestGetNamespaceDetailIncludesCountsAllocationAssociationsAndEvents(t *testing.T) {
	controller := true
	namespaceUID := k8stypes.UID("namespace-demo")
	replicaSetUID := k8stypes.UID("replicaset-api")
	deploymentUID := k8stypes.UID("deployment-api")
	client := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "demo",
				UID:    namespaceUID,
				Labels: map[string]string{"environment": "test"},
			},
			Spec: corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes}},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
				Conditions: []corev1.NamespaceCondition{{
					Type: corev1.NamespaceContentRemaining, Status: corev1.ConditionFalse, Reason: "ContentRemoved", Message: "all content removed",
				}},
			},
		},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "demo", UID: deploymentUID}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "demo"}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: "demo"}},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-v1", Namespace: "demo", UID: replicaSetUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: deploymentUID, Controller: &controller}},
			},
		},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "migrate", Namespace: "demo"}},
		&batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "demo"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "demo"}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "demo"}},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "demo"},
			Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
			}},
		},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "api-config", Namespace: "demo"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-1", Namespace: "demo",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-v1", UID: replicaSetUID, Controller: &controller}},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "api", Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("128Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("1Gi")},
						Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("512Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("3Gi")},
					}},
					{Name: "sidecar", Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("300m"), corev1.ResourceMemory: resource.MustParse("128Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("2Gi")},
						Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("256Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("2Gi")},
					}},
				},
				InitContainers: []corev1.Container{
					{Name: "init-a", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("700m"), corev1.ResourceMemory: resource.MustParse("64Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("4Gi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("1Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("4Gi")}}},
				},
				Overhead: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("32Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("1Gi")},
			},
			Status: corev1.PodStatus{
				Phase:      corev1.PodRunning,
				Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-2", Namespace: "demo",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-v1", UID: replicaSetUID, Controller: &controller}},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("64Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("1Gi")}}}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "api", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-completed", Namespace: "demo",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-v1", UID: replicaSetUID, Controller: &controller}},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("20"), corev1.ResourceMemory: resource.MustParse("20Gi")}}}}},
			Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "other"}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "namespace-event"}, InvolvedObject: corev1.ObjectReference{Kind: "Namespace", Name: "demo", UID: namespaceUID}, Reason: "Updated"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stale-namespace-event"}, InvolvedObject: corev1.ObjectReference{Kind: "Namespace", Name: "demo", UID: "old-namespace"}, Reason: "Stale"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pod-event", Namespace: "demo"}, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-1"}, Reason: "Started"},
	)
	service := NewService(client, "test")

	detail, err := service.GetNamespace(context.Background(), "demo")
	if err != nil {
		t.Fatalf("get namespace detail: %v", err)
	}
	if detail.Status != "Active" || detail.Labels["environment"] != "test" || len(detail.Finalizers) != 1 || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected namespace metadata: %#v", detail)
	}
	if detail.Counts.Pods != 3 || detail.Counts.ReadyPods != 1 || detail.Counts.AbnormalPods != 1 || detail.Counts.Deployments != 1 || detail.Counts.StatefulSets != 1 || detail.Counts.DaemonSets != 1 || detail.Counts.ReplicaSets != 1 || detail.Counts.Jobs != 1 || detail.Counts.CronJobs != 1 || detail.Counts.Services != 1 || detail.Counts.Ingresses != 1 || detail.Counts.PersistentVolumeClaims != 1 || detail.Counts.ConfigMaps != 1 {
		t.Fatalf("unexpected namespace counts: %#v", detail.Counts)
	}
	if detail.Allocated.Requests.CPU != "1" || detail.Allocated.Requests.Memory != "352Mi" || detail.Allocated.Requests.EphemeralStorage != "6Gi" || detail.Allocated.Requests.Pods != 2 {
		t.Fatalf("unexpected namespace requests: %#v", detail.Allocated.Requests)
	}
	if detail.Allocated.Limits.CPU != "2050m" || detail.Allocated.Limits.Memory != "1056Mi" || detail.Allocated.Limits.EphemeralStorage != "6Gi" {
		t.Fatalf("unexpected namespace limits: %#v", detail.Allocated.Limits)
	}
	if len(detail.Workloads) != 1 || detail.Workloads[0].Kind != "Deployment" || detail.Workloads[0].Namespace != "demo" || detail.Workloads[0].Name != "api" {
		t.Fatalf("unexpected namespace workloads: %#v", detail.Workloads)
	}
	if len(detail.Services) != 1 || len(detail.Ingresses) != 1 || len(detail.PersistentVolumeClaims) != 1 {
		t.Fatalf("unexpected namespace associations: services=%#v ingresses=%#v pvcs=%#v", detail.Services, detail.Ingresses, detail.PersistentVolumeClaims)
	}
	if len(detail.Events) != 1 || detail.Events[0].Reason != "Updated" {
		t.Fatalf("unexpected namespace events: %#v", detail.Events)
	}

	namespaceYAML, err := service.ResourceYAML(context.Background(), "namespace", "", "demo")
	if err != nil || !strings.Contains(namespaceYAML, "kind: Namespace") || strings.Contains(namespaceYAML, "managedFields") {
		t.Fatalf("unexpected namespace yaml: err=%v yaml=%q", err, namespaceYAML)
	}
}

func TestGetNodeDetailIncludesDeclaredAllocationPodsWorkloadsAndEvents(t *testing.T) {
	controller := true
	nodeUID := k8stypes.UID("node-a")
	statefulSetUID := k8stypes.UID("stateful-api")
	taintTime := metav1.NewTime(time.Now().Add(-time.Hour))
	client := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-a", UID: nodeUID, Labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "", "node-role.kubernetes.io/master": "", "node-role.kubernetes.io/worker": "", "node-role.kubernetes.io/edge": "",
			}},
			Spec: corev1.NodeSpec{
				PodCIDRs:      []string{"10.42.0.0/24"},
				Unschedulable: true,
				Taints:        []corev1.Taint{{Key: "dedicated", Value: "ops", Effect: corev1.TaintEffectNoSchedule, TimeAdded: &taintTime}},
			},
			Status: corev1.NodeStatus{
				Capacity:    corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("8"), corev1.ResourceMemory: resource.MustParse("16Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("200Gi"), corev1.ResourcePods: resource.MustParse("110")},
				Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("8Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("100Gi"), corev1.ResourcePods: resource.MustParse("100")},
				Addresses:   []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "10.0.0.10"}, {Type: corev1.NodeHostName, Address: "node-a"}},
				Conditions:  []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue, Reason: "KubeletReady", Message: "ready"}},
				NodeInfo: corev1.NodeSystemInfo{
					MachineID: "machine-a", SystemUUID: "system-a", BootID: "boot-a", KernelVersion: "6.1", OSImage: "Linux", ContainerRuntimeVersion: "containerd://1.7", KubeletVersion: "v1.30.3", KubeProxyVersion: "v1.30.3", OperatingSystem: "linux", Architecture: "amd64",
				},
			},
		},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "demo", UID: statefulSetUID}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: "demo", OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", Name: "api", UID: statefulSetUID, Controller: &controller}}},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{Name: "api", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("1Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("1Gi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("3Gi")}}},
					{Name: "sidecar", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("512Mi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("1Gi")}}},
				},
				InitContainers: []corev1.Container{{Name: "init", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("2Gi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi"), corev1.ResourceEphemeralStorage: resource.MustParse("4Gi")}}}},
				Overhead:       corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi"), corev1.ResourceEphemeralStorage: resource.MustParse("1Gi")},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "completed", Namespace: "demo"}, Spec: corev1.PodSpec{NodeName: "node-a", Containers: []corev1.Container{{Name: "done", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10"), corev1.ResourceMemory: resource.MustParse("10Gi")}}}}}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other-node", Namespace: "demo"}, Spec: corev1.PodSpec{NodeName: "node-b"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "node-event"}, InvolvedObject: corev1.ObjectReference{Kind: "Node", Name: "node-a", UID: nodeUID}, Reason: "Ready"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stale-node-event"}, InvolvedObject: corev1.ObjectReference{Kind: "Node", Name: "node-a", UID: "old-node"}, Reason: "Stale"},
	)
	service := NewService(client, "test")

	detail, err := service.GetNode(context.Background(), "node-a")
	if err != nil {
		t.Fatalf("get node detail: %v", err)
	}
	if detail.Status != "Ready" || !detail.Unschedulable || strings.Join(detail.Roles, ",") != "control-plane,edge,master,worker" || len(detail.Addresses) != 2 || len(detail.Taints) != 1 || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected node metadata: %#v", detail)
	}
	if detail.SystemInfo.KubeletVersion != "v1.30.3" || detail.Capacity.CPU != "8" || detail.Allocatable.Memory != "8Gi" || detail.Allocatable.Pods != 100 {
		t.Fatalf("unexpected node system/capacity: %#v", detail)
	}
	if detail.Allocated.Requests.CPU != "1100m" || detail.Allocated.Requests.Memory != "2304Mi" || detail.Allocated.Requests.EphemeralStorage != "3Gi" || detail.Allocated.Requests.Pods != 1 {
		t.Fatalf("unexpected node requests: %#v", detail.Allocated.Requests)
	}
	if detail.Allocated.Limits.CPU != "2100m" || detail.Allocated.Limits.Memory != "4352Mi" || detail.Allocated.Limits.EphemeralStorage != "5Gi" {
		t.Fatalf("unexpected node limits: %#v", detail.Allocated.Limits)
	}
	if detail.Allocated.CPURequestPercent != 27.5 || detail.Allocated.MemoryRequestPercent != 28.13 || detail.Allocated.PodPercent != 1 {
		t.Fatalf("unexpected node allocation percentages: %#v", detail.Allocated)
	}
	if len(detail.Pods) != 2 || detail.Pods[0].Name != "api-0" || detail.Pods[1].Name != "completed" {
		t.Fatalf("unexpected node pods: %#v", detail.Pods)
	}
	if len(detail.Workloads) != 1 || detail.Workloads[0].Kind != "StatefulSet" || detail.Workloads[0].Namespace != "demo" || detail.Workloads[0].Name != "api" {
		t.Fatalf("unexpected node workloads: %#v", detail.Workloads)
	}
	if len(detail.Events) != 1 || detail.Events[0].Reason != "Ready" {
		t.Fatalf("unexpected node events: %#v", detail.Events)
	}
}

func TestGetPodDiagnosticDetailAndFilteredEvents(t *testing.T) {
	controller := true
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-7d9f",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "api", Controller: &controller},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-7d9f-abcde",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "api-7d9f", Controller: &controller},
				},
			},
			Spec: corev1.PodSpec{
				NodeName:           "node-a",
				ServiceAccountName: "api",
				RestartPolicy:      corev1.RestartPolicyAlways,
			},
			Status: corev1.PodStatus{
				Phase:    corev1.PodRunning,
				PodIP:    "10.42.0.8",
				HostIP:   "10.0.0.10",
				QOSClass: corev1.PodQOSBurstable,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "api",
						Image:        "example/api:v1",
						Ready:        false,
						RestartCount: 4,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
						},
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137},
						},
					},
				},
			},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "pod-warning", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-7d9f-abcde", Namespace: "default"},
			Type:           "Warning", Reason: "BackOff", Message: "Back-off restarting failed container", Count: 3,
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "service-event", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Service", Name: "api", Namespace: "default"},
			Type:           "Normal", Reason: "Created", Message: "service created", Count: 1,
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetPod(context.Background(), "default", "api-7d9f-abcde")
	if err != nil {
		t.Fatalf("get pod detail: %v", err)
	}
	if detail.PodIP != "10.42.0.8" || detail.HostIP != "10.0.0.10" || detail.ServiceAccount != "api" {
		t.Fatalf("unexpected pod runtime detail: %#v", detail)
	}
	if len(detail.ControllerChain) != 2 || detail.ControllerChain[0].Kind != "ReplicaSet" || detail.ControllerChain[1].Kind != "Deployment" {
		t.Fatalf("unexpected controller chain: %#v", detail.ControllerChain)
	}
	if len(detail.Containers) != 1 || detail.Containers[0].LastReason != "OOMKilled" || detail.Containers[0].LastExitCode != 137 {
		t.Fatalf("unexpected container diagnostic detail: %#v", detail.Containers)
	}

	events, err := service.ListEvents(context.Background(), "default", "Pod", "api-7d9f-abcde")
	if err != nil {
		t.Fatalf("list filtered events: %v", err)
	}
	if len(events) != 1 || events[0].Reason != "BackOff" {
		t.Fatalf("unexpected filtered events: %#v", events)
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

func TestGetDeploymentDetailIncludesOwnedReplicaSetsPodsAndEvents(t *testing.T) {
	controller := true
	replicas := int32(2)
	maxSurge := intstr.FromString("25%")
	maxUnavailable := intstr.FromInt32(1)
	progressDeadline := int32(600)
	revisionHistory := int32(10)
	deploymentUID := k8stypes.UID("deployment-api")
	replicaSetUID := k8stypes.UID("replicaset-api-v2")
	podUID := k8stypes.UID("pod-api-v2")
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default", UID: deploymentUID, Generation: 4},
			Spec: appsv1.DeploymentSpec{
				Replicas:                &replicas,
				Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				ProgressDeadlineSeconds: &progressDeadline,
				RevisionHistoryLimit:    &revisionHistory,
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{
						MaxSurge:       &maxSurge,
						MaxUnavailable: &maxUnavailable,
					},
				},
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api:v2"}}}},
			},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 4,
				ReadyReplicas:      1,
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable", Message: "rollout progressing"},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-v2", Namespace: "default", UID: replicaSetUID,
				Annotations:     map[string]string{"deployment.kubernetes.io/revision": "2"},
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: deploymentUID, Controller: &controller}},
			},
			Spec:   appsv1.ReplicaSetSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api:v2"}}}}},
			Status: appsv1.ReplicaSetStatus{Replicas: 2, ReadyReplicas: 1, AvailableReplicas: 1, FullyLabeledReplicas: 2},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "unrelated", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "other", Controller: &controller}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-v2-abcde", Namespace: "default", UID: podUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-v2", UID: replicaSetUID, Controller: &controller}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "unrelated-abcde", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "unrelated", Controller: &controller}},
			},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "deployment-progress", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Deployment", Name: "api", UID: deploymentUID},
			Type:           "Normal", Reason: "ScalingReplicaSet", Message: "scaled api-v2",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "pod-backoff", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-v2-abcde", UID: podUID},
			Type:           "Warning", Reason: "BackOff", Message: "back-off restarting container",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "stale-pod-event", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-v2-abcde", UID: k8stypes.UID("old-pod-api-v2")},
			Type:           "Warning", Reason: "Failed", Message: "stale event for an old Pod UID",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "unrelated-event", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "unrelated-abcde"},
			Type:           "Warning", Reason: "Failed", Message: "unrelated",
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetDeployment(context.Background(), "default", "api")
	if err != nil {
		t.Fatalf("get deployment detail: %v", err)
	}
	if detail.Generation != 4 || detail.ObservedGeneration != 4 || detail.Strategy != "RollingUpdate" {
		t.Fatalf("unexpected deployment rollout detail: %#v", detail)
	}
	if detail.MaxSurge != "25%" || detail.MaxUnavailable != "1" || detail.Selector["app"] != "api" {
		t.Fatalf("unexpected deployment strategy detail: %#v", detail)
	}
	if len(detail.ReplicaSets) != 1 || detail.ReplicaSets[0].Name != "api-v2" || detail.ReplicaSets[0].Revision != "2" {
		t.Fatalf("unexpected owned replicasets: %#v", detail.ReplicaSets)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "api-v2-abcde" {
		t.Fatalf("unexpected owned pods: %#v", detail.Pods)
	}
	if len(detail.Events) != 2 || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected deployment diagnostic context: events=%#v conditions=%#v", detail.Events, detail.Conditions)
	}
}

func TestGetReplicaSetDetailIncludesOwnerPodsAndEvents(t *testing.T) {
	controller := true
	replicas := int32(2)
	deploymentUID := k8stypes.UID("deployment-api")
	replicaSetUID := k8stypes.UID("replicaset-api-v3")
	podUID := k8stypes.UID("pod-api-v3")
	client := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-v3", Namespace: "default", UID: replicaSetUID,
				Annotations:     map[string]string{"deployment.kubernetes.io/revision": "3"},
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: deploymentUID, Controller: &controller}},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas:        &replicas,
				MinReadySeconds: 5,
				Selector:        &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api", "pod-template-hash": "v3"}},
				Template:        corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "example/api:v3"}}}},
			},
			Status: appsv1.ReplicaSetStatus{
				ObservedGeneration:   2,
				Replicas:             2,
				ReadyReplicas:        1,
				AvailableReplicas:    1,
				FullyLabeledReplicas: 2,
				Conditions: []appsv1.ReplicaSetCondition{
					{Type: appsv1.ReplicaSetReplicaFailure, Status: corev1.ConditionTrue, Reason: "FailedCreate", Message: "quota exceeded"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-v3-abcde", Namespace: "default", UID: podUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-v3", UID: replicaSetUID, Controller: &controller}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodPending},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "unrelated-abcde", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "other", Controller: &controller}},
			},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "replicaset-failed", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "ReplicaSet", Name: "api-v3", UID: replicaSetUID},
			Type:           "Warning", Reason: "FailedCreate", Message: "quota exceeded",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "pod-failed", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-v3-abcde", UID: podUID},
			Type:           "Warning", Reason: "FailedScheduling", Message: "insufficient cpu",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "stale-pod", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-v3-abcde", UID: k8stypes.UID("old-pod")},
			Type:           "Warning", Reason: "BackOff", Message: "stale",
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetReplicaSet(context.Background(), "default", "api-v3")
	if err != nil {
		t.Fatalf("get replicaset detail: %v", err)
	}
	if detail.Revision != "3" || detail.ObservedGeneration != 2 || detail.MinReadySeconds != 5 {
		t.Fatalf("unexpected replicaset rollout detail: %#v", detail)
	}
	if detail.Owner == nil || detail.Owner.Kind != "Deployment" || detail.Owner.Name != "api" {
		t.Fatalf("unexpected replicaset owner: %#v", detail.Owner)
	}
	if detail.Selector["app"] != "api" || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected replicaset selector/conditions: %#v", detail)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "api-v3-abcde" {
		t.Fatalf("unexpected replicaset pods: %#v", detail.Pods)
	}
	if len(detail.Events) != 2 {
		t.Fatalf("unexpected replicaset events: %#v", detail.Events)
	}
}

func TestGetJobDetailIncludesOwnerPodsConditionsAndEvents(t *testing.T) {
	controller := true
	parallelism := int32(2)
	completions := int32(3)
	backoffLimit := int32(4)
	activeDeadline := int64(300)
	ttl := int32(600)
	suspend := false
	manualSelector := true
	completionMode := batchv1.IndexedCompletion
	cronJobUID := k8stypes.UID("cronjob-report")
	jobUID := k8stypes.UID("job-report-123")
	podUID := k8stypes.UID("pod-report-0")
	client := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "report-123", Namespace: "default", UID: jobUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "CronJob", Name: "report", UID: cronJobUID, Controller: &controller}},
			},
			Spec: batchv1.JobSpec{
				Parallelism:             &parallelism,
				Completions:             &completions,
				BackoffLimit:            &backoffLimit,
				ActiveDeadlineSeconds:   &activeDeadline,
				TTLSecondsAfterFinished: &ttl,
				CompletionMode:          &completionMode,
				Suspend:                 &suspend,
				ManualSelector:          &manualSelector,
				Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"batch.kubernetes.io/job-name": "report-123"}},
				Template:                corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker", Image: "example/report:v1"}}}},
			},
			Status: batchv1.JobStatus{
				Active: 1, Succeeded: 1, Failed: 1,
				Conditions: []batchv1.JobCondition{
					{Type: batchv1.JobFailureTarget, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded", Message: "job reached backoff limit"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "report-123-0-abcde", Namespace: "default", UID: podUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "Job", Name: "report-123", UID: jobUID, Controller: &controller}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodFailed},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "unrelated", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "Job", Name: "other", Controller: &controller}},
			},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "job-backoff", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Job", Name: "report-123", UID: jobUID},
			Type:           "Warning", Reason: "BackoffLimitExceeded", Message: "job failed",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "pod-failed", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "report-123-0-abcde", UID: podUID},
			Type:           "Warning", Reason: "Failed", Message: "worker exited 1",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "stale-pod-event", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "report-123-0-abcde", UID: k8stypes.UID("old-pod")},
			Type:           "Warning", Reason: "Failed", Message: "stale",
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetJob(context.Background(), "default", "report-123")
	if err != nil {
		t.Fatalf("get job detail: %v", err)
	}
	if detail.Parallelism != 2 || detail.Completions != 3 || detail.BackoffLimit != 4 || detail.ActiveDeadlineSeconds != 300 || detail.TTLSecondsAfterFinished != 600 {
		t.Fatalf("unexpected job policy detail: %#v", detail)
	}
	if detail.CompletionMode != "Indexed" || !detail.ManualSelector || detail.Selector["batch.kubernetes.io/job-name"] != "report-123" {
		t.Fatalf("unexpected job selector/mode detail: %#v", detail)
	}
	if detail.Owner == nil || detail.Owner.Kind != "CronJob" || detail.Owner.Name != "report" {
		t.Fatalf("unexpected job owner: %#v", detail.Owner)
	}
	if len(detail.Images) != 1 || detail.Images[0] != "example/report:v1" || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected job images/conditions: %#v", detail)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "report-123-0-abcde" || len(detail.Events) != 2 {
		t.Fatalf("unexpected job diagnostic associations: pods=%#v events=%#v", detail.Pods, detail.Events)
	}
}

func TestGetCronJobDetailIncludesPolicyJobsPodsAndEvents(t *testing.T) {
	controller := true
	timeZone := "Asia/Shanghai"
	startingDeadline := int64(90)
	successHistory := int32(4)
	failedHistory := int32(2)
	parallelism := int32(2)
	completions := int32(2)
	backoffLimit := int32(3)
	ttl := int32(600)
	cronJobUID := k8stypes.UID("cronjob-report")
	jobUID := k8stypes.UID("job-report-100")
	podUID := k8stypes.UID("pod-report-100")
	lastSchedule := metav1.NewTime(time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC))
	lastSuccessful := metav1.NewTime(time.Date(2026, 8, 12, 8, 1, 0, 0, time.UTC))
	client := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Name: "report", Namespace: "default", UID: cronJobUID},
			Spec: batchv1.CronJobSpec{
				Schedule:                   "0 * * * *",
				TimeZone:                   &timeZone,
				StartingDeadlineSeconds:    &startingDeadline,
				ConcurrencyPolicy:          batchv1.ForbidConcurrent,
				SuccessfulJobsHistoryLimit: &successHistory,
				FailedJobsHistoryLimit:     &failedHistory,
				JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{
					Parallelism:             &parallelism,
					Completions:             &completions,
					BackoffLimit:            &backoffLimit,
					TTLSecondsAfterFinished: &ttl,
					Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers:    []corev1.Container{{Name: "worker", Image: "example/report:v2"}},
					}},
				}},
			},
			Status: batchv1.CronJobStatus{LastScheduleTime: &lastSchedule, LastSuccessfulTime: &lastSuccessful},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "report-100", Namespace: "default", UID: jobUID,
				CreationTimestamp: metav1.NewTime(time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)),
				OwnerReferences:   []metav1.OwnerReference{{Kind: "CronJob", Name: "report", UID: cronJobUID, Controller: &controller}},
			},
			Spec:   batchv1.JobSpec{Completions: &completions},
			Status: batchv1.JobStatus{Succeeded: 2, Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "unrelated", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "CronJob", Name: "other", Controller: &controller}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "report-100-abcde", Namespace: "default", UID: podUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "Job", Name: "report-100", UID: jobUID, Controller: &controller}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "cronjob-created", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "CronJob", Name: "report", UID: cronJobUID},
			Reason:         "SuccessfulCreate",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "job-complete", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Job", Name: "report-100", UID: jobUID},
			Reason:         "Completed",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "pod-complete", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "report-100-abcde", UID: podUID},
			Reason:         "Completed",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "stale-job", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Job", Name: "report-100", UID: k8stypes.UID("old-job")},
			Reason:         "Old",
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetCronJob(context.Background(), "default", "report")
	if err != nil {
		t.Fatalf("get cronjob detail: %v", err)
	}
	if detail.Schedule != "0 * * * *" || detail.TimeZone != timeZone || detail.ConcurrencyPolicy != "Forbid" || detail.StartingDeadlineSeconds != 90 {
		t.Fatalf("unexpected cronjob schedule policy: %#v", detail)
	}
	if detail.SuccessfulJobsHistoryLimit != 4 || detail.FailedJobsHistoryLimit != 2 || !detail.LastSuccessfulTime.Equal(lastSuccessful.Time) {
		t.Fatalf("unexpected cronjob history policy/status: %#v", detail)
	}
	if detail.JobTemplate.Parallelism != 2 || detail.JobTemplate.Completions != 2 || detail.JobTemplate.BackoffLimit != 3 || detail.JobTemplate.TTLSecondsAfterFinished != 600 {
		t.Fatalf("unexpected cronjob job template: %#v", detail.JobTemplate)
	}
	if detail.JobTemplate.RestartPolicy != "Never" || len(detail.JobTemplate.Images) != 1 || detail.JobTemplate.Images[0] != "example/report:v2" {
		t.Fatalf("unexpected cronjob pod template: %#v", detail.JobTemplate)
	}
	if len(detail.Jobs) != 1 || detail.Jobs[0].Name != "report-100" || len(detail.Jobs[0].Pods) != 1 {
		t.Fatalf("unexpected cronjob jobs/pods: %#v", detail.Jobs)
	}
	if len(detail.Events) != 3 {
		t.Fatalf("unexpected cronjob events: %#v", detail.Events)
	}
}

func TestGetServiceDetailPrefersEndpointSlicesAndAssociatesPodsAndEvents(t *testing.T) {
	serviceUID := k8stypes.UID("service-api")
	sliceUID := k8stypes.UID("slice-api-v1")
	readyPodUID := k8stypes.UID("pod-api-ready")
	pendingPodUID := k8stypes.UID("pod-api-pending")
	manualPodUID := k8stypes.UID("pod-api-manual")
	appProtocol := "http"
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	internalTrafficPolicy := corev1.ServiceInternalTrafficPolicyLocal
	portName := "http"
	endpointPort := int32(8080)
	endpointProtocol := corev1.ProtocolTCP
	nodeName := "worker-1"
	zone := "zone-a"
	ready := true
	notReady := false
	serving := true
	terminating := true
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default", UID: serviceUID},
			Spec: corev1.ServiceSpec{
				Type:                     corev1.ServiceTypeLoadBalancer,
				ClusterIP:                "10.43.0.10",
				ClusterIPs:               []string{"10.43.0.10", "fd00::10"},
				IPFamilies:               []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol},
				IPFamilyPolicy:           &ipFamilyPolicy,
				Selector:                 map[string]string{"app": "api"},
				ExternalTrafficPolicy:    corev1.ServiceExternalTrafficPolicyLocal,
				InternalTrafficPolicy:    &internalTrafficPolicy,
				SessionAffinity:          corev1.ServiceAffinityClientIP,
				PublishNotReadyAddresses: true,
				LoadBalancerSourceRanges: []string{"10.0.0.0/8"},
				Ports: []corev1.ServicePort{{
					Name: "http", Protocol: corev1.ProtocolTCP, Port: 80,
					TargetPort: intstr.FromString("http"), NodePort: 30080, AppProtocol: &appProtocol,
				}},
			},
		},
		&discoveryv1.EndpointSlice{
			ObjectMeta:  metav1.ObjectMeta{Name: "api-v1", Namespace: "default", UID: sliceUID, Labels: map[string]string{discoveryv1.LabelServiceName: "api"}},
			AddressType: discoveryv1.AddressTypeIPv4,
			Ports:       []discoveryv1.EndpointPort{{Name: &portName, Port: &endpointPort, Protocol: &endpointProtocol}},
			Endpoints: []discoveryv1.Endpoint{
				{
					Addresses: []string{"10.42.0.10"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready},
					NodeName: &nodeName, Zone: &zone,
					TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "api-ready", UID: readyPodUID},
				},
				{
					Addresses: []string{"10.42.0.11"}, Conditions: discoveryv1.EndpointConditions{Ready: &notReady, Serving: &serving, Terminating: &terminating},
					TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "manual-pod", UID: manualPodUID},
				},
			},
		},
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default", UID: k8stypes.UID("legacy-api")},
			Subsets:    []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "10.42.0.99"}}}},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-ready", Namespace: "default", UID: readyPodUID, Labels: map[string]string{"app": "api"}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-pending", Namespace: "default", UID: pendingPodUID, Labels: map[string]string{"app": "api"}}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "manual-pod", Namespace: "default", UID: manualPodUID, Labels: map[string]string{"app": "manual"}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default", Labels: map[string]string{"app": "other"}}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "service-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Service", Name: "api", UID: serviceUID}, Reason: "Updated", LastTimestamp: metav1.NewTime(now.Add(-3 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "slice-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "EndpointSlice", Name: "api-v1", UID: sliceUID}, Reason: "Synced", LastTimestamp: metav1.NewTime(now.Add(-2 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pod-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-ready", UID: readyPodUID}, Reason: "Ready", LastTimestamp: metav1.NewTime(now.Add(-time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stale-slice-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "EndpointSlice", Name: "api-v1", UID: k8stypes.UID("old-slice")}, Reason: "Stale"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "legacy-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Endpoints", Name: "api", UID: k8stypes.UID("legacy-api")}, Reason: "Legacy"},
	)
	service := NewService(client, "test")

	detail, err := service.GetService(context.Background(), "default", "api")
	if err != nil {
		t.Fatalf("get service detail: %v", err)
	}
	if detail.Type != "LoadBalancer" || len(detail.ClusterIPs) != 2 || detail.IPFamilyPolicy != "PreferDualStack" {
		t.Fatalf("unexpected service network identity: %#v", detail)
	}
	if detail.SessionAffinity != "ClientIP" || detail.ExternalTrafficPolicy != "Local" || detail.InternalTrafficPolicy != "Local" || !detail.PublishNotReadyAddresses {
		t.Fatalf("unexpected service traffic policy: %#v", detail)
	}
	if len(detail.PortDetails) != 1 || detail.PortDetails[0].TargetPort != "http" || detail.PortDetails[0].AppProtocol != "http" {
		t.Fatalf("unexpected service ports: %#v", detail.PortDetails)
	}
	if detail.EndpointSource != "EndpointSlice" || len(detail.Endpoints) != 2 {
		t.Fatalf("expected EndpointSlice endpoints only: %#v", detail.Endpoints)
	}
	if detail.Endpoints[0].Addresses[0] != "10.42.0.10" || !detail.Endpoints[0].Ready || !detail.Endpoints[0].Serving || detail.Endpoints[0].Terminating {
		t.Fatalf("unexpected ready endpoint: %#v", detail.Endpoints[0])
	}
	if detail.Endpoints[1].Ready || !detail.Endpoints[1].Serving || !detail.Endpoints[1].Terminating || detail.Endpoints[1].TargetName != "manual-pod" {
		t.Fatalf("unexpected terminating endpoint: %#v", detail.Endpoints[1])
	}
	if len(detail.Pods) != 3 || detail.Pods[0].Name != "api-pending" || detail.Pods[1].Name != "api-ready" || detail.Pods[2].Name != "manual-pod" {
		t.Fatalf("unexpected selector/targetRef pods: %#v", detail.Pods)
	}
	if len(detail.Events) != 4 || detail.Events[0].Reason != "Ready" || detail.Events[1].Reason != "Synced" || detail.Events[2].Reason != "Updated" {
		t.Fatalf("unexpected service events: %#v", detail.Events)
	}
	if detail.Events[3].Reason != "Legacy" {
		t.Fatalf("expected related legacy Endpoints event without mixing its addresses into endpoints: %#v", detail.Events)
	}
}

func TestGetServiceDetailFallsBackToLegacyEndpoints(t *testing.T) {
	serviceUID := k8stypes.UID("service-manual")
	endpointsUID := k8stypes.UID("endpoints-manual")
	podUID := k8stypes.UID("pod-manual")
	nodeName := "worker-1"
	client := fake.NewSimpleClientset(
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "manual", Namespace: "default", UID: serviceUID}, Spec: corev1.ServiceSpec{ClusterIP: "None"}},
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "manual", Namespace: "default", UID: endpointsUID},
			Subsets: []corev1.EndpointSubset{{
				Ports:             []corev1.EndpointPort{{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP}},
				Addresses:         []corev1.EndpointAddress{{IP: "10.42.0.20", NodeName: &nodeName, TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: "manual-ready", UID: podUID}}},
				NotReadyAddresses: []corev1.EndpointAddress{{IP: "10.42.0.21"}},
			}},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "manual-ready", Namespace: "default", UID: podUID}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default"}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "endpoints-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Endpoints", Name: "manual", UID: endpointsUID}, Reason: "Updated"},
	)
	service := NewService(client, "test")

	detail, err := service.GetService(context.Background(), "default", "manual")
	if err != nil {
		t.Fatalf("get service detail: %v", err)
	}
	if detail.EndpointSource != "Endpoints" || len(detail.Endpoints) != 2 {
		t.Fatalf("expected legacy Endpoints fallback: %#v", detail)
	}
	if !detail.Endpoints[0].Ready || detail.Endpoints[1].Ready || detail.Endpoints[1].Serving {
		t.Fatalf("unexpected legacy readiness: %#v", detail.Endpoints)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "manual-ready" {
		t.Fatalf("unexpected targetRef pods: %#v", detail.Pods)
	}
	if len(detail.Events) != 1 || detail.Events[0].InvolvedKind != "Endpoints" {
		t.Fatalf("unexpected legacy endpoint events: %#v", detail.Events)
	}
}

func TestGetIngressDetailIncludesRulesBackendServicesPodsAndEvents(t *testing.T) {
	ingressUID := k8stypes.UID("ingress-platform")
	apiServiceUID := k8stypes.UID("service-api")
	webServiceUID := k8stypes.UID("service-web")
	apiSliceUID := k8stypes.UID("slice-api")
	webEndpointsUID := k8stypes.UID("endpoints-web")
	apiPodUID := k8stypes.UID("pod-api")
	webPodUID := k8stypes.UID("pod-web")
	pathType := networkingv1.PathTypePrefix
	portName := "http"
	portNumber := int32(8080)
	protocol := corev1.ProtocolTCP
	ready := true
	resourceAPIGroup := "example.io"
	now := time.Date(2026, 8, 12, 13, 0, 0, 0, time.UTC)
	client := fake.NewSimpleClientset(
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "platform", Namespace: "default", UID: ingressUID},
			Spec: networkingv1.IngressSpec{
				IngressClassName: stringPointer("traefik"),
				DefaultBackend: &networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
					Name: "web", Port: networkingv1.ServiceBackendPort{Number: 80},
				}},
				TLS: []networkingv1.IngressTLS{{Hosts: []string{"platform.example.com"}, SecretName: "platform-tls"}},
				Rules: []networkingv1.IngressRule{
					{
						Host: "platform.example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{
							{Path: "/api", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "api", Port: networkingv1.ServiceBackendPort{Name: "http"}}}},
							{Path: "/", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web", Port: networkingv1.ServiceBackendPort{Number: 80}}}},
						}}},
					},
					{
						Host: "broken.example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{
							{Path: "/wrong-port", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web", Port: networkingv1.ServiceBackendPort{Number: 81}}}},
							{Path: "/missing", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "missing", Port: networkingv1.ServiceBackendPort{Number: 80}}}},
							{Path: "/resource", PathType: &pathType, Backend: networkingv1.IngressBackend{Resource: &corev1.TypedLocalObjectReference{APIGroup: &resourceAPIGroup, Kind: "StorageBucket", Name: "assets"}}},
						}}},
					},
				},
			},
			Status: networkingv1.IngressStatus{LoadBalancer: networkingv1.IngressLoadBalancerStatus{Ingress: []networkingv1.IngressLoadBalancerIngress{{IP: "203.0.113.10"}}}},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default", UID: apiServiceUID},
			Spec:       corev1.ServiceSpec{ClusterIP: "10.43.0.10", Selector: map[string]string{"app": "api"}, Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}}},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default", UID: webServiceUID},
			Spec:       corev1.ServiceSpec{ClusterIP: "10.43.0.20", Selector: map[string]string{"app": "web"}, Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}}},
		},
		&discoveryv1.EndpointSlice{
			ObjectMeta:  metav1.ObjectMeta{Name: "api-slice", Namespace: "default", UID: apiSliceUID, Labels: map[string]string{discoveryv1.LabelServiceName: "api"}},
			AddressType: discoveryv1.AddressTypeIPv4,
			Ports:       []discoveryv1.EndpointPort{{Name: &portName, Port: &portNumber, Protocol: &protocol}},
			Endpoints: []discoveryv1.Endpoint{{
				Addresses: []string{"10.42.0.10"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready},
				TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "api-abcde", UID: apiPodUID},
			}},
		},
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default", UID: webEndpointsUID},
			Subsets: []corev1.EndpointSubset{{
				Addresses: []corev1.EndpointAddress{{IP: "10.42.0.20", TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "web-abcde", UID: webPodUID}}},
				Ports:     []corev1.EndpointPort{{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP}},
			}},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-abcde", Namespace: "default", UID: apiPodUID, Labels: map[string]string{"app": "api"}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "web-abcde", Namespace: "default", UID: webPodUID, Labels: map[string]string{"app": "web"}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default", Labels: map[string]string{"app": "other"}}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "ingress-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Ingress", Name: "platform", UID: ingressUID}, Reason: "Synced", LastTimestamp: metav1.NewTime(now.Add(-5 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "api-service-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Service", Name: "api", UID: apiServiceUID}, Reason: "APIService", LastTimestamp: metav1.NewTime(now.Add(-4 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "api-slice-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "EndpointSlice", Name: "api-slice", UID: apiSliceUID}, Reason: "APISlice", LastTimestamp: metav1.NewTime(now.Add(-3 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "web-endpoints-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Endpoints", Name: "web", UID: webEndpointsUID}, Reason: "WebEndpoints", LastTimestamp: metav1.NewTime(now.Add(-2 * time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "web-pod-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "web-abcde", UID: webPodUID}, Reason: "WebPod", LastTimestamp: metav1.NewTime(now.Add(-time.Minute))},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stale-ingress-event", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Ingress", Name: "platform", UID: k8stypes.UID("old-ingress")}, Reason: "Stale"},
	)
	service := NewService(client, "test")

	detail, err := service.GetIngress(context.Background(), "default", "platform")
	if err != nil {
		t.Fatalf("get ingress detail: %v", err)
	}
	if detail.ClassName != "traefik" || len(detail.Addresses) != 1 || detail.Addresses[0] != "203.0.113.10" || !detail.TLS {
		t.Fatalf("unexpected ingress summary: %#v", detail.IngressSummary)
	}
	if len(detail.TLSDetails) != 1 || detail.TLSDetails[0].SecretName != "platform-tls" || len(detail.TLSDetails[0].Hosts) != 1 {
		t.Fatalf("unexpected ingress TLS details: %#v", detail.TLSDetails)
	}
	if len(detail.Backends) != 6 {
		t.Fatalf("unexpected ingress backends: %#v", detail.Backends)
	}
	if !detail.Backends[0].IsDefault || detail.Backends[0].BackendName != "web" || detail.Backends[0].BackendPort != "80" || !detail.Backends[0].ServiceFound || !detail.Backends[0].ServicePortFound {
		t.Fatalf("unexpected default backend: %#v", detail.Backends[0])
	}
	if detail.Backends[1].Host != "platform.example.com" || detail.Backends[1].Path != "/api" || detail.Backends[1].PathType != "Prefix" || detail.Backends[1].BackendName != "api" || detail.Backends[1].BackendPort != "http" || !detail.Backends[1].ServicePortFound {
		t.Fatalf("unexpected rule backend: %#v", detail.Backends[1])
	}
	if detail.Backends[3].BackendName != "web" || !detail.Backends[3].ServiceFound || detail.Backends[3].ServicePortFound {
		t.Fatalf("expected existing Service with missing backend port to remain visible: %#v", detail.Backends[3])
	}
	if detail.Backends[4].BackendName != "missing" || detail.Backends[4].ServiceFound || detail.Backends[4].ServicePortFound {
		t.Fatalf("expected missing Service backend to remain visible: %#v", detail.Backends[4])
	}
	if detail.Backends[5].BackendKind != "StorageBucket" || detail.Backends[5].BackendAPIGroup != "example.io" || detail.Backends[5].BackendName != "assets" {
		t.Fatalf("unexpected resource backend: %#v", detail.Backends[5])
	}
	if len(detail.Services) != 2 || detail.Services[0].Name != "web" || detail.Services[1].Name != "api" {
		t.Fatalf("expected de-duplicated backend services in first-reference order: %#v", detail.Services)
	}
	if detail.Services[0].EndpointSource != "Endpoints" || len(detail.Services[0].Pods) != 1 || detail.Services[0].Pods[0].Name != "web-abcde" {
		t.Fatalf("unexpected legacy-backed web Service: %#v", detail.Services[0])
	}
	if detail.Services[1].EndpointSource != "EndpointSlice" || len(detail.Services[1].Pods) != 1 || detail.Services[1].Pods[0].Name != "api-abcde" {
		t.Fatalf("unexpected EndpointSlice-backed api Service: %#v", detail.Services[1])
	}
	if len(detail.Events) != 5 || detail.Events[0].Reason != "WebPod" || detail.Events[4].Reason != "Synced" {
		t.Fatalf("unexpected ingress-associated events: %#v", detail.Events)
	}
}

func stringPointer(value string) *string {
	return &value
}

func TestGetStatefulSetDetailAndYAML(t *testing.T) {
	replicas := int32(3)
	storageClass := "local-path"
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "default", Labels: map[string]string{"app": "database"}},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         "database-headless",
			PodManagementPolicy: appsv1.ParallelPodManagement,
			UpdateStrategy:      appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
			Selector:            &metav1.LabelSelector{MatchLabels: map[string]string{"app": "database"}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "database", Image: "postgres:16-alpine"}}},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "data"},
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: &storageClass,
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
						},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   2,
			CurrentReplicas: 3,
			UpdatedReplicas: 2,
			CurrentRevision: "database-a",
			UpdateRevision:  "database-b",
		},
	})
	service := NewService(client, "test")

	detail, err := service.GetStatefulSet(context.Background(), "default", "database")
	if err != nil {
		t.Fatalf("get statefulset detail: %v", err)
	}
	if detail.Replicas != 3 || detail.ReadyReplicas != 2 || detail.ServiceName != "database-headless" {
		t.Fatalf("unexpected statefulset detail: %#v", detail)
	}
	if detail.UpdateStrategy != string(appsv1.RollingUpdateStatefulSetStrategyType) || len(detail.VolumeClaims) != 1 {
		t.Fatalf("expected update strategy and volume claim details: %#v", detail)
	}
	if detail.VolumeClaims[0].RequestedStorage != "5Gi" || detail.VolumeClaims[0].StorageClass != storageClass {
		t.Fatalf("unexpected volume claim detail: %#v", detail.VolumeClaims[0])
	}

	raw, err := service.StatefulSetYAML(context.Background(), "default", "database")
	if err != nil {
		t.Fatalf("get statefulset yaml: %v", err)
	}
	if !strings.Contains(raw, "kind: StatefulSet") || !strings.Contains(raw, "name: database") {
		t.Fatalf("unexpected statefulset yaml: %s", raw)
	}
}

func TestGetDaemonSetDetailAndYAML(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.DaemonSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
		ObjectMeta: metav1.ObjectMeta{Name: "node-agent", Namespace: "default", Labels: map[string]string{"app": "node-agent"}},
		Spec: appsv1.DaemonSetSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-agent"}},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{"kubernetes.io/os": "linux"},
					Tolerations: []corev1.Toleration{
						{Key: "node-role.kubernetes.io/control-plane", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
					},
					Containers: []corev1.Container{{Name: "agent", Image: "example/agent:v1"}},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 1,
			CurrentNumberScheduled: 1,
			NumberReady:            1,
			UpdatedNumberScheduled: 1,
			NumberAvailable:        1,
			NumberMisscheduled:     0,
			NumberUnavailable:      0,
		},
	})
	service := NewService(client, "test")

	detail, err := service.GetDaemonSet(context.Background(), "default", "node-agent")
	if err != nil {
		t.Fatalf("get daemonset detail: %v", err)
	}
	if detail.DesiredNumber != 1 || detail.ReadyNumber != 1 || detail.UpdateStrategy != string(appsv1.RollingUpdateDaemonSetStrategyType) {
		t.Fatalf("unexpected daemonset detail: %#v", detail)
	}
	if detail.NodeSelector["kubernetes.io/os"] != "linux" || len(detail.Tolerations) != 1 {
		t.Fatalf("expected scheduling details: %#v", detail)
	}
	if detail.Tolerations[0] != "node-role.kubernetes.io/control-plane:Exists:NoSchedule" {
		t.Fatalf("unexpected toleration: %s", detail.Tolerations[0])
	}

	raw, err := service.DaemonSetYAML(context.Background(), "default", "node-agent")
	if err != nil {
		t.Fatalf("get daemonset yaml: %v", err)
	}
	if !strings.Contains(raw, "kind: DaemonSet") || !strings.Contains(raw, "name: node-agent") {
		t.Fatalf("unexpected daemonset yaml: %s", raw)
	}
}

func TestGetPVCDetailIncludesStatefulSetPodMountsAndFilteredEvents(t *testing.T) {
	controller := true
	readOnly := true
	pvcUID := k8stypes.UID("pvc-data-stateful-demo-0")
	pvUID := k8stypes.UID("pv-data-stateful-demo-0")
	statefulSetUID := k8stypes.UID("statefulset-demo")
	podUID := k8stypes.UID("pod-stateful-demo-0")
	storageClass := "local-path"
	volumeMode := corev1.PersistentVolumeBlock
	apiGroup := "snapshot.storage.k8s.io"
	dataSourceNamespace := "snapshots"
	client := fake.NewSimpleClientset(
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "stateful-demo", Namespace: "default", UID: statefulSetUID}},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data-stateful-demo-0", Namespace: "default", UID: pvcUID},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClass,
				VolumeName:       "pv-stateful-demo-0",
				VolumeMode:       &volumeMode,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels:      map[string]string{"disk": "fast"},
					MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "zone", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}}},
				},
				DataSourceRef: &corev1.TypedObjectReference{APIGroup: &apiGroup, Kind: "VolumeSnapshot", Name: "database-snapshot", Namespace: &dataSourceNamespace},
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase:       corev1.ClaimBound,
				Capacity:    corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Conditions:  []corev1.PersistentVolumeClaimCondition{{Type: corev1.PersistentVolumeClaimResizing, Status: corev1.ConditionFalse, Reason: "Stable"}},
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "pv-stateful-demo-0", UID: pvUID},
			Spec: corev1.PersistentVolumeSpec{
				StorageClassName: storageClass,
				ClaimRef:         &corev1.ObjectReference{Namespace: "default", Name: "data-stateful-demo-0", UID: pvcUID},
				Capacity:         corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				PersistentVolumeSource: corev1.PersistentVolumeSource{CSI: &corev1.CSIPersistentVolumeSource{
					Driver:               "example.csi.internal",
					VolumeHandle:         "sensitive-provider-volume-id",
					ReadOnly:             readOnly,
					FSType:               "ext4",
					VolumeAttributes:     map[string]string{"token": "must-not-be-returned"},
					NodePublishSecretRef: &corev1.SecretReference{Name: "csi-secret", Namespace: "default"},
				}},
			},
			Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "stateful-demo-0", Namespace: "default", UID: podUID,
				OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", Name: "stateful-demo", UID: statefulSetUID, Controller: &controller}},
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-stateful-demo-0"}}},
					{Name: "other", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "other-claim"}}},
				},
				InitContainers: []corev1.Container{{
					Name:         "init-data",
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/bootstrap", SubPath: "seed", ReadOnly: true}},
				}},
				Containers: []corev1.Container{
					{Name: "database", VolumeDevices: []corev1.VolumeDevice{{Name: "data", DevicePath: "/dev/xvda"}}},
					{Name: "sidecar", VolumeMounts: []corev1.VolumeMount{{Name: "other", MountPath: "/other"}}},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default", UID: "unrelated-pod"},
			Spec:       corev1.PodSpec{Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "other-claim"}}}}},
		},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pvc-current", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "PersistentVolumeClaim", Namespace: "default", Name: "data-stateful-demo-0", UID: pvcUID}, Reason: "ProvisioningSucceeded"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pvc-old", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "PersistentVolumeClaim", Namespace: "default", Name: "data-stateful-demo-0", UID: "old-pvc"}, Reason: "OldPVC"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pv-current", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "PersistentVolume", Name: "pv-stateful-demo-0", UID: pvUID}, Reason: "Bound"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pod-current", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "stateful-demo-0", UID: podUID}, Reason: "Started"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "pod-old", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "stateful-demo-0", UID: "old-pod"}, Reason: "OldPod"},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "stateful-current", Namespace: "default"}, InvolvedObject: corev1.ObjectReference{Kind: "StatefulSet", Namespace: "default", Name: "stateful-demo", UID: statefulSetUID}, Reason: "SuccessfulCreate"},
	)
	service := NewService(client, "test")

	detail, err := service.GetPVC(context.Background(), "default", "data-stateful-demo-0")
	if err != nil {
		t.Fatalf("get pvc detail: %v", err)
	}
	if detail.PV == nil || detail.PV.Name != "pv-stateful-demo-0" || detail.VolumeMode != string(corev1.PersistentVolumeBlock) {
		t.Fatalf("unexpected PVC/PV binding detail: %#v", detail)
	}
	if detail.DataSource == nil || detail.DataSource.Name != "database-snapshot" || detail.DataSource.Namespace != dataSourceNamespace {
		t.Fatalf("unexpected data source: %#v", detail.DataSource)
	}
	if detail.Selector["disk"] != "fast" || len(detail.SelectorExpressions) != 1 || len(detail.Conditions) != 1 {
		t.Fatalf("unexpected selector or conditions: %#v", detail)
	}
	if len(detail.Pods) != 1 || detail.Pods[0].Name != "stateful-demo-0" {
		t.Fatalf("expected only the PVC-consuming Pod, got %#v", detail.Pods)
	}
	if len(detail.Workloads) != 1 || detail.Workloads[0].Kind != "StatefulSet" || detail.Workloads[0].Name != "stateful-demo" {
		t.Fatalf("unexpected workloads: %#v", detail.Workloads)
	}
	if len(detail.Mounts) != 2 {
		t.Fatalf("expected init mount and container device, got %#v", detail.Mounts)
	}
	mounts := map[string]VolumeMountDetail{}
	for _, mount := range detail.Mounts {
		mounts[mount.ContainerName] = mount
	}
	if mounts["init-data"].MountPath != "/bootstrap" || mounts["init-data"].SubPath != "seed" || !mounts["init-data"].ReadOnly {
		t.Fatalf("unexpected initContainer mount: %#v", mounts["init-data"])
	}
	if mounts["database"].DevicePath != "/dev/xvda" || mounts["database"].ContainerType != "Container" {
		t.Fatalf("unexpected container device: %#v", mounts["database"])
	}
	if len(detail.Events) != 4 {
		t.Fatalf("expected current PVC/PV/Pod/StatefulSet events only, got %#v", detail.Events)
	}
	for _, event := range detail.Events {
		if event.Reason == "OldPVC" || event.Reason == "OldPod" {
			t.Fatalf("old UID event leaked into detail: %#v", event)
		}
	}

	pvDetail, err := service.GetPV(context.Background(), "pv-stateful-demo-0")
	if err != nil {
		t.Fatalf("get pv detail: %v", err)
	}
	if pvDetail.PVC == nil || pvDetail.PVC.Name != "data-stateful-demo-0" || len(pvDetail.Pods) != 1 || len(pvDetail.Workloads) != 1 {
		t.Fatalf("unexpected PV reverse associations: %#v", pvDetail)
	}
	if pvDetail.VolumeSourceType != "CSI" || pvDetail.VolumeSourceInfo["driver"] != "example.csi.internal" {
		t.Fatalf("unexpected safe CSI summary: %#v", pvDetail.VolumeSourceInfo)
	}
	if _, found := pvDetail.VolumeSourceInfo["token"]; found {
		t.Fatalf("CSI volume attributes must not be returned: %#v", pvDetail.VolumeSourceInfo)
	}
	if strings.Contains(fmt.Sprint(pvDetail.VolumeSourceInfo), "sensitive-provider-volume-id") || strings.Contains(fmt.Sprint(pvDetail.VolumeSourceInfo), "csi-secret") {
		t.Fatalf("CSI handle or SecretRef leaked: %#v", pvDetail.VolumeSourceInfo)
	}
}

func TestGetPVCDetailResolvesReplicaSetToDeployment(t *testing.T) {
	controller := true
	deploymentUID := k8stypes.UID("deployment-api")
	replicaSetUID := k8stypes.UID("replicaset-api")
	client := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "shared-data", Namespace: "default"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default", UID: deploymentUID}},
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name: "api-abc", Namespace: "default", UID: replicaSetUID,
			OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "api", UID: deploymentUID, Controller: &controller}},
		}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "api-abc-123", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "api-abc", UID: replicaSetUID, Controller: &controller}}},
			Spec: corev1.PodSpec{
				Volumes:    []corev1.Volume{{Name: "shared", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "shared-data"}}}},
				Containers: []corev1.Container{{Name: "api", VolumeMounts: []corev1.VolumeMount{{Name: "shared", MountPath: "/srv/data"}}}},
			},
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetPVC(context.Background(), "default", "shared-data")
	if err != nil {
		t.Fatalf("get deployment PVC detail: %v", err)
	}
	if len(detail.Workloads) != 1 || detail.Workloads[0].Kind != "Deployment" || detail.Workloads[0].Name != "api" {
		t.Fatalf("expected ReplicaSet to resolve to Deployment, got %#v", detail.Workloads)
	}
	if len(detail.Mounts) != 1 || detail.Mounts[0].MountPath != "/srv/data" {
		t.Fatalf("unexpected deployment mount: %#v", detail.Mounts)
	}
}

func TestGetPVCDetailWithoutConsumersOrValidPVBinding(t *testing.T) {
	pvcUID := k8stypes.UID("new-pvc")
	client := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "unused", Namespace: "default", UID: pvcUID},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "stale-pv"},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "stale-pv"},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{Namespace: "default", Name: "unused", UID: "old-pvc"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default"},
			Spec:       corev1.PodSpec{Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "other"}}}}},
		},
	)
	service := NewService(client, "test")

	detail, err := service.GetPVC(context.Background(), "default", "unused")
	if err != nil {
		t.Fatalf("get unused PVC detail: %v", err)
	}
	if detail.PV != nil || len(detail.Pods) != 0 || len(detail.Workloads) != 0 || len(detail.Mounts) != 0 {
		t.Fatalf("expected no associations for unused/stale PVC, got %#v", detail)
	}
	pvDetail, err := service.GetPV(context.Background(), "stale-pv")
	if err != nil {
		t.Fatalf("get stale PV detail: %v", err)
	}
	if pvDetail.PVC != nil || len(pvDetail.Pods) != 0 {
		t.Fatalf("stale UID claimRef must not resolve: %#v", pvDetail)
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
