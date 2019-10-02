package component

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/build"
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
	clientset := o.Context.Client.KubeClient
	namespace := o.Context.Client.Namespace

	glog.V(0).Infof("Namespace: %s\n", namespace)

	idpClaimName := ""
	PVCs, _ := o.Context.Client.GetPVCsFromSelector("app=idp")
	if len(PVCs) == 1 {
		idpClaimName = PVCs[0].GetName()
	}
	glog.V(0).Infof("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	glog.V(0).Infof("Service Account: %s\n", serviceAccountName)

	// cwd is the project root dir, where udo command will run
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	glog.V(0).Infof("CWD: %s\n", cwd)

	if !o.useRuntimeContainer {
		// Create a Build Container for re-use if not present

		// Create the Reusable Build Container deployment object
		ReusableBuildContainerInstance := build.BuildTask{
			UseRuntime:         o.useRuntimeContainer,
			Kind:               build.ReusableBuildContainer,
			Name:               strings.ToLower(o.projectName) + "-reusable-build-container",
			Image:              build.BuildContainerImage,
			ContainerName:      build.BuildContainerName,
			Namespace:          namespace,
			PVCName:            idpClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged: true,
			MountPath:  build.BuildContainerMountPath,
			SubPath:    "projects/" + o.projectName,
		}
		ReusableBuildContainerInstance.Labels = map[string]string{
			"app": ReusableBuildContainerInstance.Name,
		}

		// Check if the Reusable Build Container has already been deployed
		// Check if the pod is running and grab the pod name
		glog.V(0).Infof("Checking if Reusable Build Container has already been deployed...\n")
		foundReusableBuildContainer := false
		timeout := int64(10)
		watchOptions := metav1.ListOptions{
			LabelSelector:  "app=" + ReusableBuildContainerInstance.Name,
			TimeoutSeconds: &timeout,
		}
		po, _ := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Reusable Container is up")
		if po != nil {
			glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
			ReusableBuildContainerInstance.PodName = po.Name
			foundReusableBuildContainer = true
		}

		if !foundReusableBuildContainer {
			glog.V(0).Info("===============================")
			glog.V(0).Info("Creating a pod...")
			volumes, volumeMounts := ReusableBuildContainerInstance.SetVolumes()
			envVars := ReusableBuildContainerInstance.SetEnvVars()

			pod, err := o.Context.Client.CreatePod(ReusableBuildContainerInstance.Name, ReusableBuildContainerInstance.ContainerName, ReusableBuildContainerInstance.Image, ReusableBuildContainerInstance.ServiceAccountName, ReusableBuildContainerInstance.Labels, volumes, volumeMounts, envVars, ReusableBuildContainerInstance.Privileged)
			if err != nil {
				err = errors.New("Failed to create a pod " + ReusableBuildContainerInstance.Name)
				return err
			}
			glog.V(0).Info("Created pod: " + pod.GetName())
			glog.V(0).Info("===============================")
			// Wait for pods to start and grab the pod name
			glog.V(0).Infof("Waiting for pod to run\n")
			watchOptions := metav1.ListOptions{
				LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
			}
			po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Reusable Build Container to run")
			if err != nil {
				err = errors.New("The Reusable Build Container failed to run")
				return err
			}

			ReusableBuildContainerInstance.PodName = po.Name
		}

		glog.V(0).Infof("The Reusable Build Container Pod Name: %s\n", ReusableBuildContainerInstance.PodName)

		watchOptions = metav1.ListOptions{
			LabelSelector: "app=" + ReusableBuildContainerInstance.Name,
		}
		err := o.syncProjectToRunningContainer(watchOptions, cwd, ReusableBuildContainerInstance.MountPath+"/src", ReusableBuildContainerInstance.ContainerName)
		if err != nil {
			glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", ReusableBuildContainerInstance.PodName, err)
			err = errors.New("Unable to sync to the pod: " + err.Error())
			return err
		}

		// Execute the Build Tasks in the Build Container
		command := []string{"/bin/sh", "-c", ReusableBuildContainerInstance.MountPath + "/src" + build.FullBuildTask}
		if !o.fullBuild {
			command = []string{"/bin/sh", "-c", ReusableBuildContainerInstance.MountPath + "/src" + build.IncrementalBuildTask}
		}
		err = o.Context.Client.ExecCMDInContainer(ReusableBuildContainerInstance.PodName, "", command, os.Stdout, os.Stdout, nil, false)
		if err != nil {
			glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), ReusableBuildContainerInstance.PodName, err)
			err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the reusable build container: " + err.Error())
			return err
		}

		glog.V(0).Info("Finished executing the IDP Build Task in the Reusable Build Container...")
	}

	// Create the Runtime Task Instance
	RuntimeTaskInstance := build.BuildTask{
		UseRuntime:         o.useRuntimeContainer,
		Kind:               build.Component,
		Name:               strings.ToLower(o.projectName) + "-runtime",
		Image:              build.RuntimeConainerImage,
		ContainerName:      build.RuntimeContainerName,
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
		MountPath:  build.RuntimeContainerMountPathDefault,
		SubPath:    "projects/" + o.projectName + "/buildartifacts/",
	}

	if o.useRuntimeContainer {
		RuntimeTaskInstance.Image = build.RuntimeContainerImageWithBuildTools
		RuntimeTaskInstance.PVCName = ""
		RuntimeTaskInstance.MountPath = build.RuntimeContainerMountPathEmptyDir
		RuntimeTaskInstance.SubPath = ""
	}

	if o.useRuntimeContainer || o.fullBuild {
		// Check if the Runtime Pod has been deployed
		// Check if the pod is running and grab the pod name
		glog.V(0).Info("Checking if Runtime Container has already been deployed...\n")
		foundRuntimeContainer := false
		timeout := int64(10)
		watchOptions := metav1.ListOptions{
			LabelSelector:  "app=" + RuntimeTaskInstance.Name + "-selector,chart=" + RuntimeTaskInstance.Name + "-1.0.0,release=" + RuntimeTaskInstance.Name,
			TimeoutSeconds: &timeout,
		}
		po, _ := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Runtime Container has already been deployed")
		if po != nil {
			glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
			RuntimeTaskInstance.PodName = po.Name
			foundRuntimeContainer = true
		}

		if !foundRuntimeContainer {
			// Deploy the application if it is a full build type and a running pod is not found
			glog.V(0).Info("Deploying the application")

			RuntimeTaskInstance.Labels = map[string]string{
				"app":     RuntimeTaskInstance.Name + "-selector",
				"chart":   RuntimeTaskInstance.Name + "-1.0.0",
				"release": RuntimeTaskInstance.Name,
			}

			// Deploy Application
			deploy := RuntimeTaskInstance.CreateDeploy()
			service := RuntimeTaskInstance.CreateService()

			glog.V(0).Info("===============================")
			glog.V(0).Info("Deploying application...")
			_, err = clientset.CoreV1().Services(namespace).Create(&service)
			if err != nil {
				err = errors.New("Unable to create component service: " + err.Error())
				return err
			}
			glog.V(0).Info("The service has been created.")

			_, err = clientset.AppsV1().Deployments(namespace).Create(&deploy)
			if err != nil {
				err = errors.New("Unable to create component deployment: " + err.Error())
				return err
			}
			glog.V(0).Info("The deployment has been created.")
			glog.V(0).Info("===============================")

			// Wait for the pod to run
			glog.V(0).Info("Waiting for pod to run\n")
			watchOptions := metav1.ListOptions{
				LabelSelector: "app=" + RuntimeTaskInstance.Name + "-selector,chart=" + RuntimeTaskInstance.Name + "-1.0.0,release=" + RuntimeTaskInstance.Name,
			}
			po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Component Container to run")
			if err != nil {
				err = errors.New("The Component Container failed to run")
				return err
			}
			glog.V(0).Info("The Component Pod is up and running: " + po.Name)
			RuntimeTaskInstance.PodName = po.Name
		}
	}

	if o.useRuntimeContainer {
		watchOptions := metav1.ListOptions{
			LabelSelector: "app=" + RuntimeTaskInstance.Name + "-selector,chart=" + RuntimeTaskInstance.Name + "-1.0.0,release=" + RuntimeTaskInstance.Name,
		}
		err := o.syncProjectToRunningContainer(watchOptions, cwd, RuntimeTaskInstance.MountPath+"/src", RuntimeTaskInstance.ContainerName)
		if err != nil {
			glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", RuntimeTaskInstance.PodName, err)
			err = errors.New("Unable to sync to the pod: " + err.Error())
			return err
		}

		// Execute the Runtime task in the Runtime Container
		command := []string{"/bin/sh", "-c", RuntimeTaskInstance.MountPath + "/src" + build.FullRunTask}
		if !o.fullBuild {
			command = []string{"/bin/sh", "-c", RuntimeTaskInstance.MountPath + "/src" + build.IncrementalRunTask}
		}
		err = o.Context.Client.ExecCMDInContainer(RuntimeTaskInstance.PodName, "", command, os.Stdout, os.Stdout, nil, false)
		if err != nil {
			glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), RuntimeTaskInstance.PodName, err)
			err = errors.New("Unable to exec command " + strings.Join(command, " ") + " in the runtime container: " + err.Error())
			return err
		}
	}

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
	buildCmd.Flags().BoolVar(&o.fullBuild, "fullBuild", false, "Uses the full build scenario for the IDP tasks")

	return buildCmd
}
