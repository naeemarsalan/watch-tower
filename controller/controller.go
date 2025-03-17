package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"watch-tower/dbconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
)

// StartWatchLoop continuously monitors the AutomationController and updates replicas
func StartWatchLoop(dynamicClient dynamic.Interface, dbCredentials *dbconfig.DatabaseCredentials, namespace string) {
	fmt.Println("🚀 Starting Watch Loop for AutomationController...")

	// Define GVR (GroupVersionResource) for AutomationController CRD
	gvr := schema.GroupVersionResource{
		Group:    "automationcontroller.ansible.com", // Correct API Group
		Version:  "v1beta1",                          // Correct API Version
		Resource: "automationcontrollers",            // Correct Resource Name
	}

	for {
		fmt.Println("\n🔄 Checking AutomationController and Database Role...")

		// Get the list of AutomationControllers in the namespace
		automationControllers, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("❌ Failed to list AutomationController CRDs:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Connect to the database and check its role
		conn, err := dbconfig.ConnectToDB(dbCredentials)
		if err != nil {
			fmt.Println("❌ Database connection failed:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		defer conn.Close(context.Background())

		dbRole, err := dbconfig.CheckDBRole(conn)
		if err != nil {
			fmt.Println("❌ Failed to check database role:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("🔍 Database Role:", dbRole)

		// Loop through all found AutomationControllers
		for _, controller := range automationControllers.Items {
			name := controller.GetName()
			annotations := controller.GetAnnotations()

			// Check if annotation exists
			replicasStr, exists := annotations["watch-tower/replicas"]
			if !exists {
				fmt.Printf("⚠️  Skipping %s (No watch-tower/replicas annotation)\n", name)
				continue
			}

			// Convert annotation value from string to integer
			replicas, err := strconv.Atoi(replicasStr)
			if err != nil || replicas < 0 {
				fmt.Printf("⚠️  Invalid watch-tower/replicas value in %s: %s\n", name, replicasStr)
				continue
			}

			// Determine desired replica count
			var desiredReplicas int
			if dbRole == "Primary" {
				desiredReplicas = replicas
			} else {
				desiredReplicas = 0
			}

			// Get current spec.replicas value
			currentReplicas, err := getCurrentReplicas(&controller)
			if err != nil {
				fmt.Printf("❌ Failed to get current replicas for %s: %v\n", name, err)
				continue
			}

			// Only patch if the replicas value is different
			if currentReplicas == desiredReplicas {
				fmt.Printf("✅ No update needed for %s (replicas already set to %d)\n", name, desiredReplicas)
				continue
			}

			// Patch AutomationController's spec.replicas
			err = patchAutomationControllerReplicas(dynamicClient, namespace, name, desiredReplicas)
			if err != nil {
				fmt.Printf("❌ Failed to patch %s: %v\n", name, err)
			} else {
				fmt.Printf("✅ Successfully patched %s: spec.replicas = %d\n", name, desiredReplicas)
			}
		}

		// Sleep before checking again
		time.Sleep(30 * time.Second) // Adjust as needed
	}
}

// getCurrentReplicas retrieves the current spec.replicas value from an AutomationController CRD
func getCurrentReplicas(controller *unstructured.Unstructured) (int, error) {
	replicas, found, err := unstructured.NestedInt64(controller.Object, "spec", "replicas")
	if err != nil || !found {
		return 0, fmt.Errorf("spec.replicas field not found")
	}
	return int(replicas), nil
}

// patchAutomationControllerReplicas updates the spec.replicas field of an AutomationController CRD
func patchAutomationControllerReplicas(dynamicClient dynamic.Interface, namespace, name string, replicas int) error {
	// Define GVR
	gvr := schema.GroupVersionResource{
		Group:    "automationcontroller.ansible.com",
		Version:  "v1beta1",
		Resource: "automationcontrollers",
	}

	// Create patch payload
	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("failed to marshal patch data: %v", err)
	}

	// Apply patch
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(
		context.TODO(),
		name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	return err
}

