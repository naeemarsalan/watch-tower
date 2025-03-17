package k8s

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetK8sClient creates a Kubernetes client using either in-cluster config or kubeconfig
func GetK8sClient() (*kubernetes.Clientset, *dynamic.DynamicClient, error) {
	var config *rest.Config
	var err error

	// Check if running inside Kubernetes (service account exists)
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		// Running inside a Kubernetes pod
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create in-cluster Kubernetes config: %v", err)
		}
		fmt.Println("✅ Using in-cluster Kubernetes configuration")
	} else {
		// Running locally - use Kubeconfig
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.ExpandEnv("$HOME/.kube/config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load kubeconfig: %v", err)
		}
		fmt.Println("✅ Using local Kubeconfig")
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	// Create Kubernetes dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes dynamic client: %v", err)
	}

	return clientset, dynamicClient, nil
}

// GetAutomationController fetches the AutomationController CRD inside the given namespace
func GetAutomationController(dynamicClient dynamic.Interface, namespace string) error {
	// Ensure namespace is set
	if namespace == "" {
		return fmt.Errorf("AAP_NAMESPACE environment variable is not set")
	}

	// Define the GVR (GroupVersionResource) for AutomationController
	gvr := schema.GroupVersionResource{
		Group:    "automationcontroller.ansible.com", // Replace with the actual API group
		Version:  "v1beta1",                          // Replace with the actual version
		Resource: "automationcontrollers",
	}

	// Get list of AutomationControllers in the namespace
	automationControllers, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list automationcontroller CRD: %v", err)
	}

	// Check if any automationcontrollers exist
	if len(automationControllers.Items) == 0 {
		fmt.Printf("❌ No automationcontroller found in namespace %s\n", namespace)
		return nil
	}

	// Print details of each AutomationController
	fmt.Printf("✅ Found %d automationcontroller CRDs in namespace %s:\n", len(automationControllers.Items), namespace)
	for _, controller := range automationControllers.Items {
		fmt.Printf("  - Name: %s\n", controller.GetName())
		fmt.Printf("  - UID: %s\n", controller.GetUID())
		fmt.Printf("  - CreationTimestamp: %s\n", controller.GetCreationTimestamp())
	}

	return nil
}
