package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"ops-platform/internal/config"
)

func RESTConfig(cfg config.KubernetesConfig) (*rest.Config, error) {
	mode := strings.ToLower(cfg.Mode)
	switch mode {
	case "in-cluster", "in_cluster":
		return rest.InClusterConfig()
	case "kubeconfig":
		return clientcmd.BuildConfigFromFlags("", expandHome(cfg.Kubeconfig))
	case "auto", "":
		if cfg.Kubeconfig != "" {
			if restCfg, err := clientcmd.BuildConfigFromFlags("", expandHome(cfg.Kubeconfig)); err == nil {
				return restCfg, nil
			}
		}
		return rest.InClusterConfig()
	default:
		return nil, fmt.Errorf("unsupported kubernetes.mode %q", cfg.Mode)
	}
}

func NewClientset(cfg config.KubernetesConfig) (*kubernetes.Clientset, error) {
	restCfg, err := RESTConfig(cfg)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restCfg)
}

func expandHome(path string) string {
	if path == "" || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, ".kube", "config")
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
