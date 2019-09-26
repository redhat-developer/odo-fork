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

// RunTaskScript is of type string which indicates the script path for the run task
type RunTaskScript string

// BuildTaskKind is of type string which indicates the kind of build task
type BuildTaskKind string

// WebSphereLibertyImage is of type string which indicates the Liberty image
type WebSphereLibertyImage string

const (
	// Incremental is of type BuildTaskType which indicates it is an incremental build
	Incremental BuildTaskType = "inc"
	// Full is of type BuildTaskType which indicates it is a full build
	Full BuildTaskType = "full"

	// FullBuildTask is the relative path of the IDP full build task in the Persistent Volume's project directory
	FullBuildTask BuildTaskScript = "/.udo/build-container-full.sh"
	// IncrementalBuildTask is the relative path of the IDP incremental build task in the Persistent Volume's project directory
	IncrementalBuildTask BuildTaskScript = "/.udo/build-container-update.sh"

	// FullRunTask is the relative path of the IDP full run task in the Runtime Container Empty Dir Volume's project directory
	FullRunTask RunTaskScript = "/.udo/runtime-container-full.sh"
	// IncrementalRunTask is the relative path of the IDP incremental run task in the Runtime Container Empty Dir Volume's project directory
	IncrementalRunTask RunTaskScript = "/.udo/runtime-container-update.sh"

	// ReusableBuildContainer is a BuildTaskKind where udo will reuse the build container to build projects
	ReusableBuildContainer BuildTaskKind = "ReusableBuildContainer"
	// KubeJob is a BuildTaskKind where udo will kick off a Kube job to build projects
	KubeJob BuildTaskKind = "KubeJob"
	// Component is a BuildTaskKind where udo will deploy a component
	Component BuildTaskKind = "Component"

	// RuntimeImage is the default WebSphere Liberty Runtime image
	RuntimeImage WebSphereLibertyImage = "websphere-liberty:19.0.0.3-webProfile7"
	// RuntimeWithMavenJavaImage is the default WebSphere Liberty Runtime image with Maven and Java installed
	RuntimeWithMavenJavaImage WebSphereLibertyImage = "maysunfaisal/libertymvnjava"
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
