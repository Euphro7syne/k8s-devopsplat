package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"ops-platform/internal/auth"
	"ops-platform/internal/resources"
)

func TestSecretRoutesEnforceRoleAndConfirmationBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "demo"},
		Data:       map[string][]byte{"password": []byte("must-not-leak-by-default")},
	})
	server := &Server{}
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		auth.WithPrincipal(c, auth.Principal{UserID: 1, Roles: []string{c.GetHeader("X-Test-Role")}})
		c.Next()
	})
	resources.NewHandler(resources.NewService(client, "test")).Register(
		engine.Group("/api/v1"),
		server.requireRoles("viewer", "operator", "configadmin", "auditor", "admin"),
		server.requireRoles("operator", "admin"),
		server.requireRoles("configadmin", "admin"),
		server.requireRoles("admin"),
	)

	assertRouteStatus(t, engine, http.MethodGet, "/api/v1/namespaces/demo/secrets", "viewer", http.StatusForbidden, false)
	assertRouteStatus(t, engine, http.MethodGet, "/api/v1/namespaces/demo/secrets", "operator", http.StatusForbidden, false)
	assertRouteStatus(t, engine, http.MethodGet, "/api/v1/namespaces/demo/secrets", "auditor", http.StatusForbidden, false)
	assertRouteStatus(t, engine, http.MethodGet, "/api/v1/namespaces/demo/secrets", "configadmin", http.StatusOK, false)
	assertRouteStatus(t, engine, http.MethodGet, "/api/v1/namespaces/demo/secrets/app-secret", "configadmin", http.StatusOK, false)
	assertRouteStatus(t, engine, http.MethodPost, "/api/v1/namespaces/demo/secrets/app-secret/values/password?confirm=true", "configadmin", http.StatusForbidden, false)
	assertRouteStatus(t, engine, http.MethodPost, "/api/v1/namespaces/demo/secrets/app-secret/values/password", "admin", http.StatusBadRequest, false)
	recorder := assertRouteStatus(t, engine, http.MethodPost, "/api/v1/namespaces/demo/secrets/app-secret/values/password?confirm=true", "admin", http.StatusOK, true)
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("Secret plaintext response must disable caching: %#v", recorder.Header())
	}
}

func assertRouteStatus(t *testing.T, handler http.Handler, method, path, role string, expectedStatus int, expectValue bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Test-Role", role)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != expectedStatus {
		t.Fatalf("%s %s as %s returned HTTP %d, expected %d: %s", method, path, role, recorder.Code, expectedStatus, recorder.Body.String())
	}
	containsValue := strings.Contains(recorder.Body.String(), "must-not-leak-by-default")
	if containsValue != expectValue {
		t.Fatalf("%s %s as %s plaintext presence=%t, expected %t: %s", method, path, role, containsValue, expectValue, recorder.Body.String())
	}
	return recorder
}
