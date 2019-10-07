package component

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"

	ktemplates "k8s.io/kubectl/pkg/util/templates"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildRecommendedCommandName is the recommended catalog command name
const BuildRecommendedCommandName = "build"

var buildCmdExample = ktemplates.Examples(`  # Command for a full-build
%[1]s <project name> --fullbuild

# Command for an incremental-build with runtime.
%[1]s <project name> --useRuntimeContainer
  `)

// BuildIDPOptions encapsulates the options for the udo catalog list idp command
type BuildIDPOptions struct {
	// list of build options
	projectName         string
	useRuntimeContainer bool
	fullBuild           bool
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewBuildIDPOptions creates a new BuildIDPOptions instance
func NewBuildIDPOptions() *BuildIDPOptions {
	return &BuildIDPOptions{}
}

// Complete completes BuildIDPOptions after they've been created
func (o *BuildIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	glog.V(0).Info("Build arguments: " + strings.Join(args, " "))
	o.Context = genericclioptions.NewContext(cmd)
	o.projectName = args[0]
	glog.V(0).Info("useRuntimeContainer flag: ", o.useRuntimeContainer)
	glog.V(0).Info("fullBuild flag: ", o.fullBuild)
	return
}

// Validate validates the BuildIDPOptions based on completed values
func (o *BuildIDPOptions) Validate() (err error) {
	return
}

// Run contains the logic for the command associated with BuildIDPOptions
func (o *BuildIDPOptions) Run() (err error) {

	// if !o.useRuntimeContainer {
	// 	component.BuildTaskExec(o.Context.Client, o.projectName, o.fullBuild)
	// } else {
	// 	component.RunTaskExec(o.Context.Client, o.projectName, o.fullBuild)
	// }

	return
}

// SyncProjectToRunningContainer Wait for the Pod to run, create the targetPath in the Pod and sync the project to the targetPath
func (o *BuildIDPOptions) syncProjectToRunningContainer(watchOptions metav1.ListOptions, sourcePath, targetPath, containerName string) error {
	// Wait for the pod to run
	glog.V(0).Infof("Waiting for pod to run\n")
	po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking if the container is up before syncing")
	if err != nil {
		err = errors.New("The Container failed to run")
		return err
	}
	podName := po.Name
	glog.V(0).Info("The Pod is up and running: " + podName)

	// Before Syncing, create the destination directory in the Build Container
	command := []string{"/bin/sh", "-c", "rm -rf " + targetPath + " && mkdir -p " + targetPath}
	err = o.Context.Client.ExecCMDInContainer(podName, "", command, os.Stdout, os.Stdout, nil, false)
	if err != nil {
		glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), podName, err)
		err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the reusable build container: " + err.Error())
		return err
	}

	// Sync the project to the specified Pod's target path
	err = o.Context.Client.CopyFile(sourcePath, podName, targetPath, []string{}, []string{})
	if err != nil {
		err = errors.New("Unable to copy files to the pod " + podName + ": " + err.Error())
		return err
	}

	return nil
}

// NewCmdBuild implements the udo catalog list idps command
func NewCmdBuild(name, fullName string) *cobra.Command {
	o := NewBuildIDPOptions()

	var buildCmd = &cobra.Command{
		Use:     name,
		Short:   "Start a IDP Build",
		Long:    "Start a IDP Build using the Build Tasks.",
		Example: fmt.Sprintf(buildCmdExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	buildCmd.Flags().BoolVar(&o.useRuntimeContainer, "useRuntimeContainer", false, "Use the runtime container for IDP Builds")
	buildCmd.Flags().BoolVar(&o.fullBuild, "fullBuild", false, "Force a full build")

	return buildCmd
}
