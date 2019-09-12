package component

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/component"
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

// BuildIDPOptions encapsulates the options for the kdo catalog list idp command
type BuildIDPOptions struct {
	// list of build options
	buildTaskType       string
	projectName         string
	reuseBuildContainer bool
	// generic context options common to all commands
	*genericclioptions.Context
}

// NewBuildIDPOptions creates a new ListIDPOptions instance
func NewBuildIDPOptions() *BuildIDPOptions {
	return &BuildIDPOptions{}
}

// Complete completes BuildIDPOptions after they've been created
func (o *BuildIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	fmt.Println("MJF Build inside Complete")
	fmt.Println("MJF args: " + strings.Join(args, " "))
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

// Validate validates the ListIDPOptions based on completed values
func (o *BuildIDPOptions) Validate() (err error) {
	fmt.Println("MJF Build inside Validate")
	if o.buildTaskType != "full" && o.buildTaskType != "inc" {
		return fmt.Errorf("The first option should be either full or inc")
	}
	return
}

// Run contains the logic for the command associated with ListIDPOptions
func (o *BuildIDPOptions) Run() (err error) {
	fmt.Println("MJF Build inside Run")
	clientset := o.Context.Client.KubeClient
	namespace := o.Context.Client.Namespace

	fmt.Printf("Namespace: %s\n", namespace)

	idpClaimName := component.GetIDPPVC(o.Context.Client, namespace, "app=idp")
	fmt.Printf("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	fmt.Printf("Service Account: %s\n", serviceAccountName)

	if o.reuseBuildContainer == true {
		fmt.Println("Reusing the build container")
	}

	buildTaskJobName := "codewind-liberty-build-job"

	job, err := component.CreateBuildTaskKubeJob(buildTaskJobName, o.buildTaskType, namespace, idpClaimName, "projects/"+o.projectName, o.projectName)
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

	// Wait for pods to start running so that we can tail the logs
	fmt.Printf("Waiting for pod to run\n")
	foundRunningPod := false
	for foundRunningPod == false {

		podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: "job-name=codewind-liberty-build-job",
			FieldSelector: "status.phase=Running",
		})

		if err != nil {
			continue
		}

		for _, pod := range podList.Items {
			fmt.Printf("Running pod found: %s Retrieving logs...\n\n", pod.Name)
			foundRunningPod = true
		}
	}

	// Print logs before deleting the job
	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "job-name=codewind-liberty-build-job",
	})

	for _, pod := range podList.Items {
		fmt.Printf("Retrieving logs for pod: %s\n\n", pod.Name)
		req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Follow: true,
		})
		readCloser, err := req.Stream()
		if err != nil {
			fmt.Printf("Unable to retrieve logs for pod: %s\n", pod.Name)
			continue
		}

		defer readCloser.Close()
		_, err = io.Copy(os.Stdout, readCloser)
	}

	// TODO: Set owner references
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

	// Create the Codewind deployment object
	BuildTaskInstance := component.BuildTask{
		Name:               "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		Image:              "websphere-liberty:19.0.0.3-webProfile7",
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
	}

	if o.buildTaskType == "full" {
		// Deploy Application
		deploy := component.CreateComponentDeploy(BuildTaskInstance, o.projectName)
		service := component.CreateComponentService(BuildTaskInstance)

		fmt.Println("===============================")
		fmt.Println("Deploying application...")
		_, err = clientset.CoreV1().Services(namespace).Create(&service)
		if err != nil {
			fmt.Printf("Unable to create application service: %v\n", err)
			os.Exit(1)
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(&deploy)
		if err != nil {
			fmt.Printf("Unable to create application deployment: %v\n", err)
			os.Exit(1)
		}
	}

	return
}

// NewCmdBuild implements the kdo catalog list idps command
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
