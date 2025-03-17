package main

import (
	"fmt"
	"os"

	"watch-tower/controller"
	"watch-tower/dbconfig"
	"watch-tower/k8s"
)

func main() {
	// Get file path for DB credentials
	filePath := os.Getenv("DB_CREDENTIAL_PATH")
	if filePath == "" {
		fmt.Println("Error: DB_CREDENTIAL_PATH environment variable is not set.")
		return
	}

	// Read database credentials
	credentials, err := dbconfig.ReadDatabaseCredentials(filePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Get Kubernetes dynamic client
	_, dynamicClient, err := k8s.GetK8sClient()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Get namespace from environment variable
	namespace := os.Getenv("AAP_NAMESPACE")
	if namespace == "" {
		fmt.Println("Error: AAP_NAMESPACE environment variable is not set.")
		return
	}

	// Start the Watch Loop
	controller.StartWatchLoop(dynamicClient, credentials, namespace)
}

