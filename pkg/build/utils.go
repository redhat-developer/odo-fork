package build

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/kclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
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

// ExecPodCmd executes command in the pod container
func ExecPodCmd(client *kclient.Client, command []string, containerName, podName, namespace string) (string, string, error) {

	fmt.Printf("Executing command: %s in pod: %s container: %s\n", strings.Join(command, " "), podName, containerName)

	clientset := client.KubeClient
	config := client.KubeClientConfig

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		panic(err)
	}

	return stdout.String(), stderr.String(), nil
}

// ListPods list the pods using the selector
func ListPods(client *kclient.Client, namespace string, listOptions metav1.ListOptions) (*corev1.PodList, error) {
	clientset := client.KubeClient
	podList, err := clientset.CoreV1().Pods(namespace).List(listOptions)

	return podList, err
}
