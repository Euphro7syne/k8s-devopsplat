package workload

import (
	"time"

	"ops-platform/internal/resources"
)

type ScaleDeploymentRequest struct {
	Replicas int32 `json:"replicas" binding:"min=0,max=100"`
}

type ScaleStatefulSetRequest struct {
	Replicas int32 `json:"replicas" binding:"min=0,max=100"`
}

type OperationResult struct {
	Kind        string                       `json:"kind"`
	Namespace   string                       `json:"namespace"`
	Name        string                       `json:"name"`
	Operation   string                       `json:"operation"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	Deployment  *resources.DeploymentSummary `json:"deployment,omitempty"`
	StatefulSet *resources.WorkloadSummary   `json:"statefulset,omitempty"`
	DaemonSet   *resources.DaemonSetSummary  `json:"daemonset,omitempty"`
	CronJob     *resources.CronJobSummary    `json:"cronjob,omitempty"`
}
