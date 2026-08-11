package server

import (
	"ops-platform/internal/audit"
	"ops-platform/internal/auth"
	"ops-platform/internal/logquery"
	"ops-platform/internal/resources"
	"ops-platform/internal/workload"
)

func (s *Server) registerRoutes() {
	s.engine.Use(
		s.recoveryMiddleware(),
		s.requestIDMiddleware(),
		s.corsMiddleware(),
	)

	api := s.engine.Group("/api/v1")
	api.GET("/healthz", s.healthz)

	authHandler := auth.NewHandler(s.authService)
	authHandler.RegisterPublic(api)

	protected := api.Group("")
	protected.Use(s.authMiddleware(), s.auditMiddleware())
	authHandler.RegisterProtected(protected)

	readAccess := s.requireRoles("viewer", "operator", "configadmin", "auditor", "admin")
	writeAccess := s.requireRoles("operator", "admin")
	auditAccess := s.requireRoles("auditor", "admin")

	protected.GET("/clusters", readAccess, s.clusters)
	resources.NewHandler(resources.NewService(s.kubeClient, "in-cluster")).Register(protected, readAccess, writeAccess)
	logquery.NewHandler(logquery.NewService(s.kubeClient)).Register(protected, readAccess)
	workload.NewHandler(workload.NewService(s.kubeClient)).Register(protected, writeAccess)
	audit.NewHandler(audit.NewService(s.store)).Register(protected, auditAccess)
}
