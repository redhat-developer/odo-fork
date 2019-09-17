package build

import (
	"fmt"
	"os"

	"github.com/redhat-developer/odo-fork/pkg/kclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildTaskType is of type string which indiciates the type of build task
type BuildTaskType string

// BuildTaskScript is of type string which indicates the script path for the build task
type BuildTaskScript string

// BuildTaskKind is of type string which indicates the kind of build task
type BuildTaskKind string

const (
	// Incremental is of type BuildTaskType which indicates it is an incremental build
	Incremental BuildTaskType = "inc"
	// Full is of type BuildTaskType which indicates it is a full build
	Full BuildTaskType = "full"

	// FullBuildTask is the IDP full build task script path in the Persistent Volume
	FullBuildTask BuildTaskScript = "/data/idp/bin/build-container-full.sh"
	// IncrementalBuildTask is the IDP incremental build task script path in the Persistent Volume
	IncrementalBuildTask BuildTaskScript = "/data/idp/bin/build-container-update.sh"

	// ReusableBuildContainer is a BuildTaskKind where udo will reuse the build container to build projects
	ReusableBuildContainer BuildTaskKind = "ReusableBuildContainer"
	// KubeJob is a BuildTaskKind where udo will kick off a Kube job to build projects
	KubeJob BuildTaskKind = "KubeJob"
	// Component is a BuildTaskKind where udo will deploy a component
	Component BuildTaskKind = "Component"
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
