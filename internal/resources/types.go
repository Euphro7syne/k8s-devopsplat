package resources

import "time"

type ListOptions struct {
	Page     int
	PageSize int
}

type ClusterOverview struct {
	Cluster          string        `json:"cluster"`
	NodeCount        int           `json:"node_count"`
	ReadyNodeCount   int           `json:"ready_node_count"`
	NamespaceCount   int           `json:"namespace_count"`
	PodCount         int           `json:"pod_count"`
	AbnormalPodCount int           `json:"abnormal_pod_count"`
	Nodes            []NodeSummary `json:"nodes"`
}

type NamespaceSummary struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ResourceReference struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type DeclaredResources struct {
	CPU              string `json:"cpu"`
	Memory           string `json:"memory"`
	EphemeralStorage string `json:"ephemeral_storage"`
	Pods             int64  `json:"pods"`
}

type ResourceAllocation struct {
	Requests             DeclaredResources `json:"requests"`
	Limits               DeclaredResources `json:"limits"`
	CPURequestPercent    float64           `json:"cpu_request_percent"`
	MemoryRequestPercent float64           `json:"memory_request_percent"`
	PodPercent           float64           `json:"pod_percent"`
}

type NamespaceResourceCounts struct {
	Pods                   int `json:"pods"`
	ReadyPods              int `json:"ready_pods"`
	AbnormalPods           int `json:"abnormal_pods"`
	Deployments            int `json:"deployments"`
	StatefulSets           int `json:"statefulsets"`
	DaemonSets             int `json:"daemonsets"`
	ReplicaSets            int `json:"replicasets"`
	Jobs                   int `json:"jobs"`
	CronJobs               int `json:"cronjobs"`
	Services               int `json:"services"`
	Ingresses              int `json:"ingresses"`
	PersistentVolumeClaims int `json:"persistent_volume_claims"`
	ConfigMaps             int `json:"configmaps"`
}

type NamespaceDetail struct {
	NamespaceSummary
	Labels                 map[string]string       `json:"labels"`
	Finalizers             []string                `json:"finalizers"`
	Conditions             []Condition             `json:"conditions"`
	Counts                 NamespaceResourceCounts `json:"counts"`
	Allocated              ResourceAllocation      `json:"allocated"`
	Pods                   []PodSummary            `json:"pods"`
	Workloads              []ResourceReference     `json:"workloads"`
	Services               []ServiceSummary        `json:"services"`
	Ingresses              []IngressSummary        `json:"ingresses"`
	PersistentVolumeClaims []PVCSummary            `json:"persistent_volume_claims"`
	Events                 []EventSummary          `json:"events"`
}

type NodeSummary struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	CPUAllocatable  string    `json:"cpu_allocatable"`
	MemAllocatable  string    `json:"memory_allocatable"`
	PodsAllocatable string    `json:"pods_allocatable"`
	CreatedAt       time.Time `json:"created_at"`
}

type NodeAddressDetail struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type NodeTaintDetail struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Effect    string    `json:"effect"`
	TimeAdded time.Time `json:"time_added"`
}

type NodeConditionDetail struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
	LastHeartbeatTime  time.Time `json:"last_heartbeat_time"`
	LastTransitionTime time.Time `json:"last_transition_time"`
}

type NodeSystemInfo struct {
	MachineID               string `json:"machine_id"`
	SystemUUID              string `json:"system_uuid"`
	BootID                  string `json:"boot_id"`
	KernelVersion           string `json:"kernel_version"`
	OSImage                 string `json:"os_image"`
	ContainerRuntimeVersion string `json:"container_runtime_version"`
	KubeletVersion          string `json:"kubelet_version"`
	KubeProxyVersion        string `json:"kube_proxy_version"`
	OperatingSystem         string `json:"operating_system"`
	Architecture            string `json:"architecture"`
}

type NodeDetail struct {
	NodeSummary
	Roles         []string              `json:"roles"`
	Unschedulable bool                  `json:"unschedulable"`
	PodCIDRs      []string              `json:"pod_cidrs"`
	Labels        map[string]string     `json:"labels"`
	Addresses     []NodeAddressDetail   `json:"addresses"`
	Taints        []NodeTaintDetail     `json:"taints"`
	Conditions    []NodeConditionDetail `json:"conditions"`
	SystemInfo    NodeSystemInfo        `json:"system_info"`
	Capacity      DeclaredResources     `json:"capacity"`
	Allocatable   DeclaredResources     `json:"allocatable"`
	Allocated     ResourceAllocation    `json:"allocated"`
	Pods          []PodSummary          `json:"pods"`
	Workloads     []ResourceReference   `json:"workloads"`
	Events        []EventSummary        `json:"events"`
}

type ContainerSummary struct {
	Name           string    `json:"name"`
	Image          string    `json:"image"`
	Ready          bool      `json:"ready"`
	RestartCount   int32     `json:"restart_count"`
	State          string    `json:"state"`
	Reason         string    `json:"reason"`
	ExitCode       int32     `json:"exit_code"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	LastState      string    `json:"last_state"`
	LastReason     string    `json:"last_reason"`
	LastExitCode   int32     `json:"last_exit_code"`
	LastFinishedAt time.Time `json:"last_finished_at"`
}

type PodSummary struct {
	Namespace    string             `json:"namespace"`
	Name         string             `json:"name"`
	Phase        string             `json:"phase"`
	Status       string             `json:"status"`
	NodeName     string             `json:"node_name"`
	Ready        bool               `json:"ready"`
	RestartCount int32              `json:"restart_count"`
	Containers   []ContainerSummary `json:"containers"`
	CreatedAt    time.Time          `json:"created_at"`
}

type PodDetail struct {
	PodSummary
	PodIP           string             `json:"pod_ip"`
	HostIP          string             `json:"host_ip"`
	QoSClass        string             `json:"qos_class"`
	ServiceAccount  string             `json:"service_account"`
	RestartPolicy   string             `json:"restart_policy"`
	StartTime       time.Time          `json:"start_time"`
	Labels          map[string]string  `json:"labels"`
	Annotations     map[string]string  `json:"annotations"`
	OwnerRefs       []OwnerReference   `json:"owner_refs"`
	ControllerChain []OwnerReference   `json:"controller_chain"`
	InitContainers  []ContainerSummary `json:"init_containers"`
	Conditions      []Condition        `json:"conditions"`
}

type OwnerReference struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Controller bool   `json:"controller"`
}

type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type EventSummary struct {
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	Namespace      string    `json:"namespace"`
	InvolvedKind   string    `json:"involved_kind"`
	InvolvedName   string    `json:"involved_name"`
	Source         string    `json:"source"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
}

type DeploymentSummary struct {
	Namespace           string            `json:"namespace"`
	Name                string            `json:"name"`
	Replicas            int32             `json:"replicas"`
	ReadyReplicas       int32             `json:"ready_replicas"`
	UpdatedReplicas     int32             `json:"updated_replicas"`
	AvailableReplicas   int32             `json:"available_replicas"`
	UnavailableReplicas int32             `json:"unavailable_replicas"`
	Labels              map[string]string `json:"labels"`
	Images              []string          `json:"images"`
	CreatedAt           time.Time         `json:"created_at"`
}

type WorkloadCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
	LastUpdateTime     time.Time `json:"last_update_time"`
	LastTransitionTime time.Time `json:"last_transition_time"`
}

type ReplicaSetDetail struct {
	WorkloadSummary
	Revision             string              `json:"revision"`
	CurrentReplicas      int32               `json:"current_replicas"`
	FullyLabeledReplicas int32               `json:"fully_labeled_replicas"`
	ObservedGeneration   int64               `json:"observed_generation"`
	MinReadySeconds      int32               `json:"min_ready_seconds"`
	Selector             map[string]string   `json:"selector"`
	Owner                *OwnerReference     `json:"owner"`
	Conditions           []WorkloadCondition `json:"conditions"`
	Pods                 []PodSummary        `json:"pods"`
	Events               []EventSummary      `json:"events"`
}

type DeploymentDetail struct {
	DeploymentSummary
	Generation              int64               `json:"generation"`
	ObservedGeneration      int64               `json:"observed_generation"`
	Paused                  bool                `json:"paused"`
	Strategy                string              `json:"strategy"`
	MaxSurge                string              `json:"max_surge"`
	MaxUnavailable          string              `json:"max_unavailable"`
	MinReadySeconds         int32               `json:"min_ready_seconds"`
	ProgressDeadlineSeconds int32               `json:"progress_deadline_seconds"`
	RevisionHistoryLimit    int32               `json:"revision_history_limit"`
	Selector                map[string]string   `json:"selector"`
	Conditions              []WorkloadCondition `json:"conditions"`
	ReplicaSets             []ReplicaSetDetail  `json:"replica_sets"`
	Pods                    []PodSummary        `json:"pods"`
	Events                  []EventSummary      `json:"events"`
}

type WorkloadSummary struct {
	Kind                string            `json:"kind"`
	Namespace           string            `json:"namespace"`
	Name                string            `json:"name"`
	Replicas            int32             `json:"replicas"`
	ReadyReplicas       int32             `json:"ready_replicas"`
	AvailableReplicas   int32             `json:"available_replicas"`
	UnavailableReplicas int32             `json:"unavailable_replicas"`
	Labels              map[string]string `json:"labels"`
	Images              []string          `json:"images"`
	CreatedAt           time.Time         `json:"created_at"`
}

type StatefulSetDetail struct {
	WorkloadSummary
	ServiceName         string                   `json:"service_name"`
	PodManagementPolicy string                   `json:"pod_management_policy"`
	UpdateStrategy      string                   `json:"update_strategy"`
	CurrentRevision     string                   `json:"current_revision"`
	UpdateRevision      string                   `json:"update_revision"`
	CurrentReplicas     int32                    `json:"current_replicas"`
	UpdatedReplicas     int32                    `json:"updated_replicas"`
	Selector            map[string]string        `json:"selector"`
	VolumeClaims        []StatefulSetVolumeClaim `json:"volume_claims"`
}

type StatefulSetVolumeClaim struct {
	Name             string   `json:"name"`
	StorageClass     string   `json:"storage_class"`
	RequestedStorage string   `json:"requested_storage"`
	AccessModes      []string `json:"access_modes"`
}

type DaemonSetSummary struct {
	Namespace          string            `json:"namespace"`
	Name               string            `json:"name"`
	DesiredNumber      int32             `json:"desired_number"`
	CurrentNumber      int32             `json:"current_number"`
	ReadyNumber        int32             `json:"ready_number"`
	UpdatedNumber      int32             `json:"updated_number"`
	AvailableNumber    int32             `json:"available_number"`
	UnavailableNumber  int32             `json:"unavailable_number"`
	MisscheduledNumber int32             `json:"misscheduled_number"`
	Labels             map[string]string `json:"labels"`
	Images             []string          `json:"images"`
	CreatedAt          time.Time         `json:"created_at"`
}

type DaemonSetDetail struct {
	DaemonSetSummary
	UpdateStrategy string            `json:"update_strategy"`
	Selector       map[string]string `json:"selector"`
	NodeSelector   map[string]string `json:"node_selector"`
	Tolerations    []string          `json:"tolerations"`
}

type JobSummary struct {
	Namespace      string    `json:"namespace"`
	Name           string    `json:"name"`
	Completions    int32     `json:"completions"`
	Succeeded      int32     `json:"succeeded"`
	Failed         int32     `json:"failed"`
	Active         int32     `json:"active"`
	StartTime      time.Time `json:"start_time"`
	CompletionTime time.Time `json:"completion_time"`
	CreatedAt      time.Time `json:"created_at"`
}

type JobDetail struct {
	JobSummary
	Parallelism             int32               `json:"parallelism"`
	BackoffLimit            int32               `json:"backoff_limit"`
	ActiveDeadlineSeconds   int64               `json:"active_deadline_seconds"`
	TTLSecondsAfterFinished int32               `json:"ttl_seconds_after_finished"`
	CompletionMode          string              `json:"completion_mode"`
	Suspend                 bool                `json:"suspend"`
	ManualSelector          bool                `json:"manual_selector"`
	Selector                map[string]string   `json:"selector"`
	Owner                   *OwnerReference     `json:"owner"`
	Images                  []string            `json:"images"`
	Conditions              []WorkloadCondition `json:"conditions"`
	Pods                    []PodSummary        `json:"pods"`
	Events                  []EventSummary      `json:"events"`
}

type CronJobSummary struct {
	Namespace        string    `json:"namespace"`
	Name             string    `json:"name"`
	Schedule         string    `json:"schedule"`
	Suspend          bool      `json:"suspend"`
	Active           int       `json:"active"`
	LastScheduleTime time.Time `json:"last_schedule_time"`
	CreatedAt        time.Time `json:"created_at"`
}

type JobTemplatePolicy struct {
	Parallelism             int32    `json:"parallelism"`
	Completions             int32    `json:"completions"`
	BackoffLimit            int32    `json:"backoff_limit"`
	ActiveDeadlineSeconds   int64    `json:"active_deadline_seconds"`
	TTLSecondsAfterFinished int32    `json:"ttl_seconds_after_finished"`
	CompletionMode          string   `json:"completion_mode"`
	Suspend                 bool     `json:"suspend"`
	RestartPolicy           string   `json:"restart_policy"`
	Images                  []string `json:"images"`
}

type CronJobDetail struct {
	CronJobSummary
	TimeZone                   string            `json:"time_zone"`
	ConcurrencyPolicy          string            `json:"concurrency_policy"`
	StartingDeadlineSeconds    int64             `json:"starting_deadline_seconds"`
	SuccessfulJobsHistoryLimit int32             `json:"successful_jobs_history_limit"`
	FailedJobsHistoryLimit     int32             `json:"failed_jobs_history_limit"`
	LastSuccessfulTime         time.Time         `json:"last_successful_time"`
	JobTemplate                JobTemplatePolicy `json:"job_template"`
	Jobs                       []JobDetail       `json:"jobs"`
	Events                     []EventSummary    `json:"events"`
}

type ServiceSummary struct {
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"cluster_ip"`
	ExternalIP string            `json:"external_ip"`
	Ports      []string          `json:"ports"`
	Selector   map[string]string `json:"selector"`
	CreatedAt  time.Time         `json:"created_at"`
}

type ServicePortDetail struct {
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"target_port"`
	NodePort    int32  `json:"node_port"`
	AppProtocol string `json:"app_protocol"`
}

type ServiceEndpoint struct {
	Source      string   `json:"source"`
	SourceName  string   `json:"source_name"`
	Addresses   []string `json:"addresses"`
	Ready       bool     `json:"ready"`
	Serving     bool     `json:"serving"`
	Terminating bool     `json:"terminating"`
	Hostname    string   `json:"hostname"`
	NodeName    string   `json:"node_name"`
	Zone        string   `json:"zone"`
	TargetKind  string   `json:"target_kind"`
	TargetName  string   `json:"target_name"`
	Ports       []string `json:"ports"`
}

type ServiceDetail struct {
	ServiceSummary
	ClusterIPs               []string            `json:"cluster_ips"`
	ExternalName             string              `json:"external_name"`
	IPFamilies               []string            `json:"ip_families"`
	IPFamilyPolicy           string              `json:"ip_family_policy"`
	SessionAffinity          string              `json:"session_affinity"`
	ExternalTrafficPolicy    string              `json:"external_traffic_policy"`
	InternalTrafficPolicy    string              `json:"internal_traffic_policy"`
	PublishNotReadyAddresses bool                `json:"publish_not_ready_addresses"`
	LoadBalancerSourceRanges []string            `json:"load_balancer_source_ranges"`
	PortDetails              []ServicePortDetail `json:"port_details"`
	EndpointSource           string              `json:"endpoint_source"`
	Endpoints                []ServiceEndpoint   `json:"endpoints"`
	Pods                     []PodSummary        `json:"pods"`
	Events                   []EventSummary      `json:"events"`
}

type IngressSummary struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	ClassName string    `json:"class_name"`
	Hosts     []string  `json:"hosts"`
	Addresses []string  `json:"addresses"`
	TLS       bool      `json:"tls"`
	CreatedAt time.Time `json:"created_at"`
}

type IngressBackendDetail struct {
	Host             string `json:"host"`
	Path             string `json:"path"`
	PathType         string `json:"path_type"`
	IsDefault        bool   `json:"is_default"`
	BackendKind      string `json:"backend_kind"`
	BackendAPIGroup  string `json:"backend_api_group"`
	BackendName      string `json:"backend_name"`
	BackendPort      string `json:"backend_port"`
	ServiceFound     bool   `json:"service_found"`
	ServicePortFound bool   `json:"service_port_found"`
}

type IngressTLSDetail struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secret_name"`
}

type IngressDetail struct {
	IngressSummary
	Backends   []IngressBackendDetail `json:"backends"`
	TLSDetails []IngressTLSDetail     `json:"tls_details"`
	Services   []ServiceDetail        `json:"services"`
	Events     []EventSummary         `json:"events"`
}

type ConfigMapSummary struct {
	Namespace       string    `json:"namespace"`
	Name            string    `json:"name"`
	KeyCount        int       `json:"key_count"`
	BinaryDataCount int       `json:"binary_data_count"`
	Keys            []string  `json:"keys"`
	CreatedAt       time.Time `json:"created_at"`
}

type PVCSummary struct {
	Namespace    string    `json:"namespace"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	StorageClass string    `json:"storage_class"`
	VolumeName   string    `json:"volume_name"`
	Requested    string    `json:"requested"`
	Capacity     string    `json:"capacity"`
	AccessModes  []string  `json:"access_modes"`
	CreatedAt    time.Time `json:"created_at"`
}

type PVSummary struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	StorageClass   string    `json:"storage_class"`
	Capacity       string    `json:"capacity"`
	ClaimNamespace string    `json:"claim_namespace"`
	ClaimName      string    `json:"claim_name"`
	ReclaimPolicy  string    `json:"reclaim_policy"`
	AccessModes    []string  `json:"access_modes"`
	CreatedAt      time.Time `json:"created_at"`
}

type VolumeDataSource struct {
	APIGroup  string `json:"api_group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type VolumeMountDetail struct {
	PodNamespace  string `json:"pod_namespace"`
	PodName       string `json:"pod_name"`
	VolumeName    string `json:"volume_name"`
	ContainerType string `json:"container_type"`
	ContainerName string `json:"container_name"`
	MountPath     string `json:"mount_path"`
	DevicePath    string `json:"device_path"`
	SubPath       string `json:"sub_path"`
	ReadOnly      bool   `json:"read_only"`
}

type PVCDetail struct {
	PVCSummary
	VolumeMode          string              `json:"volume_mode"`
	Selector            map[string]string   `json:"selector"`
	SelectorExpressions []string            `json:"selector_expressions"`
	DataSource          *VolumeDataSource   `json:"data_source"`
	Conditions          []Condition         `json:"conditions"`
	PV                  *PVSummary          `json:"pv"`
	Pods                []PodSummary        `json:"pods"`
	Workloads           []OwnerReference    `json:"workloads"`
	Mounts              []VolumeMountDetail `json:"mounts"`
	Events              []EventSummary      `json:"events"`
}

type PVDetail struct {
	PVSummary
	VolumeMode       string              `json:"volume_mode"`
	MountOptions     []string            `json:"mount_options"`
	NodeAffinity     []string            `json:"node_affinity"`
	VolumeSourceType string              `json:"volume_source_type"`
	VolumeSourceInfo map[string]string   `json:"volume_source_info"`
	PVC              *PVCSummary         `json:"pvc"`
	Pods             []PodSummary        `json:"pods"`
	Workloads        []OwnerReference    `json:"workloads"`
	Mounts           []VolumeMountDetail `json:"mounts"`
	Events           []EventSummary      `json:"events"`
}

type StorageClassSummary struct {
	Name                 string    `json:"name"`
	Provisioner          string    `json:"provisioner"`
	ReclaimPolicy        string    `json:"reclaim_policy"`
	VolumeBindingMode    string    `json:"volume_binding_mode"`
	AllowVolumeExpansion bool      `json:"allow_volume_expansion"`
	CreatedAt            time.Time `json:"created_at"`
}

type ResourceYAMLUpdateRequest struct {
	Kind      string `json:"kind" binding:"required"`
	Namespace string `json:"namespace"`
	Name      string `json:"name" binding:"required"`
	YAML      string `json:"yaml" binding:"required"`
}

type ResourceYAMLUpdateResult struct {
	Kind      string    `json:"kind"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Operation string    `json:"operation"`
	UpdatedAt time.Time `json:"updated_at"`
	YAML      string    `json:"yaml"`
}
