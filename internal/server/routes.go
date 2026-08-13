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
	mfaVerification := api.Group("")
	mfaVerification.Use(s.auditMiddleware())
	authHandler.RegisterMFAVerification(mfaVerification)

	protected := api.Group("")
	protected.Use(s.authMiddleware(), s.auditMiddleware())
	authHandler.RegisterProtected(protected)

	readAccess := s.requireRoles("viewer", "operator", "configadmin", "auditor", "admin")
	writeAccess := s.requireRoles("operator", "admin")
	auditAccess := s.requireRoles("auditor", "admin")
	adminAccess := s.requireRoles("admin")

	protected.GET("/clusters", readAccess, s.clusters)
	authHandler.RegisterAdmin(protected, adminAccess)
	resources.NewHandler(resources.NewService(s.kubeClient, "in-cluster")).Register(protected, readAccess, writeAccess)
	logHandler := logquery.NewHandler(logquery.NewService(s.kubeClient), s.cfg.Server.CORS.AllowedOrigins)
	logHandler.Register(protected, readAccess)
	websocketRoutes := s.engine.Group("/ws/v1")
	websocketRoutes.Use(s.websocketAuthMiddleware())
	logHandler.RegisterWebSocket(websocketRoutes, readAccess)
	workload.NewHandler(workload.NewService(s.kubeClient)).Register(protected, writeAccess)
	audit.NewHandler(audit.NewService(s.store)).Register(protected, auditAccess)
}
