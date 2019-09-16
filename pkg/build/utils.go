package build

import (
	"fmt"
	"os"

	"github.com/redhat-developer/odo-fork/pkg/kclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Build Task Types

	// FullBuildTask is the IDP full build task script path in the Persistent Volume
	FullBuildTask = "/data/idp/bin/build-container-full.sh"
	// IncrementalBuildTask is the IDP incremental build task script path in the Persistent Volume
	IncrementalBuildTask = "/data/idp/bin/build-container-update.sh"

	// Build Task Struct Kind

	// ReusableBuildContainer is a Build Task Kind where udo will reuse the build container to build projects
	ReusableBuildContainer string = "ReusableBuildContainer"
	// KubeJob is a Build Task Kind where udo will kick off a Kube job to build projects
	KubeJob string = "KubeJob"
	// Component is a Build Task Kind where udo will deploy a component
	Component string = "Component"
)

// GetIDPPVC retrieves the PVC (Persistent Volume Claim) associated with the Iterative Development Pack
func GetIDPPVC(client *kclient.Client, namespace string, labels string) string {
	var pvcName string
	clientset := client.KubeClient

	PVCs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil || PVCs == nil {
		fmt.Printf("Error, unable to retrieve PVCs: %v\n", err)
		os.Exit(1)
	} else if len(PVCs.Items) == 1 {
		pvcName = PVCs.Items[0].GetName()
	} else {
		// We couldn't find the workspace PVC, use a default value
		pvcName = "claim-che-workspace"
	}

	return pvcName
}
