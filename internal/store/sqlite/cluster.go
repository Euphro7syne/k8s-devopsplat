package sqlite

import (
	"context"
	"fmt"

	"ops-platform/internal/model"
)

func (s *Store) ListClusters(ctx context.Context) ([]model.Cluster, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, kubeconfig_ref, in_cluster, status, created_at
FROM clusters
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	defer rows.Close()

	clusters := make([]model.Cluster, 0)
	for rows.Next() {
		var cluster model.Cluster
		var inCluster int
		if err := rows.Scan(
			&cluster.ID,
			&cluster.Name,
			&cluster.KubeconfigRef,
			&inCluster,
			&cluster.Status,
			&cluster.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cluster: %w", err)
		}
		cluster.InCluster = inCluster == 1
		clusters = append(clusters, cluster)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clusters: %w", err)
	}
	return clusters, nil
}
