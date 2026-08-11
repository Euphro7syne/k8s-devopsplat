package workload

import (
	"time"

	"ops-platform/internal/resources"
)

type ScaleDeploymentRequest struct {
	Replicas int32 `json:"replicas" binding:"min=0,max=100"`
}

type OperationResult struct {
	Kind       string                       `json:"kind"`
	Namespace  string                       `json:"namespace"`
	Name       string                       `json:"name"`
	Operation  string                       `json:"operation"`
	UpdatedAt  time.Time                    `json:"updated_at"`
	Deployment *resources.DeploymentSummary `json:"deployment,omitempty"`
}
