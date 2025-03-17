package controller

import (
	"context"
	"fmt"
	"net"
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

// StartWatchLoop continuously monitors AutomationController and updates replicas
func StartWatchLoop(dynamicClient dynamic.Interface, dbCredentials *dbconfig.DatabaseCredentials, namespace string) {
	fmt.Println("🚀 Starting Watch Loop for AutomationController...")

	// Define GVR (GroupVersionResource) for AutomationController CRD
	gvr := schema.GroupVersionResource{
		Group:    "automationcontroller.ansible.com",
		Version:  "v1beta1",
		Resource: "automationcontrollers",
	}

	for {
		fmt.Println("\n🔄 Checking AutomationController and Database Role...")

		// ✅ Step 1: Quick Port Check (Fast Fail if DB is Down)
		dbAddr := fmt.Sprintf("%s:%s", dbCredentials.Host, dbCredentials.Port)
		if !isPostgresReachable(dbAddr) {
			fmt.Println("❌ Database port is unreachable! Scaling down AutomationController immediately...")
			scaleDownAllControllers(dynamicClient, namespace, gvr)
			fmt.Println("🔄 Retrying in 10 seconds...")
			time.Sleep(10 * time.Second) // ⏳ Fast retry when DB is down
			continue
		}

		// ✅ Step 2: Connect to PostgreSQL (3s Timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		conn, err := dbconfig.ConnectToDB(dbCredentials)
		if err != nil {
			fmt.Println("❌ Database connection failed! Scaling down AutomationController...")
			scaleDownAllControllers(dynamicClient, namespace, gvr)
			fmt.Println("🔄 Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}
		defer conn.Close(ctx)

		// ✅ Step 3: Check DB Role (Primary or Standby)
		dbRole, err := dbconfig.CheckDBRole(conn)
		if err != nil {
			fmt.Println("❌ Failed to check database role. Scaling down AutomationController...")
			scaleDownAllControllers(dynamicClient, namespace, gvr)
			fmt.Println("🔄 Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("🔍 Database Role:", dbRole)

		// ✅ Step 4: Get list of AutomationControllers
		automationControllers, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("❌ Failed to list AutomationController CRDs:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// ✅ Step 5: Process each AutomationController
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

			// ✅ Step 6: Get current spec.replicas value
			currentReplicas, err := getCurrentReplicas(&controller)
			if err != nil {
				fmt.Printf("❌ Failed to get current replicas for %s: %v\n", name, err)
				continue
			}

			// ✅ Step 7: Only patch if replicas value is different
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

// ✅ Quick TCP Check for DB Port
func isPostgresReachable(address string) bool {
	timeout := 2 * time.Second // Fast fail
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func getCurrentReplicas(controller *unstructured.Unstructured) (int, error) {
	// Extract the "spec.replicas" field
	replicasField, found, err := unstructured.NestedFieldCopy(controller.Object, "spec", "replicas")
	if err != nil || !found {
		return 0, fmt.Errorf("spec.replicas field not found")
	}

	// Convert value to int safely
	switch v := replicasField.(type) {
	case int64:
		return int(v), nil
	case int:
		return v, nil
	case float64:
		return int(v), nil // Convert float64 to int (Go treats JSON numbers as float64)
	default:
		return 0, fmt.Errorf("unexpected type for spec.replicas: %T", v)
	}
}

// ✅ Scale Down All Controllers if DB is Down
func scaleDownAllControllers(dynamicClient dynamic.Interface, namespace string, gvr schema.GroupVersionResource) {
	controllers, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("❌ Failed to list AutomationControllers for scaling down:", err)
		return
	}

	for _, controller := range controllers.Items {
		name := controller.GetName()
		err := patchAutomationControllerReplicas(dynamicClient, namespace, name, 0)
		if err != nil {
			fmt.Printf("❌ Failed to scale down %s: %v\n", name, err)
		} else {
			fmt.Printf("⚠️  Scaled down %s to 0 due to DB failure\n", name)
		}
	}
}

// ✅ Patch AutomationController's `spec.replicas`
func patchAutomationControllerReplicas(dynamicClient dynamic.Interface, namespace, name string, replicas int) error {
	gvr := schema.GroupVersionResource{
		Group:    "automationcontroller.ansible.com",
		Version:  "v1beta1",
		Resource: "automationcontrollers",
	}

	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("failed to marshal patch data: %v", err)
	}

	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Patch(
		context.TODO(),
		name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	return err
}
