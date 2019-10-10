package component

import (
	"os"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/log"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncProjectToRunningContainer Wait for the Pod to run, create the targetPath in the Pod and sync the project to the targetPath
func syncProjectToRunningContainer(client *kclient.Client, watchOptions metav1.ListOptions, sourcePath, targetPath string) error {
	// Wait for the pod to run
	glog.V(0).Infof("Waiting for pod to run\n")
	po, err := client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking if the container is up before syncing")
	if err != nil {
		err = errors.New("The Container failed to run")
		return err
	}
	podName := po.Name
	glog.V(0).Info("The Pod is up and running: " + podName)

	s := log.Spinner("Syncing project to component pod")
	defer s.End(false)

	// Before Syncing, create the destination directory in the Build Container
	command := []string{"/bin/sh", "-c", "rm -rf " + targetPath + " && mkdir -p " + targetPath}
	err = client.ExecCMDInContainer(podName, "", command, os.Stdout, os.Stdout, nil, false)
	if err != nil {
		glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), podName, err)
		err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the reusable build container: " + err.Error())
		return err
	}

	// Sync the project to the specified Pod's target path
	err = client.CopyFile(sourcePath, podName, targetPath, []string{}, []string{})
	if err != nil {
		err = errors.New("Unable to copy files to the pod " + podName + ": " + err.Error())
		return err
	}

	s.End(true)

	return nil
}
