package cluster

import (
	"sync"

	"ops-platform/internal/model"
)

type Registry struct {
	mu       sync.RWMutex
	clusters map[string]model.Cluster
}

func NewRegistry(defaultCluster model.Cluster) *Registry {
	return &Registry{
		clusters: map[string]model.Cluster{
			defaultCluster.Name: defaultCluster,
		},
	}
}

func (r *Registry) Get(name string) (model.Cluster, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cluster, ok := r.clusters[name]
	return cluster, ok
}
