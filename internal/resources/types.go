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

type NodeSummary struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	CPUAllocatable  string    `json:"cpu_allocatable"`
	MemAllocatable  string    `json:"memory_allocatable"`
	PodsAllocatable string    `json:"pods_allocatable"`
	CreatedAt       time.Time `json:"created_at"`
}

type ContainerSummary struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restart_count"`
	State        string `json:"state"`
	Reason       string `json:"reason"`
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
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	OwnerRefs   []OwnerReference  `json:"owner_refs"`
	Conditions  []Condition       `json:"conditions"`
}

type OwnerReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
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

type DaemonSetSummary struct {
	Namespace       string            `json:"namespace"`
	Name            string            `json:"name"`
	DesiredNumber   int32             `json:"desired_number"`
	CurrentNumber   int32             `json:"current_number"`
	ReadyNumber     int32             `json:"ready_number"`
	AvailableNumber int32             `json:"available_number"`
	Labels          map[string]string `json:"labels"`
	Images          []string          `json:"images"`
	CreatedAt       time.Time         `json:"created_at"`
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

type CronJobSummary struct {
	Namespace        string    `json:"namespace"`
	Name             string    `json:"name"`
	Schedule         string    `json:"schedule"`
	Suspend          bool      `json:"suspend"`
	Active           int       `json:"active"`
	LastScheduleTime time.Time `json:"last_schedule_time"`
	CreatedAt        time.Time `json:"created_at"`
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

type IngressSummary struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	ClassName string    `json:"class_name"`
	Hosts     []string  `json:"hosts"`
	Addresses []string  `json:"addresses"`
	TLS       bool      `json:"tls"`
	CreatedAt time.Time `json:"created_at"`
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
