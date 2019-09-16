package component

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/build"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	ktemplates "k8s.io/kubectl/pkg/util/templates"

	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildRecommendedCommandName is the recommended catalog command name
const BuildRecommendedCommandName = "build"

var buildCmdExample = ktemplates.Examples(`  # Command for a full-build
%[1]s full <project name> <reuse build container>

# Command for an incremental-build.
%[1]s inc <project name> <reuse build container>
  `)

// BuildIDPOptions encapsulates the options for the udo catalog list idp command
type BuildIDPOptions struct {
	// list of build options
	buildTaskType       string
	projectName         string
	reuseBuildContainer bool
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewBuildIDPOptions creates a new BuildIDPOptions instance
func NewBuildIDPOptions() *BuildIDPOptions {
	return &BuildIDPOptions{}
}

// Complete completes BuildIDPOptions after they've been created
func (o *BuildIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	fmt.Println("Build arguments: " + strings.Join(args, " "))
	o.Context = genericclioptions.NewContext(cmd)
	o.buildTaskType = args[0]
	o.projectName = args[1]
	reuseBuildContainer, err := strconv.ParseBool(args[2])
	if err != nil {
		return fmt.Errorf("The third option for reusing the build container should be either true or false")
	}
	o.reuseBuildContainer = reuseBuildContainer
	return
}

// Validate validates the BuildIDPOptions based on completed values
func (o *BuildIDPOptions) Validate() (err error) {
	if o.buildTaskType != "full" && o.buildTaskType != "inc" {
		return fmt.Errorf("The first option should be either full or inc")
	}
	return
}

// Run contains the logic for the command associated with BuildIDPOptions
func (o *BuildIDPOptions) Run() (err error) {
	clientset := o.Context.Client.KubeClient
	namespace := o.Context.Client.Namespace

	fmt.Printf("Namespace: %s\n", namespace)

	idpClaimName := build.GetIDPPVC(o.Context.Client, namespace, "app=idp")
	fmt.Printf("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	fmt.Printf("Service Account: %s\n", serviceAccountName)

	if o.reuseBuildContainer == true {
		// Create a Build Container for re-use if not present

		// Create the Reusable Build Container deployment object
		ReusableBuildContainerInstance := build.BuildTask{
			Kind:               build.ReusableBuildContainer,
			Name:               strings.ToLower(o.projectName) + "-reusable-build-container",
			Image:              "docker.io/maven:3.6",
			ContainerName:      "maven-build",
			Namespace:          namespace,
			PVCName:            idpClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged: true,
			MountPath:  "/data/idp/",
			SubPath:    "projects/" + o.projectName,
		}
		ReusableBuildContainerInstance.Labels = map[string]string{
			"app": ReusableBuildContainerInstance.Name,
		}

		// Check if the Reusable Build Container has already been deployed
		// Check if the pod is running and grab the pod name
		fmt.Printf("Checking if Reusable Build Container has already been deployed...\n")
		foundReusableBuildContainer := false
		timeout := int64(10)
		watchOptions := metav1.ListOptions{
			LabelSelector:  "app=" + ReusableBuildContainerInstance.Name,
			TimeoutSeconds: &timeout,
		}
		po, _ := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Reusable Container is up...")
		if po != nil {
			fmt.Printf("Running pod found: %s...\n\n", po.Name)
			ReusableBuildContainerInstance.PodName = po.Name
			foundReusableBuildContainer = true
		}

		if !foundReusableBuildContainer {
			reusableBuildContainerDeploy := ReusableBuildContainerInstance.CreateDeploy()
			reusableBuildContainerService := ReusableBuildContainerInstance.CreateService()

			fmt.Println("===============================")
			fmt.Println("Deploying reusable build container...")
			_, err = clientset.CoreV1().Services(namespace).Create(&reusableBuildContainerService)
			if err != nil {
				fmt.Printf("Unable to create application service: %v\n", err)
				os.Exit(1)
			} else {
				fmt.Println("The service has been created.")
			}
			_, err = clientset.AppsV1().Deployments(namespace).Create(&reusableBuildContainerDeploy)
			if err != nil {
				fmt.Printf("Unable to create application deployment: %v\n", err)
				os.Exit(1)
			} else {
				fmt.Println("The deployment has been created.")
			}
			fmt.Println("===============================")

			// Wait for pods to start and grab the pod name
			fmt.Printf("Waiting for pod to run\n")
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

		fmt.Printf("The Reusable Build Container Pod Name: %s\n", ReusableBuildContainerInstance.PodName)

		// Execute the Mvm command in the Build Container
		command := []string{"/bin/sh", "-c", build.FullBuildTask}
		if o.buildTaskType == "inc" {
			command = []string{"/bin/sh", "-c", build.IncrementalBuildTask}
		}
		// command := []string{"/bin/sh", "-c", "hostname", "-f"}
		output, stderr, err := o.Context.Client.ExecPodCmd(command, ReusableBuildContainerInstance.ContainerName, ReusableBuildContainerInstance.PodName)
		if len(stderr) != 0 {
			fmt.Println("Reusable Build Container stderr: ", stderr)
		}
		if err != nil {
			fmt.Printf("Error occured while executing command %s in the pod %s: %s\n", strings.Join(command, " "), ReusableBuildContainerInstance.PodName, err)
			os.Exit(1)
		} else {
			fmt.Printf("Reusable Build Container Output: \n%s\n", output)
		}

		fmt.Println("Finished executing the IDP Build Task in the Reusable Build Container...")
	} else {
		// Create a Kube Job for building
		fmt.Println("Creating a Kube Job for building...")

		buildTaskJobName := "codewind-liberty-build-job"

		job, err := build.CreateBuildTaskKubeJob(buildTaskJobName, o.buildTaskType, namespace, idpClaimName, "projects/"+o.projectName, o.projectName)
		if err != nil {
			fmt.Println("There was a problem with the job configuration, exiting...")
			panic(err.Error())
		}

		kubeJob, err := clientset.BatchV1().Jobs(namespace).Create(job)
		if err != nil {
			fmt.Println("Failed to create a job, exiting...")
			panic(err.Error())
		}

		fmt.Printf("The job %s has been created\n", kubeJob.Name)
		watchOptions := metav1.ListOptions{
			LabelSelector: "job-name=codewind-liberty-build-job",
		}
		// Wait for pods to start running so that we can tail the logs
		po, err := o.Context.Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the build job to run")
		if err != nil {
			err = errors.New("The build job failed to run")
			return err
		}

		err = o.Context.Client.GetPodLogs(po, os.Stdout)
		if err != nil {
			err = errors.New("Failed to get the logs for the build job")
			return err
		}

		// TODO-KDO: Set owner references
		var jobSucceeded bool
		// Loop and see if the job either succeeded or failed
		loop := true
		for loop == true {
			jobs, err := clientset.BatchV1().Jobs(namespace).List(metav1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}
			for _, job := range jobs.Items {
				if strings.Contains(job.Name, buildTaskJobName) {
					succeeded := job.Status.Succeeded
					failed := job.Status.Failed
					if succeeded == 1 {
						fmt.Printf("The job %s succeeded\n", job.Name)
						jobSucceeded = true
						loop = false
					} else if failed > 0 {
						fmt.Printf("The job %s failed\n", job.Name)
						jobSucceeded = false
						loop = false
					}
				}
			}
		}

		if loop == false {
			// delete the job
			gracePeriodSeconds := int64(0)
			deletionPolicy := metav1.DeletePropagationForeground
			err := clientset.BatchV1().Jobs(namespace).Delete(buildTaskJobName, &metav1.DeleteOptions{
				PropagationPolicy:  &deletionPolicy,
				GracePeriodSeconds: &gracePeriodSeconds,
			})
			if err != nil {
				panic(err.Error())
			} else {
				fmt.Printf("The job %s has been deleted\n", buildTaskJobName)
			}
		}

		if !jobSucceeded {
			fmt.Println("The job failed, exiting...")
			os.Exit(1)
		}
	}

	if o.buildTaskType == "full" {
		// Deploy the application if it is a full build type
		fmt.Println("Deploying application on a full build")

		// Create the Codewind deployment object
		BuildTaskInstance := build.BuildTask{
			Kind:               build.Component,
			Name:               "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
			Image:              "websphere-liberty:19.0.0.3-webProfile7",
			ContainerName:      "libertyproject",
			Namespace:          namespace,
			PVCName:            idpClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged: true,
			MountPath:  "/config",
			SubPath:    "projects/" + o.projectName + "/buildartifacts/",
		}

		BuildTaskInstance.Labels = map[string]string{
			"app":     "javamicroprofiletemplate-selector",
			"chart":   "javamicroprofiletemplate-1.0.0",
			"release": BuildTaskInstance.Name,
		}

		// Deploy Application
		deploy := BuildTaskInstance.CreateDeploy()
		service := BuildTaskInstance.CreateService()

		fmt.Println("===============================")
		fmt.Println("Deploying application...")
		_, err = clientset.CoreV1().Services(namespace).Create(&service)
		if err != nil {
			fmt.Printf("Unable to create application service: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("The service has been created.")
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(&deploy)
		if err != nil {
			fmt.Printf("Unable to create application deployment: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("The deployment has been created.")
		}
		fmt.Println("===============================")
	}

	return
}

// NewCmdBuild implements the udo catalog list idps command
func NewCmdBuild(name, fullName string) *cobra.Command {
	o := NewBuildIDPOptions()

	var buildCmd = &cobra.Command{
		Use:     name,
		Short:   "Start a IDP Build",
		Long:    "Start a IDP Build using the Build Tasks.",
		Example: fmt.Sprintf(buildCmdExample, fullName),
		Args:    cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	return buildCmd
}
