package provider

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig returns the Kubernetes client configuration.
// It first tries in-cluster config, then falls back to kubeconfig.
func GetConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	return kubeConfig.ClientConfig()
}

// NewClientset creates a new Kubernetes clientset.
// It first tries in-cluster config, then falls back to kubeconfig.
func NewClientset() (*kubernetes.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
