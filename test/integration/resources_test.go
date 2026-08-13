//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const (
	demoNamespace        = "demo-app"
	statefulSetName      = "stateful-demo"
	standalonePodName    = "standalone-demo"
	daemonSetName        = "daemon-demo"
	jobName              = "job-demo"
	cronJobName          = "cronjob-demo"
	serviceName          = "nginx-demo"
	ingressName          = "nginx-demo"
	statefulScaleRoute   = "/api/v1/namespaces/:namespace/statefulsets/:name/scale"
	statefulRestartRoute = "/api/v1/namespaces/:namespace/statefulsets/:name/restart"
	daemonRestartRoute   = "/api/v1/namespaces/:namespace/daemonsets/:name/restart"
	podRestartRoute      = "/api/v1/namespaces/:namespace/pods/:pod/restart"
	cronJobSuspendRoute  = "/api/v1/namespaces/:namespace/cronjobs/:name/suspend"
	cronJobResumeRoute   = "/api/v1/namespaces/:namespace/cronjobs/:name/resume"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type integrationClient struct {
	baseURL string
	token   string
	http    *http.Client
}

type statefulSetDetail struct {
	Name           string                `json:"name"`
	Replicas       int32                 `json:"replicas"`
	ServiceName    string                `json:"service_name"`
	UpdateStrategy string                `json:"update_strategy"`
	VolumeClaims   []statefulVolumeClaim `json:"volume_claims"`
}

type deploymentDetail struct {
	Name        string                 `json:"name"`
	Strategy    string                 `json:"strategy"`
	ReplicaSets []deploymentReplicaSet `json:"replica_sets"`
	Pods        []deploymentPod        `json:"pods"`
	Events      []eventSummary         `json:"events"`
	Conditions  []deploymentCondition  `json:"conditions"`
}

type deploymentReplicaSet struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}

type deploymentPod struct {
	Name     string `json:"name"`
	NodeName string `json:"node_name"`
	Status   string `json:"status"`
}

type deploymentCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type replicaSetDetail struct {
	Name       string           `json:"name"`
	Revision   string           `json:"revision"`
	Owner      *ownerReference  `json:"owner"`
	Pods       []deploymentPod  `json:"pods"`
	Events     []eventSummary   `json:"events"`
	Conditions []workloadStatus `json:"conditions"`
}

type workloadStatus struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type statefulVolumeClaim struct {
	Name             string   `json:"name"`
	RequestedStorage string   `json:"requested_storage"`
	AccessModes      []string `json:"access_modes"`
}

type daemonSetDetail struct {
	Name             string `json:"name"`
	DesiredNumber    int32  `json:"desired_number"`
	ReadyNumber      int32  `json:"ready_number"`
	UpdateStrategy   string `json:"update_strategy"`
	UnavailableCount int32  `json:"unavailable_number"`
}

type jobDetail struct {
	Name           string           `json:"name"`
	Completions    int32            `json:"completions"`
	Succeeded      int32            `json:"succeeded"`
	Failed         int32            `json:"failed"`
	Active         int32            `json:"active"`
	Parallelism    int32            `json:"parallelism"`
	BackoffLimit   int32            `json:"backoff_limit"`
	CompletionMode string           `json:"completion_mode"`
	Owner          *ownerReference  `json:"owner"`
	Pods           []deploymentPod  `json:"pods"`
	Events         []eventSummary   `json:"events"`
	Conditions     []workloadStatus `json:"conditions"`
}

type cronJobDetail struct {
	Name                       string         `json:"name"`
	Schedule                   string         `json:"schedule"`
	TimeZone                   string         `json:"time_zone"`
	ConcurrencyPolicy          string         `json:"concurrency_policy"`
	StartingDeadlineSeconds    int64          `json:"starting_deadline_seconds"`
	SuccessfulJobsHistoryLimit int32          `json:"successful_jobs_history_limit"`
	FailedJobsHistoryLimit     int32          `json:"failed_jobs_history_limit"`
	Suspend                    bool           `json:"suspend"`
	JobTemplate                jobTemplate    `json:"job_template"`
	Jobs                       []jobDetail    `json:"jobs"`
	Events                     []eventSummary `json:"events"`
}

type serviceDetail struct {
	Name           string            `json:"name"`
	EndpointSource string            `json:"endpoint_source"`
	Endpoints      []serviceEndpoint `json:"endpoints"`
	Pods           []deploymentPod   `json:"pods"`
	Events         []eventSummary    `json:"events"`
}

type serviceEndpoint struct {
	Addresses  []string `json:"addresses"`
	Ready      bool     `json:"ready"`
	Serving    bool     `json:"serving"`
	TargetKind string   `json:"target_kind"`
	TargetName string   `json:"target_name"`
}

type ingressDetail struct {
	Name     string                 `json:"name"`
	Hosts    []string               `json:"hosts"`
	Backends []ingressBackendDetail `json:"backends"`
	Services []serviceDetail        `json:"services"`
	Events   []eventSummary         `json:"events"`
}

type ingressBackendDetail struct {
	Host             string `json:"host"`
	Path             string `json:"path"`
	PathType         string `json:"path_type"`
	BackendKind      string `json:"backend_kind"`
	BackendName      string `json:"backend_name"`
	BackendPort      string `json:"backend_port"`
	ServiceFound     bool   `json:"service_found"`
	ServicePortFound bool   `json:"service_port_found"`
}

type storagePVCDetail struct {
	Name       string            `json:"name"`
	VolumeName string            `json:"volume_name"`
	PV         *storagePVSummary `json:"pv"`
	Pods       []deploymentPod   `json:"pods"`
	Workloads  []ownerReference  `json:"workloads"`
	Mounts     []storageMount    `json:"mounts"`
	Events     []eventSummary    `json:"events"`
}

type storagePVDetail struct {
	Name             string             `json:"name"`
	PVC              *storagePVCSummary `json:"pvc"`
	Pods             []deploymentPod    `json:"pods"`
	Workloads        []ownerReference   `json:"workloads"`
	Mounts           []storageMount     `json:"mounts"`
	VolumeSourceType string             `json:"volume_source_type"`
	Events           []eventSummary     `json:"events"`
}

type storagePVSummary struct {
	Name string `json:"name"`
}

type storagePVCSummary struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type storageMount struct {
	PodName       string `json:"pod_name"`
	ContainerType string `json:"container_type"`
	ContainerName string `json:"container_name"`
	MountPath     string `json:"mount_path"`
	DevicePath    string `json:"device_path"`
}

type declaredResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   int64  `json:"pods"`
}

type resourceAllocation struct {
	Requests             declaredResources `json:"requests"`
	CPURequestPercent    float64           `json:"cpu_request_percent"`
	MemoryRequestPercent float64           `json:"memory_request_percent"`
	PodPercent           float64           `json:"pod_percent"`
}

type resourceReference struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type namespaceDetail struct {
	Name                   string                  `json:"name"`
	Status                 string                  `json:"status"`
	Counts                 namespaceResourceCounts `json:"counts"`
	Allocated              resourceAllocation      `json:"allocated"`
	Pods                   []deploymentPod         `json:"pods"`
	Workloads              []resourceReference     `json:"workloads"`
	Services               []serviceDetail         `json:"services"`
	Ingresses              []ingressDetail         `json:"ingresses"`
	PersistentVolumeClaims []storagePVCDetail      `json:"persistent_volume_claims"`
}

type namespaceResourceCounts struct {
	Pods                   int `json:"pods"`
	Deployments            int `json:"deployments"`
	StatefulSets           int `json:"statefulsets"`
	DaemonSets             int `json:"daemonsets"`
	Jobs                   int `json:"jobs"`
	CronJobs               int `json:"cronjobs"`
	Services               int `json:"services"`
	Ingresses              int `json:"ingresses"`
	PersistentVolumeClaims int `json:"persistent_volume_claims"`
}

type nodeDetail struct {
	Name        string              `json:"name"`
	Status      string              `json:"status"`
	Addresses   []nodeAddress       `json:"addresses"`
	Conditions  []workloadStatus    `json:"conditions"`
	Allocatable declaredResources   `json:"allocatable"`
	Allocated   resourceAllocation  `json:"allocated"`
	Pods        []deploymentPod     `json:"pods"`
	Workloads   []resourceReference `json:"workloads"`
}

type nodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type jobTemplate struct {
	Parallelism   int32    `json:"parallelism"`
	Completions   int32    `json:"completions"`
	BackoffLimit  int32    `json:"backoff_limit"`
	RestartPolicy string   `json:"restart_policy"`
	Images        []string `json:"images"`
}

type podDetail struct {
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	NodeName        string           `json:"node_name"`
	ControllerChain []ownerReference `json:"controller_chain"`
}

type ownerReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type eventSummary struct {
	Reason       string `json:"reason"`
	InvolvedKind string `json:"involved_kind"`
	InvolvedName string `json:"involved_name"`
}

type logResult struct {
	Lines []struct {
		Raw string `json:"raw"`
	} `json:"lines"`
}

type logStreamMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    *struct {
		Raw string `json:"raw"`
	} `json:"line"`
}

type auditLog struct {
	Action       string `json:"action"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
}

func TestStatefulSetAndPodRestartWorkflow(t *testing.T) {
	client := newIntegrationClient(t)
	client.login(t)

	kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "deployment/nginx-demo", "--timeout=180s")
	kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "deployment/log-demo", "--timeout=180s")
	kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "statefulset/"+statefulSetName, "--timeout=180s")
	kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "daemonset/"+daemonSetName, "--timeout=180s")
	kubectl(t, 2*time.Minute, "-n", demoNamespace, "wait", "--for=condition=Ready", "pod/"+standalonePodName, "--timeout=120s")
	kubectl(t, 2*time.Minute, "-n", demoNamespace, "wait", "--for=condition=complete", "job/"+jobName, "--timeout=120s")
	kubectl(t, 3*time.Minute, "-n", demoNamespace, "wait", "--for=condition=complete", "job", "-l", "app=cronjob-demo", "--timeout=150s")

	t.Run("deployment detail associations", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/deployments/log-demo", nil, http.StatusOK)
		var detail deploymentDetail
		decodeData(t, response, &detail)
		if detail.Name != "log-demo" || detail.Strategy != "RollingUpdate" {
			t.Fatalf("unexpected Deployment detail: %#v", detail)
		}
		if len(detail.ReplicaSets) == 0 || detail.ReplicaSets[0].Name == "" || detail.ReplicaSets[0].Revision == "" {
			t.Fatalf("expected Deployment-owned ReplicaSet with revision: %#v", detail.ReplicaSets)
		}
		if len(detail.Pods) != 1 || detail.Pods[0].Name == "" || detail.Pods[0].NodeName == "" {
			t.Fatalf("expected one Deployment Pod with scheduling detail: %#v", detail.Pods)
		}
		if len(detail.Conditions) == 0 {
			t.Fatal("expected Deployment conditions")
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "Deployment" && event.InvolvedKind != "ReplicaSet" && event.InvolvedKind != "Pod" {
				t.Fatalf("Deployment detail returned unrelated Event: %#v", event)
			}
		}
	})

	t.Run("service endpoints and pod associations", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/services/"+serviceName, nil, http.StatusOK)
		var detail serviceDetail
		decodeData(t, response, &detail)
		if detail.Name != serviceName || detail.EndpointSource != "EndpointSlice" {
			t.Fatalf("unexpected Service detail: %#v", detail)
		}
		if len(detail.Endpoints) < 2 {
			t.Fatalf("expected at least two Service endpoints: %#v", detail.Endpoints)
		}
		for _, endpoint := range detail.Endpoints {
			if len(endpoint.Addresses) == 0 || !endpoint.Ready || !endpoint.Serving {
				t.Fatalf("expected ready and serving endpoint: %#v", endpoint)
			}
			if endpoint.TargetKind != "Pod" || endpoint.TargetName == "" {
				t.Fatalf("expected Pod targetRef: %#v", endpoint)
			}
		}
		if len(detail.Pods) != 2 {
			t.Fatalf("expected two selector-associated nginx Pods: %#v", detail.Pods)
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "Service" && event.InvolvedKind != "EndpointSlice" && event.InvolvedKind != "Endpoints" && event.InvolvedKind != "Pod" {
				t.Fatalf("Service detail returned unrelated Event: %#v", event)
			}
		}
	})

	t.Run("ingress rule service endpoint and pod associations", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/ingresses/"+ingressName, nil, http.StatusOK)
		var detail ingressDetail
		decodeData(t, response, &detail)
		if detail.Name != ingressName || len(detail.Hosts) != 1 || detail.Hosts[0] != "nginx-demo.example.test" {
			t.Fatalf("unexpected Ingress detail: %#v", detail)
		}
		if len(detail.Backends) != 1 {
			t.Fatalf("expected one Ingress backend: %#v", detail.Backends)
		}
		backend := detail.Backends[0]
		if backend.Host != "nginx-demo.example.test" || backend.Path != "/" || backend.PathType != "Prefix" || backend.BackendKind != "Service" || backend.BackendName != serviceName || backend.BackendPort != "http" || !backend.ServiceFound || !backend.ServicePortFound {
			t.Fatalf("unexpected Ingress backend resolution: %#v", backend)
		}
		if len(detail.Services) != 1 || detail.Services[0].Name != serviceName || detail.Services[0].EndpointSource != "EndpointSlice" {
			t.Fatalf("expected one EndpointSlice-backed Service: %#v", detail.Services)
		}
		if len(detail.Services[0].Endpoints) < 2 || len(detail.Services[0].Pods) != 2 {
			t.Fatalf("expected nginx Service endpoints and Pods: %#v", detail.Services[0])
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "Ingress" && event.InvolvedKind != "Service" && event.InvolvedKind != "EndpointSlice" && event.InvolvedKind != "Endpoints" && event.InvolvedKind != "Pod" {
				t.Fatalf("Ingress detail returned unrelated Event: %#v", event)
			}
		}
	})

	t.Run("replicaset detail associations", func(t *testing.T) {
		replicaSetName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "replicasets", "-l", "app=log-demo", "-o", "jsonpath={.items[0].metadata.name}"))
		if replicaSetName == "" {
			t.Fatal("log-demo ReplicaSet not found")
		}
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/replicasets/"+replicaSetName, nil, http.StatusOK)
		var detail replicaSetDetail
		decodeData(t, response, &detail)
		if detail.Name != replicaSetName || detail.Revision == "" {
			t.Fatalf("unexpected ReplicaSet detail: %#v", detail)
		}
		if detail.Owner == nil || detail.Owner.Kind != "Deployment" || detail.Owner.Name != "log-demo" {
			t.Fatalf("unexpected ReplicaSet owner: %#v", detail.Owner)
		}
		if len(detail.Pods) != 1 || detail.Pods[0].Name == "" || detail.Pods[0].NodeName == "" {
			t.Fatalf("expected one directly owned Pod: %#v", detail.Pods)
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "ReplicaSet" && event.InvolvedKind != "Pod" {
				t.Fatalf("ReplicaSet detail returned unrelated Event: %#v", event)
			}
		}
	})

	t.Run("job detail associations and pod restart rejection", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/jobs/"+jobName, nil, http.StatusOK)
		var detail jobDetail
		decodeData(t, response, &detail)
		if detail.Name != jobName || detail.Completions != 1 || detail.Succeeded != 1 || detail.Failed != 0 || detail.Active != 0 {
			t.Fatalf("unexpected Job detail status: %#v", detail)
		}
		if detail.Parallelism != 1 || detail.BackoffLimit != 1 || detail.CompletionMode != "NonIndexed" || detail.Owner != nil {
			t.Fatalf("unexpected Job execution policy: %#v", detail)
		}
		if len(detail.Conditions) == 0 || len(detail.Pods) != 1 || detail.Pods[0].Name == "" || detail.Pods[0].Status != "Succeeded" {
			t.Fatalf("expected completed Job condition and directly owned Pod: %#v", detail)
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "Job" && event.InvolvedKind != "Pod" {
				t.Fatalf("Job detail returned unrelated Event: %#v", event)
			}
		}

		podName := detail.Pods[0].Name
		oldUID := podUID(t, podName)
		response = client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/pods/"+podName+"/restart?confirm=true", nil, http.StatusConflict)
		if response.Code != 10005 {
			t.Fatalf("expected Job Pod restart conflict code 10005, got %d (%s)", response.Code, response.Message)
		}
		if currentUID := podUID(t, podName); currentUID != oldUID {
			t.Fatalf("Job Pod was unexpectedly replaced: old=%s new=%s", oldUID, currentUID)
		}
	})

	t.Run("cronjob detail associations and suspend resume", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/cronjobs/"+cronJobName, nil, http.StatusOK)
		var detail cronJobDetail
		decodeData(t, response, &detail)
		if detail.Name != cronJobName || detail.Schedule != "* * * * *" || detail.TimeZone != "Etc/UTC" || detail.ConcurrencyPolicy != "Forbid" {
			t.Fatalf("unexpected CronJob schedule policy: %#v", detail)
		}
		if detail.StartingDeadlineSeconds != 60 || detail.SuccessfulJobsHistoryLimit != 2 || detail.FailedJobsHistoryLimit != 1 {
			t.Fatalf("unexpected CronJob history policy: %#v", detail)
		}
		if detail.JobTemplate.Parallelism != 1 || detail.JobTemplate.Completions != 1 || detail.JobTemplate.BackoffLimit != 1 || detail.JobTemplate.RestartPolicy != "Never" {
			t.Fatalf("unexpected CronJob Job template: %#v", detail.JobTemplate)
		}
		if len(detail.Jobs) == 0 || len(detail.Jobs[0].Pods) == 0 || detail.Jobs[0].Owner == nil || detail.Jobs[0].Owner.Kind != "CronJob" {
			t.Fatalf("expected CronJob-owned Job and descendant Pod: %#v", detail.Jobs)
		}
		for _, event := range detail.Events {
			if event.InvolvedKind != "CronJob" && event.InvolvedKind != "Job" && event.InvolvedKind != "Pod" {
				t.Fatalf("CronJob detail returned unrelated Event: %#v", event)
			}
		}

		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/cronjobs/"+cronJobName+"/suspend?confirm=true", nil, http.StatusOK)
		response = client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/cronjobs/"+cronJobName, nil, http.StatusOK)
		decodeData(t, response, &detail)
		if !detail.Suspend {
			t.Fatal("expected CronJob to be suspended")
		}

		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/cronjobs/"+cronJobName+"/resume?confirm=true", nil, http.StatusOK)
		response = client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/cronjobs/"+cronJobName, nil, http.StatusOK)
		decodeData(t, response, &detail)
		if detail.Suspend {
			t.Fatal("expected CronJob to be resumed")
		}
	})

	t.Run("statefulset detail", func(t *testing.T) {
		detail := client.statefulSet(t)
		if detail.Name != statefulSetName || detail.ServiceName != "stateful-demo-headless" {
			t.Fatalf("unexpected StatefulSet detail: %#v", detail)
		}
		if detail.Replicas != 2 || detail.UpdateStrategy != "RollingUpdate" {
			t.Fatalf("unexpected StatefulSet state: %#v", detail)
		}
		if len(detail.VolumeClaims) != 1 || detail.VolumeClaims[0].Name != "data" || detail.VolumeClaims[0].RequestedStorage != "32Mi" {
			t.Fatalf("unexpected StatefulSet volume claim templates: %#v", detail.VolumeClaims)
		}
	})

	t.Run("namespace detail associations and declared resources", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/"+demoNamespace, nil, http.StatusOK)
		var detail namespaceDetail
		decodeData(t, response, &detail)
		if detail.Name != demoNamespace || detail.Status != "Active" || detail.Counts.Pods == 0 {
			t.Fatalf("unexpected Namespace detail: %#v", detail)
		}
		if detail.Counts.Deployments < 2 || detail.Counts.StatefulSets < 1 || detail.Counts.DaemonSets < 1 || detail.Counts.Jobs < 1 || detail.Counts.CronJobs < 1 {
			t.Fatalf("expected workload counts in Namespace detail: %#v", detail.Counts)
		}
		if detail.Counts.Services < 2 || detail.Counts.Ingresses < 1 || detail.Counts.PersistentVolumeClaims < 2 || len(detail.Services) == 0 || len(detail.Ingresses) == 0 || len(detail.PersistentVolumeClaims) == 0 {
			t.Fatalf("expected network/storage associations in Namespace detail: %#v", detail)
		}
		if detail.Allocated.Requests.CPU == "" || detail.Allocated.Requests.CPU == "0" || detail.Allocated.Requests.Memory == "" || detail.Allocated.Requests.Memory == "0" || detail.Allocated.Requests.Pods == 0 {
			t.Fatalf("expected active Pod requests in Namespace detail: %#v", detail.Allocated)
		}
		expectedKinds := map[string]bool{"Deployment": false, "StatefulSet": false, "DaemonSet": false, "Job": false, "CronJob": false}
		for _, workload := range detail.Workloads {
			if _, found := expectedKinds[workload.Kind]; found {
				expectedKinds[workload.Kind] = true
			}
		}
		for kind, found := range expectedKinds {
			if !found {
				t.Fatalf("Namespace detail did not include %s workload: %#v", kind, detail.Workloads)
			}
		}
	})

	t.Run("node detail associations and declared allocation", func(t *testing.T) {
		nodeName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pod", "stateful-demo-0", "-o", "jsonpath={.spec.nodeName}"))
		if nodeName == "" {
			t.Fatal("stateful-demo-0 has no scheduled Node")
		}
		response := client.request(t, http.MethodGet, "/api/v1/nodes/"+nodeName, nil, http.StatusOK)
		var detail nodeDetail
		decodeData(t, response, &detail)
		if detail.Name != nodeName || detail.Status != "Ready" || len(detail.Conditions) == 0 || len(detail.Addresses) == 0 {
			t.Fatalf("unexpected Node detail metadata: %#v", detail)
		}
		if detail.Allocatable.CPU == "" || detail.Allocatable.Memory == "" || detail.Allocatable.Pods == 0 {
			t.Fatalf("expected Node allocatable CPU/Memory/Pods: %#v", detail.Allocatable)
		}
		if detail.Allocated.Requests.CPU == "" || detail.Allocated.Requests.CPU == "0" || detail.Allocated.Requests.Memory == "" || detail.Allocated.Requests.Memory == "0" || detail.Allocated.Requests.Pods == 0 {
			t.Fatalf("expected Node Pod requests and count: %#v", detail.Allocated)
		}
		foundPod := false
		for _, pod := range detail.Pods {
			if pod.Name == "stateful-demo-0" {
				foundPod = true
				break
			}
		}
		if !foundPod {
			t.Fatalf("Node detail did not include stateful-demo-0: %#v", detail.Pods)
		}
		foundWorkload := false
		for _, workload := range detail.Workloads {
			if workload.Kind == "StatefulSet" && workload.Namespace == demoNamespace && workload.Name == statefulSetName {
				foundWorkload = true
				break
			}
		}
		if !foundWorkload {
			t.Fatalf("Node detail did not include StatefulSet association: %#v", detail.Workloads)
		}
	})

	t.Run("pvc and pv storage associations", func(t *testing.T) {
		pvcName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pvc", "data-stateful-demo-0", "-o", "jsonpath={.metadata.name}"))
		if pvcName == "" {
			t.Fatal("StatefulSet PVC was not found")
		}
		pvName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pvc", pvcName, "-o", "jsonpath={.spec.volumeName}"))
		if pvName == "" {
			t.Fatalf("StatefulSet PVC %s is not bound to a PV", pvcName)
		}

		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/persistentvolumeclaims/"+pvcName, nil, http.StatusOK)
		var pvcDetail storagePVCDetail
		decodeData(t, response, &pvcDetail)
		if pvcDetail.Name != pvcName || pvcDetail.VolumeName != pvName || pvcDetail.PV == nil || pvcDetail.PV.Name != pvName {
			t.Fatalf("unexpected PVC/PV binding detail: %#v", pvcDetail)
		}
		if len(pvcDetail.Pods) != 1 || pvcDetail.Pods[0].Name != "stateful-demo-0" {
			t.Fatalf("expected one StatefulSet Pod for PVC %s: %#v", pvcName, pvcDetail.Pods)
		}
		if len(pvcDetail.Workloads) != 1 || pvcDetail.Workloads[0].Kind != "StatefulSet" || pvcDetail.Workloads[0].Name != statefulSetName {
			t.Fatalf("expected StatefulSet workload association: %#v", pvcDetail.Workloads)
		}
		if len(pvcDetail.Mounts) != 1 || pvcDetail.Mounts[0].PodName != "stateful-demo-0" || pvcDetail.Mounts[0].ContainerType != "Container" || pvcDetail.Mounts[0].ContainerName != "worker" || pvcDetail.Mounts[0].MountPath != "/data" {
			t.Fatalf("unexpected StatefulSet PVC mount detail: %#v", pvcDetail.Mounts)
		}
		for _, event := range pvcDetail.Events {
			if event.InvolvedKind != "PersistentVolumeClaim" && event.InvolvedKind != "PersistentVolume" && event.InvolvedKind != "Pod" && event.InvolvedKind != "StatefulSet" {
				t.Fatalf("PVC detail returned unrelated Event: %#v", event)
			}
		}

		response = client.request(t, http.MethodGet, "/api/v1/persistentvolumes/"+pvName, nil, http.StatusOK)
		var pvDetail storagePVDetail
		decodeData(t, response, &pvDetail)
		if pvDetail.Name != pvName || pvDetail.PVC == nil || pvDetail.PVC.Namespace != demoNamespace || pvDetail.PVC.Name != pvcName {
			t.Fatalf("unexpected PV/PVC reverse binding detail: %#v", pvDetail)
		}
		if len(pvDetail.Pods) != 1 || pvDetail.Pods[0].Name != "stateful-demo-0" || len(pvDetail.Workloads) != 1 || pvDetail.Workloads[0].Kind != "StatefulSet" {
			t.Fatalf("unexpected PV Pod/Workload reverse associations: %#v", pvDetail)
		}
		if len(pvDetail.Mounts) != 1 || pvDetail.Mounts[0].MountPath != "/data" || pvDetail.VolumeSourceType == "" {
			t.Fatalf("unexpected PV source/mount detail: %#v", pvDetail)
		}
	})

	t.Run("statefulset scale", func(t *testing.T) {
		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/statefulsets/stateful-demo/scale", map[string]int32{"replicas": 3}, http.StatusOK)
		kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "statefulset/"+statefulSetName, "--timeout=180s")
		if detail := client.statefulSet(t); detail.Replicas != 3 {
			t.Fatalf("expected 3 replicas after scale up, got %d", detail.Replicas)
		}

		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/statefulsets/stateful-demo/scale", map[string]int32{"replicas": 2}, http.StatusOK)
		kubectl(t, 2*time.Minute, "-n", demoNamespace, "wait", "--for=delete", "pod/stateful-demo-2", "--timeout=120s")
		kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "statefulset/"+statefulSetName, "--timeout=180s")
		if detail := client.statefulSet(t); detail.Replicas != 2 {
			t.Fatalf("expected 2 replicas after scale down, got %d", detail.Replicas)
		}
	})

	t.Run("statefulset rolling restart", func(t *testing.T) {
		oldUID0 := podUID(t, "stateful-demo-0")
		oldUID1 := podUID(t, "stateful-demo-1")
		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/statefulsets/stateful-demo/restart", nil, http.StatusOK)
		kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "statefulset/"+statefulSetName, "--timeout=180s")
		waitForPodUIDChange(t, "stateful-demo-0", oldUID0, 2*time.Minute)
		waitForPodUIDChange(t, "stateful-demo-1", oldUID1, 2*time.Minute)
	})

	t.Run("statefulset on-delete restart rejected", func(t *testing.T) {
		kubectl(t, 30*time.Second, "-n", demoNamespace, "patch", "statefulset", statefulSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"OnDelete","rollingUpdate":null}}}`)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = kubectlOutput(ctx, "-n", demoNamespace, "patch", "statefulset", statefulSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"RollingUpdate","rollingUpdate":{"partition":0}}}}`)
		})

		response := client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/statefulsets/stateful-demo/restart", nil, http.StatusConflict)
		if response.Code != 10005 {
			t.Fatalf("expected conflict code 10005, got %d (%s)", response.Code, response.Message)
		}
		kubectl(t, 30*time.Second, "-n", demoNamespace, "patch", "statefulset", statefulSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"RollingUpdate","rollingUpdate":{"partition":0}}}}`)
	})

	t.Run("statefulset managed pod restart", func(t *testing.T) {
		oldUID := podUID(t, "stateful-demo-0")
		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/pods/stateful-demo-0/restart?confirm=true", nil, http.StatusOK)
		waitForPodUIDChange(t, "stateful-demo-0", oldUID, 2*time.Minute)
		kubectl(t, 3*time.Minute, "-n", demoNamespace, "wait", "--for=condition=Ready", "pod/stateful-demo-0", "--timeout=120s")
	})

	t.Run("daemonset detail and rolling restart", func(t *testing.T) {
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/daemonsets/daemon-demo", nil, http.StatusOK)
		var detail daemonSetDetail
		decodeData(t, response, &detail)
		if detail.Name != daemonSetName || detail.DesiredNumber < 1 || detail.ReadyNumber != detail.DesiredNumber || detail.UpdateStrategy != "RollingUpdate" {
			t.Fatalf("unexpected DaemonSet detail: %#v", detail)
		}

		podName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pods", "-l", "app=daemon-demo", "-o", "jsonpath={.items[0].metadata.name}"))
		if podName == "" {
			t.Fatal("daemonset pod not found")
		}
		oldUID := podUID(t, podName)
		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/daemonsets/daemon-demo/restart", nil, http.StatusOK)
		kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "daemonset/"+daemonSetName, "--timeout=180s")
		waitForControllerReplacement(t, "app=daemon-demo", oldUID, 1, 2*time.Minute)
	})

	t.Run("daemonset on-delete restart rejected", func(t *testing.T) {
		kubectl(t, 30*time.Second, "-n", demoNamespace, "patch", "daemonset", daemonSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"OnDelete","rollingUpdate":null}}}`)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = kubectlOutput(ctx, "-n", demoNamespace, "patch", "daemonset", daemonSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":1}}}}`)
		})

		response := client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/daemonsets/daemon-demo/restart", nil, http.StatusConflict)
		if response.Code != 10005 {
			t.Fatalf("expected conflict code 10005, got %d (%s)", response.Code, response.Message)
		}
		kubectl(t, 30*time.Second, "-n", demoNamespace, "patch", "daemonset", daemonSetName, "--type=merge", "-p", `{"spec":{"updateStrategy":{"type":"RollingUpdate","rollingUpdate":{"maxUnavailable":1}}}}`)
	})

	t.Run("deployment managed pod restart", func(t *testing.T) {
		podName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pods", "-l", "app=nginx-demo", "-o", "jsonpath={.items[0].metadata.name}"))
		if podName == "" {
			t.Fatal("deployment pod not found")
		}
		oldUID := podUID(t, podName)
		client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/pods/"+podName+"/restart?confirm=true", nil, http.StatusOK)
		waitForControllerReplacement(t, "app=nginx-demo", oldUID, 2, 2*time.Minute)
		kubectl(t, 4*time.Minute, "-n", demoNamespace, "rollout", "status", "deployment/nginx-demo", "--timeout=180s")
	})

	t.Run("pod diagnostic detail and filtered events", func(t *testing.T) {
		podName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pods", "-l", "app=log-demo", "-o", "jsonpath={.items[0].metadata.name}"))
		if podName == "" {
			t.Fatal("log-demo pod not found")
		}
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/pods/"+podName, nil, http.StatusOK)
		var detail podDetail
		decodeData(t, response, &detail)
		if detail.Name != podName || detail.NodeName == "" || len(detail.ControllerChain) != 2 {
			t.Fatalf("unexpected Pod diagnostic detail: %#v", detail)
		}
		if detail.ControllerChain[0].Kind != "ReplicaSet" || detail.ControllerChain[1].Kind != "Deployment" || detail.ControllerChain[1].Name != "log-demo" {
			t.Fatalf("unexpected Pod controller chain: %#v", detail.ControllerChain)
		}

		response = client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/events?involved_kind=Pod&involved_name="+podName, nil, http.StatusOK)
		var events []eventSummary
		decodeData(t, response, &events)
		if len(events) == 0 {
			t.Fatal("expected at least one event for log-demo Pod")
		}
		for _, event := range events {
			if event.InvolvedKind != "Pod" || event.InvolvedName != podName {
				t.Fatalf("event filter returned unrelated item: %#v", event)
			}
		}
	})

	t.Run("retained and realtime pod logs", func(t *testing.T) {
		podName := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pods", "-l", "app=log-demo", "-o", "jsonpath={.items[0].metadata.name}"))
		if podName == "" {
			t.Fatal("log-demo pod not found")
		}
		response := client.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/pods/"+podName+"/logs?container=logger&keyword=demo&limit=20", nil, http.StatusOK)
		var retained logResult
		decodeData(t, response, &retained)
		if len(retained.Lines) == 0 || !strings.Contains(retained.Lines[0].Raw, "msg=demo") {
			t.Fatalf("unexpected retained log result: %#v", retained)
		}

		message := client.followLogLine(t, podName, 20*time.Second)
		if message.Line == nil || !strings.Contains(message.Line.Raw, "msg=demo") {
			t.Fatalf("unexpected realtime log message: %#v", message)
		}
	})

	t.Run("standalone pod restart rejected", func(t *testing.T) {
		oldUID := podUID(t, standalonePodName)
		response := client.request(t, http.MethodPost, "/api/v1/namespaces/demo-app/pods/standalone-demo/restart?confirm=true", nil, http.StatusConflict)
		if response.Code != 10005 {
			t.Fatalf("expected conflict code 10005, got %d (%s)", response.Code, response.Message)
		}
		if currentUID := podUID(t, standalonePodName); currentUID != oldUID {
			t.Fatalf("standalone pod was unexpectedly replaced: old=%s new=%s", oldUID, currentUID)
		}
	})

	t.Run("audit records", func(t *testing.T) {
		items := waitForAuditRecords(t, client, 10*time.Second)
		if countAudit(items, statefulScaleRoute) < 2 {
			t.Fatalf("expected two StatefulSet scale audit records, got %#v", items)
		}
		if countAudit(items, statefulRestartRoute) < 2 {
			t.Fatalf("expected successful and rejected StatefulSet restart audit records, got %#v", items)
		}
		if countAudit(items, podRestartRoute) < 4 {
			t.Fatalf("expected controller and rejected Pod restart audit records, got %#v", items)
		}
		if countAudit(items, daemonRestartRoute) < 2 {
			t.Fatalf("expected successful and rejected DaemonSet restart audit records, got %#v", items)
		}
		if countAudit(items, cronJobSuspendRoute) < 1 || countAudit(items, cronJobResumeRoute) < 1 {
			t.Fatalf("expected CronJob suspend and resume audit records, got %#v", items)
		}
	})
}

func newIntegrationClient(t *testing.T) *integrationClient {
	t.Helper()
	baseURL := strings.TrimRight(os.Getenv("OPS_INTEGRATION_BASE_URL"), "/")
	if baseURL == "" {
		t.Fatal("OPS_INTEGRATION_BASE_URL is required")
	}
	return &integrationClient{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *integrationClient) login(t *testing.T) {
	t.Helper()
	response := c.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    os.Getenv("OPS_INTEGRATION_ADMIN_EMAIL"),
		"password": os.Getenv("OPS_INTEGRATION_ADMIN_PASSWORD"),
	}, http.StatusOK)
	var data struct {
		AccessToken string `json:"access_token"`
	}
	decodeData(t, response, &data)
	if data.AccessToken == "" {
		t.Fatal("login response did not include an access token")
	}
	c.token = data.AccessToken
}

func (c *integrationClient) statefulSet(t *testing.T) statefulSetDetail {
	t.Helper()
	response := c.request(t, http.MethodGet, "/api/v1/namespaces/demo-app/statefulsets/stateful-demo", nil, http.StatusOK)
	var detail statefulSetDetail
	decodeData(t, response, &detail)
	return detail
}

func (c *integrationClient) request(t *testing.T, method, path string, body any, expectedStatus int) apiEnvelope {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("execute %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("%s %s returned HTTP %d, expected %d: %s", method, path, resp.StatusCode, expectedStatus, raw)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode %s %s response: %v: %s", method, path, err, raw)
	}
	if expectedStatus < http.StatusBadRequest && envelope.Code != 0 {
		t.Fatalf("%s %s returned code %d: %s", method, path, envelope.Code, envelope.Message)
	}
	return envelope
}

func (c *integrationClient) followLogLine(t *testing.T, podName string, timeout time.Duration) logStreamMessage {
	t.Helper()
	websocketURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	websocketURL = strings.Replace(websocketURL, "https://", "wss://", 1)
	websocketURL += "/ws/v1/namespaces/demo-app/pods/" + podName + "/logs/follow?container=logger&keyword=demo&limit=1"
	dialer := websocket.Dialer{Subprotocols: []string{"ops-platform.logs.v1", "bearer." + c.token}}
	conn, response, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("connect realtime logs (HTTP %d): %v", status, err)
	}
	defer conn.Close()
	if conn.Subprotocol() != "ops-platform.logs.v1" {
		t.Fatalf("unexpected websocket subprotocol: %q", conn.Subprotocol())
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set realtime log deadline: %v", err)
	}
	for {
		var message logStreamMessage
		if err := conn.ReadJSON(&message); err != nil {
			t.Fatalf("read realtime log message: %v", err)
		}
		switch message.Type {
		case "line":
			return message
		case "error":
			t.Fatalf("realtime log stream returned error: %s", message.Message)
		}
	}
}

func decodeData(t *testing.T, envelope apiEnvelope, target any) {
	t.Helper()
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode response data: %v: %s", err, envelope.Data)
	}
}

func kubectl(t *testing.T, timeout time.Duration, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := kubectlOutput(ctx, args...)
	if err != nil {
		t.Fatalf("kubectl %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func kubectlOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func podUID(t *testing.T, name string) string {
	t.Helper()
	uid := strings.TrimSpace(kubectl(t, 30*time.Second, "-n", demoNamespace, "get", "pod", name, "-o", "jsonpath={.metadata.uid}"))
	if uid == "" {
		t.Fatalf("pod %s has no UID", name)
	}
	return uid
}

func waitForPodUIDChange(t *testing.T, name, oldUID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		output, err := kubectlOutput(ctx, "-n", demoNamespace, "get", "pod", name, "-o", "jsonpath={.metadata.uid}")
		cancel()
		uid := strings.TrimSpace(output)
		if err == nil && uid != "" && uid != oldUID {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("pod %s UID did not change from %s within %s", name, oldUID, timeout)
}

func waitForControllerReplacement(t *testing.T, selector, oldUID string, expectedPods int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		output, err := kubectlOutput(ctx, "-n", demoNamespace, "get", "pods", "-l", selector, "-o", "jsonpath={range .items[*]}{.metadata.uid}{\"\\n\"}{end}")
		cancel()
		if err == nil {
			uids := strings.Fields(output)
			if len(uids) >= expectedPods && !contains(uids, oldUID) {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("controller did not replace pod UID %s within %s", oldUID, timeout)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func countAudit(items []auditLog, resourceName string) int {
	count := 0
	for _, item := range items {
		if item.Action == http.MethodPost && item.ResourceName == resourceName && item.Namespace == demoNamespace {
			count++
		}
	}
	return count
}

func waitForAuditRecords(t *testing.T, client *integrationClient, timeout time.Duration) []auditLog {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last []auditLog
	for time.Now().Before(deadline) {
		response := client.request(t, http.MethodGet, "/api/v1/audit/logs?namespace=demo-app&limit=100", nil, http.StatusOK)
		var data struct {
			Items []auditLog `json:"items"`
		}
		decodeData(t, response, &data)
		last = data.Items
		if countAudit(last, statefulScaleRoute) >= 2 &&
			countAudit(last, statefulRestartRoute) >= 2 &&
			countAudit(last, daemonRestartRoute) >= 2 &&
			countAudit(last, podRestartRoute) >= 4 &&
			countAudit(last, cronJobSuspendRoute) >= 1 &&
			countAudit(last, cronJobResumeRoute) >= 1 {
			return last
		}
		time.Sleep(250 * time.Millisecond)
	}
	return last
}
