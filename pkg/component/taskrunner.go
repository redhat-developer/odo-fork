package component

import (
	"os"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/kclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

func executetask(client *kclient.Client, tasks []string, watchOptions metav1.ListOptions) error {

	pod, err := client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if the Runtime Container is up before executing the Runtime Tasks")
	if err != nil {
		err = errors.New("The Container failed to run")
		return err
	}

	podName := pod.Name

	// Execute the tasks in the specified Container
	for _, task := range tasks {
		command := []string{"/bin/sh", "-c", task}

		glog.V(0).Infof("Executing %s in the pod %s", task, podName)

		err = client.ExecCMDInContainer(podName, "", command, os.Stdout, os.Stdout, nil, false)
		if err != nil {
			glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), podName, err)
			err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the runtime container: " + err.Error())
			return err
		}
	}

	return nil
}
