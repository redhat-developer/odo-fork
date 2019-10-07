package component

import (
	"os"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/kclient"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

func executetask(client *kclient.Client, task, podName string) error {
	// Execute the Runtime task in the Runtime Container
	command := []string{"/bin/sh", "-c", task}

	glog.V(0).Infof("Executing %s in the pod %s", task, podName)

	err := client.ExecCMDInContainer(podName, "", command, os.Stdout, os.Stdout, nil, false)
	if err != nil {
		glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), podName, err)
		err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the runtime container: " + err.Error())
		return err
	}

	return nil
}
