package model

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	MFASecret    string    `json:"-"`
	Provider     string    `json:"provider"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	Roles        []string  `json:"roles"`
}

type Role struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Cluster struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	KubeconfigRef string    `json:"kubeconfig_ref"`
	InCluster     bool      `json:"in_cluster"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           int64     `json:"id"`
	UserID       *int64    `json:"user_id,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Namespace    string    `json:"namespace"`
	RequestBody  string    `json:"request_body"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}
