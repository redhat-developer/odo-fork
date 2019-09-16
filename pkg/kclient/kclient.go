package kclient

import (
	taro "archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/preference"
	"github.com/redhat-developer/odo-fork/pkg/util"

	// api resource types
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"
)

var (
	DEPLOYMENT_CONFIG_NOT_FOUND_ERROR_STR string = "deployment \"%s\" not found"
	DEPLOYMENT_CONFIG_NOT_FOUND           error  = fmt.Errorf("Requested deployment does not exist")
)

// CreateArgs is a container of attributes of component create action
type CreateArgs struct {
	Name               string
	SourcePath         string
	SourceRef          string
	SourceType         config.SrcType
	ImageName          string
	EnvVars            []string
	Ports              []string
	Resources          *corev1.ResourceRequirements
	ApplicationName    string
	Wait               bool
	StorageToBeMounted map[string]*corev1.PersistentVolumeClaim
	StdOut             io.Writer
}

const (
	// The length of the string to be generated for names of resources
	nameLength = 5

	// waitForPodTimeOut controls how long we should wait for a pod before giving up
	waitForPodTimeOut = 240 * time.Second

	// ComponentPortAnnotationName annotation is used on the secrets that are created for each exposed port of the component
	ComponentPortAnnotationName = "component-port"
)

// errorMsg is the message for user when invalid configuration error occurs
const errorMsg = `
Please ensure you have an active kubernetes context to your cluster. 

Consult your Kubernetes distribution's documentation for more details
`

// Client is a collection of fields used for client configuration and interaction
type Client struct {
	KubeClient       kubernetes.Interface
	AppsV1Client     appsv1Client.AppsV1Interface
	CoreV1Client     corev1Client.CoreV1Interface
	KubeConfig       clientcmd.ClientConfig
	KubeClientConfig *rest.Config
	Namespace        string
}

// New creates a new client
func New(skipConnectionCheck bool) (*Client, error) {
	var client Client

	// initialize client-go clients
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	client.KubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := client.KubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.New(err.Error() + errorMsg)
	}
	client.KubeClientConfig = config

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	client.KubeClient = kubeClient

	namespace, _, err := client.KubeConfig.Namespace()
	if err != nil {
		return nil, err
	}
	client.Namespace = namespace

	return &client, nil
}

// ParseImageName parse image reference
// returns (imageNamespace, imageName, tag, digest, error)
// if image is referenced by tag (name:tag)  than digest is ""
// if image is referenced by digest (name@digest) than  tag is ""
func ParseImageName(image string) (string, string, string, string, error) {
	digestParts := strings.Split(image, "@")
	if len(digestParts) == 2 {
		// image is references digest
		// Safe path image name and digest are non empty, else error
		if digestParts[0] != "" && digestParts[1] != "" {
			// Image name might be fully qualified name of form: Namespace/ImageName
			imangeNameParts := strings.Split(digestParts[0], "/")
			if len(imangeNameParts) == 2 {
				return imangeNameParts[0], imangeNameParts[1], "", digestParts[1], nil
			}
			return "", imangeNameParts[0], "", digestParts[1], nil
		}
	} else if len(digestParts) == 1 && digestParts[0] != "" { // Filter out empty image name
		tagParts := strings.Split(image, ":")
		if len(tagParts) == 2 {
			// ":1.0.0 is invalid image name"
			if tagParts[0] != "" {
				// Image name might be fully qualified name of form: Namespace/ImageName
				imangeNameParts := strings.Split(tagParts[0], "/")
				if len(imangeNameParts) == 2 {
					return imangeNameParts[0], imangeNameParts[1], tagParts[1], "", nil
				}
				return "", tagParts[0], tagParts[1], "", nil
			}
		} else if len(tagParts) == 1 {
			// Image name might be fully qualified name of form: Namespace/ImageName
			imangeNameParts := strings.Split(tagParts[0], "/")
			if len(imangeNameParts) == 2 {
				return imangeNameParts[0], imangeNameParts[1], "latest", "", nil
			}
			return "", tagParts[0], "latest", "", nil
		}
	}
	return "", "", "", "", fmt.Errorf("invalid image reference %s", image)

}

// isServerUp returns true if server is up and running
// server parameter has to be a valid url
func isServerUp(server string) bool {
	// initialising the default timeout, this will be used
	// when the value is not readable from config
	ocRequestTimeout := preference.DefaultTimeout * time.Second
	// checking the value of timeout in config
	// before proceeding with default timeout
	cfg, configReadErr := preference.New()
	if configReadErr != nil {
		glog.V(4).Info(errors.Wrap(configReadErr, "unable to read config file"))
	} else {
		ocRequestTimeout = time.Duration(cfg.GetTimeout()) * time.Second
	}
	address, err := util.GetHostWithPort(server)
	if err != nil {
		glog.V(4).Infof("Unable to parse url %s (%s)", server, err)
	}
	glog.V(4).Infof("Trying to connect to server %s", address)
	_, connectionError := net.DialTimeout("tcp", address, time.Duration(ocRequestTimeout))
	if connectionError != nil {
		glog.V(4).Info(errors.Wrap(connectionError, "unable to connect to server"))
		return false
	}

	glog.V(4).Infof("Server %v is up", server)
	return true
}

// GetCurrentNamespace returns the current namespace
func (c *Client) GetCurrentNamespace() string {
	return c.Namespace
}

// GetNamespaceNames return list of existing namespaces that user has access to.
func (c *Client) GetNamespaceNames() ([]string, error) {
	namespaces, err := c.KubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list namespaces")
	}

	var namespaceNames []string
	for _, p := range namespaces.Items {
		namespaceNames = append(namespaceNames, p.Name)
	}
	return namespaceNames, nil
}

// GetNamespace returns namespace based on the name of the project.Errors related to
// namespace not being found or forbidden are translated to nil project for compatibility
func (c *Client) GetNamespace(namespace string) (*corev1.Namespace, error) {
	ns, err := c.KubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		istatus, ok := err.(kerrors.APIStatus)
		if ok {
			status := istatus.Status()
			if status.Reason == metav1.StatusReasonNotFound || status.Reason == metav1.StatusReasonForbidden {
				return nil, nil
			}
		} else {
			return nil, err
		}

	}
	return ns, err

}

// CreateNewNamespace creates namespace with given projectName
func (c *Client) CreateNewNamespace(namespace string, wait bool) error {
	// Instantiate watcher before requesting new namespace
	// If watched is created after the namespace it can lead to situation when the namespace is created before the watcher.
	// When this happens, it gets stuck waiting for event that already happened.
	var watcher watch.Interface
	if wait {
		watcher, err := c.KubeClient.CoreV1().Namespaces().Watch(metav1.ListOptions{
			FieldSelector: fields.Set{"metadata.name": namespace}.AsSelector().String(),
		})
		if err != nil {
			return errors.Wrapf(err, "unable to watch new namespace %s creation", namespace)
		}
		defer watcher.Stop()
	}

	_, err := c.KubeClient.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})

	if err != nil {
		return errors.Wrapf(err, "unable to create new project %s", namespace)
	}

	if watcher != nil {
		for {
			val, ok := <-watcher.ResultChan()
			if !ok {
				break
			}
			if e, ok := val.Object.(*corev1.Namespace); ok {
				glog.V(4).Infof("Namespace %s now exists", e.Name)
				return nil
			}
		}
	}

	return nil
}

// SetCurrentNamespace sets the given namespace to be the current namespace
func (c *Client) SetCurrentNamespace(namespace string) error {
	rawConfig, err := c.KubeConfig.RawConfig()
	if err != nil {
		return errors.Wrapf(err, "unable to switch to %s namespace", namespace)
	}

	rawConfig.Contexts[rawConfig.CurrentContext].Namespace = namespace

	err = clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), rawConfig, true)
	if err != nil {
		return errors.Wrapf(err, "unable to switch to %s namespace", namespace)
	}

	c.Namespace = namespace
	return nil
}

// addLabelsToArgs adds labels from map to args as a new argument in format that oc requires
// --labels label1=value1,label2=value2
func addLabelsToArgs(labels map[string]string, args []string) []string {
	if labels != nil {
		var labelsString []string
		for key, value := range labels {
			labelsString = append(labelsString, fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "--labels")
		args = append(args, strings.Join(labelsString, ","))
	}

	return args
}

// GetSecret returns the Secret object in the given namespace
func (c *Client) GetSecret(name, namespace string) (*corev1.Secret, error) {
	secret, err := c.KubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the secret %s", secret)
	}
	return secret, nil
}

// Create a secret for each port, containing the host and port of the component
// This is done so other components can later inject the secret into the environment
// and have the "coordinates" to communicate with this component
func (c *Client) createSecrets(componentName string, commonObjectMeta metav1.ObjectMeta, svc *corev1.Service) error {
	originalName := commonObjectMeta.Name
	for _, svcPort := range svc.Spec.Ports {
		portAsString := fmt.Sprintf("%v", svcPort.Port)

		// we need to create multiple secrets, so each one has to contain the port in it's name
		// so we change the name of each secret by adding the port number
		commonObjectMeta.Name = fmt.Sprintf("%v-%v", originalName, portAsString)

		// we also add the port as an annotation to the secret
		// this comes in handy when we need to "query" for the appropriate secret
		// of a component based on the port
		commonObjectMeta.Annotations[ComponentPortAnnotationName] = portAsString

		err := c.CreateSecret(
			commonObjectMeta,
			map[string]string{
				secretKeyName(componentName, "host"): svc.Name,
				secretKeyName(componentName, "port"): portAsString,
			})

		if err != nil {
			return errors.Wrapf(err, "unable to create Secret for %s", commonObjectMeta.Name)
		}
	}

	// restore the original values of the fields we changed
	commonObjectMeta.Name = originalName
	delete(commonObjectMeta.Annotations, ComponentPortAnnotationName)

	return nil
}

func secretKeyName(componentName, baseKeyName string) string {
	return fmt.Sprintf("COMPONENT_%v_%v", strings.Replace(strings.ToUpper(componentName), "-", "_", -1), strings.ToUpper(baseKeyName))
}

// uniqueAppendOrOverwriteEnvVars appends/overwrites the passed existing list of env vars with the elements from the to-be appended passed list of envs
func uniqueAppendOrOverwriteEnvVars(existingEnvs []corev1.EnvVar, envVars ...corev1.EnvVar) []corev1.EnvVar {
	mapExistingEnvs := make(map[string]corev1.EnvVar)
	var retVal []corev1.EnvVar

	// Convert slice of existing env vars to map to check for existence
	for _, envVar := range existingEnvs {
		mapExistingEnvs[envVar.Name] = envVar
	}

	// For each new envVar to be appended, Add(if envVar with same name doesn't already exist) / overwrite(if envVar with same name already exists) the map
	for _, newEnvVar := range envVars {
		mapExistingEnvs[newEnvVar.Name] = newEnvVar
	}

	// append the values to the final slice
	// don't loop because we need them in order
	for _, envVar := range existingEnvs {
		if val, ok := mapExistingEnvs[envVar.Name]; ok {
			retVal = append(retVal, val)
			delete(mapExistingEnvs, envVar.Name)
		}
	}

	for _, newEnvVar := range envVars {
		if val, ok := mapExistingEnvs[newEnvVar.Name]; ok {
			retVal = append(retVal, val)
		}
	}

	return retVal
}

// deleteEnvVars deletes the passed env var from the list of passed env vars
// Parameters:
//	existingEnvs: Slice of existing env vars
//	envTobeDeleted: The name of env var to be deleted
// Returns:
//	slice of env vars with delete reflected
func deleteEnvVars(existingEnvs []corev1.EnvVar, envTobeDeleted string) []corev1.EnvVar {
	retVal := make([]corev1.EnvVar, len(existingEnvs))
	copy(retVal, existingEnvs)
	for ind, envVar := range retVal {
		if envVar.Name == envTobeDeleted {
			retVal = append(retVal[:ind], retVal[ind+1:]...)
			break
		}
	}
	return retVal
}

// CreateComponentResources creates deployment, service and other kubernetes resources for the component
func (c *Client) CreateComponentResources(params CreateArgs, commonObjectMeta metav1.ObjectMeta) error {
	imageNS, imageName, imageTag, _, err := ParseImageName(params.ImageName)

	if err != nil {
		return errors.Wrap(err, "unable to create component resoures")
	}

	inputEnvs, err := GetInputEnvVarsFromStrings(params.EnvVars)
	if err != nil {
		return errors.Wrapf(err, "error adding environment variables to the container")
	}

	// TODO-KDO: Get port information from component settings
	containerPorts := []corev1.ContainerPort{
		{
			ContainerPort: int32(9090),
			Name:          "http",
		},
	}

	commonImageMeta := CommonImageMeta{
		Name:      imageName,
		Tag:       imageTag,
		Namespace: imageNS,
		Ports:     containerPorts,
	}

	// Generate the Deployment that will be used.
	deployment := generateDeployment(
		commonObjectMeta,
		commonImageMeta,
		inputEnvs,
		[]corev1.EnvFromSource{},
		params.Resources,
	)

	if len(inputEnvs) != 0 {
		err = updateEnvVar(&deployment, inputEnvs)
		if err != nil {
			return errors.Wrapf(err, "unable to add env vars to the container")
		}
	}

	// Attach any volumes to the deployment, before creating it on the cluster
	err = addOrRemoveVolumeAndVolumeMount(c, &deployment, params.StorageToBeMounted, nil)

	_, err = c.KubeClient.AppsV1().Deployments(c.Namespace).Create(&deployment)
	if err != nil {
		return errors.Wrapf(err, "unable to create Deployment for %s", commonObjectMeta.Name)
	}

	svc, err := c.CreateService(commonObjectMeta, deployment.Spec.Template.Spec.Containers[0].Ports)
	if err != nil {
		return errors.Wrapf(err, "unable to create Service for %s", commonObjectMeta.Name)
	}

	err = c.createSecrets(params.Name, commonObjectMeta, svc)
	if err != nil {
		return err
	}

	return nil
}

// CreateService generates and creates the service
// commonObjectMeta is the ObjectMeta for the service
// containerPorts is an array of container ports
func (c *Client) CreateService(commonObjectMeta metav1.ObjectMeta, containerPorts []corev1.ContainerPort) (*corev1.Service, error) {
	// generate and create Service
	var svcPorts []corev1.ServicePort
	for _, containerPort := range containerPorts {
		svcPort := corev1.ServicePort{
			Name:       containerPort.Name,
			Port:       containerPort.ContainerPort,
			Protocol:   containerPort.Protocol,
			TargetPort: intstr.FromInt(int(containerPort.ContainerPort)),
		}
		svcPorts = append(svcPorts, svcPort)
	}
	svc := corev1.Service{
		ObjectMeta: commonObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: svcPorts,
			Selector: map[string]string{
				"deployment": commonObjectMeta.Name,
			},
		},
	}
	createdSvc, err := c.KubeClient.CoreV1().Services(c.Namespace).Create(&svc)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create Service for %s", commonObjectMeta.Name)
	}
	return createdSvc, err
}

// CreateSecret generates and creates the secret
// commonObjectMeta is the ObjectMeta for the service
func (c *Client) CreateSecret(objectMeta metav1.ObjectMeta, data map[string]string) error {

	secret := corev1.Secret{
		ObjectMeta: objectMeta,
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
	_, err := c.KubeClient.CoreV1().Secrets(c.Namespace).Create(&secret)
	if err != nil {
		return errors.Wrapf(err, "unable to create secret for %s", objectMeta.Name)
	}
	return nil
}

// updateEnvVar updates the environmental variables to the container in the Deployment
// deployment is the deployment to be updated
// envVars is the array containing the corev1.EnvVar values
func updateEnvVar(deployment *appsv1.Deployment, envVars []corev1.EnvVar) error {
	numContainers := len(deployment.Spec.Template.Spec.Containers)
	if numContainers != 1 {
		return fmt.Errorf("expected exactly one container in Deployment %v, got %v", deployment.Name, numContainers)
	}

	deployment.Spec.Template.Spec.Containers[0].Env = envVars
	return nil
}

// WaitAndGetPod block and waits until pod matching selector is in in Running state
// desiredPhase cannot be PodFailed or PodUnknown
func (c *Client) WaitAndGetPod(watchOptions metav1.ListOptions, desiredPhase corev1.PodPhase, waitMessage string) (*corev1.Pod, error) {
	glog.V(4).Infof("Waiting for %s pod", watchOptions.LabelSelector)
	s := log.Spinner(waitMessage)
	defer s.End(false)

	w, err := c.KubeClient.CoreV1().Pods(c.Namespace).Watch(watchOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch pod")
	}
	defer w.Stop()

	podChannel := make(chan *corev1.Pod)
	watchErrorChannel := make(chan error)

	go func() {
	loop:
		for {
			val, ok := <-w.ResultChan()
			if !ok {
				watchErrorChannel <- errors.New("watch channel was closed")
				break loop
			}
			if e, ok := val.Object.(*corev1.Pod); ok {
				glog.V(4).Infof("Status of %s pod is %s", e.Name, e.Status.Phase)
				switch e.Status.Phase {
				case desiredPhase:
					s.End(true)
					glog.V(4).Infof("Pod %s is %v", e.Name, desiredPhase)
					podChannel <- e
					break loop
				case corev1.PodFailed, corev1.PodUnknown:
					watchErrorChannel <- errors.Errorf("pod %s status %s", e.Name, e.Status.Phase)
					break loop
				}
			} else {
				watchErrorChannel <- errors.New("unable to convert event object to Pod")
				break loop
			}
		}
		close(podChannel)
		close(watchErrorChannel)
	}()

	select {
	case val := <-podChannel:
		return val, nil
	case err := <-watchErrorChannel:
		return nil, err
	case <-time.After(waitForPodTimeOut):
		return nil, errors.Errorf("waited %s but couldn't find running pod matching selector: '%s'", waitForPodTimeOut, watchOptions.LabelSelector)
	}
}

// GetPodLogs streams the specified pod's logs to the specified output stream
func (c *Client) GetPodLogs(pod *corev1.Pod, file *os.File) (err error) {
	fmt.Printf("Retrieving job logs for pod: %s\n\n", pod.Name)
	req := c.KubeClient.CoreV1().Pods(c.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	})
	readCloser, err := req.Stream()
	defer readCloser.Close()
	if err != nil {
		fmt.Printf("Unable to retrieve job logs for pod: %s\n", pod.Name)
		return
	}

	_, err = io.Copy(file, readCloser)
	return err
}

// ExecPodCmd executes command in the pod container
func (c *Client) ExecPodCmd(command []string, containerName, podName string) (string, string, error) {

	fmt.Printf("Executing command: %s in pod: %s container: %s\n", strings.Join(command, " "), podName, containerName)

	clientset := c.KubeClient
	config := c.KubeClientConfig
	namespace := c.Namespace

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

// WaitAndGetSecret blocks and waits until the secret is available
func (c *Client) WaitAndGetSecret(name string, namespace string) (*corev1.Secret, error) {
	glog.V(4).Infof("Waiting for secret %s to become available", name)

	w, err := c.KubeClient.CoreV1().Secrets(namespace).Watch(metav1.ListOptions{
		FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch secret")
	}
	defer w.Stop()
	for {
		val, ok := <-w.ResultChan()
		if !ok {
			break
		}
		if e, ok := val.Object.(*corev1.Secret); ok {
			glog.V(4).Infof("Secret %s now exists", e.Name)
			return e, nil
		}
	}
	return nil, errors.Errorf("unknown error while waiting for secret '%s'", name)
}

// DeleteNamespace deletes given namespace
func (c *Client) DeleteNamespace(name string) error {
	err := c.KubeClient.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to delete namespace")
	}

	// wait for delete to complete
	w, err := c.KubeClient.CoreV1().Namespaces().Watch(metav1.ListOptions{
		FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(),
	})
	if err != nil {
		return errors.Wrapf(err, "unable to watch namespace")
	}

	defer w.Stop()
	for {
		val, ok := <-w.ResultChan()
		// When marked for deletion... val looks like:
		/*
			val: {
				Type:MODIFIED
				Object:&Project{
					ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{...},
					Spec:ProjectSpec{...},
					Status:ProjectStatus{
						Phase:Terminating,
					},
				}
			}
		*/
		// Post deletion val will look like:
		/*
			val: {
				Type:DELETED
				Object:&Project{
					ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{...},
					Spec:ProjectSpec{...},
					Status:ProjectStatus{
						Phase:,
					},
				}
			}
		*/
		if !ok {
			return fmt.Errorf("received unexpected signal %+v on namespace watch channel", val)
		}
		// So we depend on val.Type as val.Object.Status.Phase is just empty string and not a mapped value constant
		if ns, ok := val.Object.(*corev1.Namespace); ok {
			glog.V(4).Infof("Status of delete of namespace %s is %s", name, ns.Status.Phase)
			switch ns.Status.Phase {
			//ns.Status.Phase can only be "Terminating" or "Active" or ""
			case "":
				if val.Type == watch.Deleted {
					return nil
				}
				if val.Type == watch.Error {
					return fmt.Errorf("failed watching the deletion of namespace %s", name)
				}
			}
		}
	}
}

// GetDeploymentLabelValues get label values of given label from objects in project that are matching selector
// returns slice of unique label values
func (c *Client) GetDeploymentLabelValues(label string, selector string) ([]string, error) {

	// List Deployment according to selectors
	deploymentList, err := c.KubeClient.AppsV1().Deployments(c.Namespace).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list Deployments")
	}

	// Grab all the matched strings
	var values []string
	for _, elem := range deploymentList.Items {
		for key, val := range elem.Labels {
			if key == label {
				values = append(values, val)
			}
		}
	}

	// Sort alphabetically
	sort.Strings(values)

	return values, nil
}

// Service struct holds the service name and its corresponding list of plans
type Service struct {
	Name     string
	Hidden   bool
	PlanList []string
}

// CreateIngress creates an ingress object for the given service and with the given labels
// serviceName is the name of the service for the target reference
// ingressDomain is the ingress domain to use for the ingress
// portNumber is the target port of the ingress
func (c *Client) CreateIngress(name string, serviceName string, ingressDomain string, portNumber intstr.IntOrString, labels map[string]string) (*extensionsv1.Ingress, error) {
	ingress := &extensionsv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: extensionsv1.IngressSpec{
			Rules: []extensionsv1.IngressRule{
				{
					Host: ingressDomain,
					IngressRuleValue: extensionsv1.IngressRuleValue{
						HTTP: &extensionsv1.HTTPIngressRuleValue{
							Paths: []extensionsv1.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensionsv1.IngressBackend{
										ServiceName: serviceName,
										ServicePort: portNumber,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	r, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.Namespace).Create(ingress)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ingress")
	}
	return r, nil
}

// DeleteIngress deleted the given route
func (c *Client) DeleteIngress(name string) error {
	err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.Namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to delete ingress")
	}
	return nil
}

// ListIngresses lists all the ingresses based on the given label selector
func (c *Client) ListIngresses(labelSelector string) ([]extensionsv1.Ingress, error) {
	routeList, err := c.KubeClient.ExtensionsV1beta1().Ingresses(c.Namespace).List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get ingress list")
	}

	return routeList.Items, nil
}

// ListIngressNames lists all the names of the ingresses based on the given label
// selector
func (c *Client) ListIngressNames(labelSelector string) ([]string, error) {
	ingresses, err := c.ListIngresses(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list ingresses")
	}

	var routeNames []string
	for _, r := range ingresses {
		routeNames = append(routeNames, r.Name)
	}

	return routeNames, nil
}

// ListSecrets lists all the secrets based on the given label selector
func (c *Client) ListSecrets(labelSelector string) ([]corev1.Secret, error) {
	listOptions := metav1.ListOptions{}
	if len(labelSelector) > 0 {
		listOptions = metav1.ListOptions{
			LabelSelector: labelSelector,
		}
	}

	secretList, err := c.KubeClient.CoreV1().Secrets(c.Namespace).List(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secret list")
	}

	return secretList.Items, nil
}

// GetDeploymentsFromSelector returns an array of Deployment
// resources which match the given selector
func (c *Client) GetDeploymentsFromSelector(selector string) ([]appsv1.Deployment, error) {
	var depList *appsv1.DeploymentList
	var err error
	if selector != "" {
		depList, err = c.KubeClient.AppsV1().Deployments(c.Namespace).List(metav1.ListOptions{
			LabelSelector: selector,
		})
	} else {
		depList, err = c.KubeClient.AppsV1().Deployments(c.Namespace).List(metav1.ListOptions{
			FieldSelector: fields.Set{"metadata.namespace": c.Namespace}.AsSelector().String(),
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to list Deployments")
	}
	return depList.Items, nil
}

// GetServicesFromSelector returns an array of Service resources which match the
// given selector
func (c *Client) GetServicesFromSelector(selector string) ([]corev1.Service, error) {
	serviceList, err := c.KubeClient.CoreV1().Services(c.Namespace).List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list Services")
	}
	return serviceList.Items, nil
}

// GetDeploymentFromName returns the Deployment resource given
// the Deployment name
func (c *Client) GetDeploymentFromName(name string) (*appsv1.Deployment, error) {
	glog.V(4).Infof("Getting Deployment: %s", name)
	deployment, err := c.KubeClient.AppsV1().Deployments(c.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), fmt.Sprintf(DEPLOYMENT_CONFIG_NOT_FOUND_ERROR_STR, name)) {
			return nil, errors.Wrapf(err, "unable to get Deployment %s", name)
		} else {
			return nil, DEPLOYMENT_CONFIG_NOT_FOUND
		}
	}
	return deployment, nil
}

// GetPVCsFromSelector returns the PVCs based on the given selector
func (c *Client) GetPVCsFromSelector(selector string) ([]corev1.PersistentVolumeClaim, error) {
	pvcList, err := c.KubeClient.CoreV1().PersistentVolumeClaims(c.Namespace).List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get PVCs for selector: %v", selector)
	}

	return pvcList.Items, nil
}

// GetPVCNamesFromSelector returns the PVC names for the given selector
func (c *Client) GetPVCNamesFromSelector(selector string) ([]string, error) {
	pvcs, err := c.GetPVCsFromSelector(selector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get PVCs from selector")
	}

	var names []string
	for _, pvc := range pvcs {
		names = append(names, pvc.Name)
	}

	return names, nil
}

// GetOneDeploymentFromSelector returns the Deployment object associated
// with the given selector.
// An error is thrown when exactly one Deployment is not found for the
// selector.
func (c *Client) GetOneDeploymentFromSelector(selector string) (*appsv1.Deployment, error) {
	deploymentConfigs, err := c.GetDeploymentsFromSelector(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get DeploymentCo for the selector: %v", selector)
	}

	numDC := len(deploymentConfigs)
	if numDC == 0 {
		return nil, fmt.Errorf("no Deployment was found for the selector: %v", selector)
	} else if numDC > 1 {
		return nil, fmt.Errorf("multiple Deployments exist for the selector: %v. Only one must be present", selector)
	}

	return &deploymentConfigs[0], nil
}

// GetOnePodFromSelector returns the Pod  object associated with the given selector.
// An error is thrown when exactly one Pod is not found.
func (c *Client) GetOnePodFromSelector(selector string) (*corev1.Pod, error) {

	pods, err := c.KubeClient.CoreV1().Pods(c.Namespace).List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get Pod for the selector: %v", selector)
	}
	numPods := len(pods.Items)
	if numPods == 0 {
		return nil, fmt.Errorf("no Pod was found for the selector: %v", selector)
	} else if numPods > 1 {
		return nil, fmt.Errorf("multiple Pods exist for the selector: %v. Only one must be present", selector)
	}

	return &pods.Items[0], nil
}

// CopyFile copies localPath directory or list of files in copyFiles list to the directory in running Pod.
// copyFiles is list of changed files captured during `odo watch` as well as binary file path
// During copying binary components, localPath represent base directory path to binary and copyFiles contains path of binary
// During copying local source components, localPath represent base directory path whereas copyFiles is empty
// During `odo watch`, localPath represent base directory path whereas copyFiles contains list of changed Files
func (c *Client) CopyFile(localPath string, targetPodName string, targetPath string, copyFiles []string, globExps []string) error {

	// Destination is set to "ToSlash" as all containers being ran within OpenShift / S2I are all
	// Linux based and thus: "\opt\app-root\src" would not work correctly.
	dest := filepath.ToSlash(filepath.Join(targetPath, filepath.Base(localPath)))
	targetPath = filepath.ToSlash(targetPath)

	glog.V(4).Infof("CopyFile arguments: localPath %s, dest %s, copyFiles %s, globalExps %s", localPath, dest, copyFiles, globExps)
	reader, writer := io.Pipe()
	// inspired from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L235
	go func() {
		defer writer.Close()

		var err error
		err = makeTar(localPath, dest, writer, copyFiles, globExps)
		if err != nil {
			glog.Errorf("Error while creating tar: %#v", err)
			os.Exit(1)
		}

	}()

	// cmdArr will run inside container
	cmdArr := []string{"tar", "xf", "-", "-C", targetPath, "--strip", "1"}
	err := c.ExecCMDInContainer(targetPodName, cmdArr, nil, nil, reader, false)
	if err != nil {
		return err
	}
	return nil
}

// checkFileExist check if given file exists or not
func checkFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// makeTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L309
// srcPath is ignored if files is set
func makeTar(srcPath, destPath string, writer io.Writer, files []string, globExps []string) error {
	// TODO: use compression here?
	tarWriter := taro.NewWriter(writer)
	defer tarWriter.Close()
	srcPath = filepath.Clean(srcPath)

	// "ToSlash" is used as all containers within OpenShisft are Linux based
	// and thus \opt\app-root\src would be an invalid path. Backward slashes
	// are converted to forward.
	destPath = filepath.ToSlash(filepath.Clean(destPath))

	glog.V(4).Infof("makeTar arguments: srcPath: %s, destPath: %s, files: %+v", srcPath, destPath, files)
	if len(files) != 0 {
		//watchTar
		for _, fileName := range files {
			if checkFileExist(fileName) {
				// Fetch path of source file relative to that of source base path so that it can be passed to recursiveTar
				// which uses path relative to base path for taro header to correctly identify file location when untarred
				srcFile, err := filepath.Rel(srcPath, fileName)
				if err != nil {
					return err
				}
				srcFile = filepath.Join(filepath.Base(srcPath), srcFile)
				// The file could be a regular file or even a folder, so use recursiveTar which handles symlinks, regular files and folders
				err = recursiveTar(filepath.Dir(srcPath), srcFile, filepath.Dir(destPath), srcFile, tarWriter, globExps)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return recursiveTar(filepath.Dir(srcPath), filepath.Base(srcPath), filepath.Dir(destPath), filepath.Base(destPath), tarWriter, globExps)
	}

	return nil
}

// Tar will be used to tar files using odo watch
// inspired from https://gist.github.com/jonmorehouse/9060515
func tar(tw *taro.Writer, fileName string, destFile string) error {
	stat, _ := os.Lstat(fileName)

	// now lets create the header as needed for this file within the tarball
	hdr, err := taro.FileInfoHeader(stat, fileName)
	if err != nil {
		return err
	}
	splitFileName := strings.Split(fileName, destFile)[1]

	// hdr.Name can have only '/' as path separator, next line makes sure there is no '\'
	// in hdr.Name on Windows by replacing '\' to '/' in splitFileName. destFile is
	// a result of path.Base() call and never have '\' in it.
	hdr.Name = destFile + strings.Replace(splitFileName, "\\", "/", -1)
	// write the header to the tarball archive
	err = tw.WriteHeader(hdr)
	if err != nil {
		return err
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// copy the file data to the tarball
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil
}

// recursiveTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L319
func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *taro.Writer, globExps []string) error {
	glog.V(4).Infof("recursiveTar arguments: srcBase: %s, srcFile: %s, destBase: %s, destFile: %s", srcBase, srcFile, destBase, destFile)

	// The destination is a LINUX container and thus we *must* use ToSlash in order
	// to get the copying over done correctly..
	destBase = filepath.ToSlash(destBase)
	destFile = filepath.ToSlash(destFile)
	glog.V(4).Infof("Corrected destinations: base: %s file: %s", destBase, destFile)

	joinedPath := filepath.Join(srcBase, srcFile)
	matchedPathsDir, err := filepath.Glob(joinedPath)
	if err != nil {
		return err
	}

	matchedPaths := []string{}

	// checking the files which are allowed by glob matching
	for _, path := range matchedPathsDir {
		matched, err := util.IsGlobExpMatch(path, globExps)
		if err != nil {
			return err
		}
		if !matched {
			matchedPaths = append(matchedPaths, path)
		}
	}

	// adding the files for taring
	for _, matchedPath := range matchedPaths {
		stat, err := os.Lstat(matchedPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := ioutil.ReadDir(matchedPath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := taro.FileInfoHeader(stat, matchedPath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := recursiveTar(srcBase, filepath.Join(srcFile, f.Name()), destBase, filepath.Join(destFile, f.Name()), tw, globExps); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := taro.FileInfoHeader(stat, joinedPath)
			target, err := os.Readlink(joinedPath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := taro.FileInfoHeader(stat, joinedPath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(joinedPath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}

// GetOneServiceFromSelector returns the Service object associated with the
// given selector.
// An error is thrown when exactly one Service is not found for the selector
func (c *Client) GetOneServiceFromSelector(selector string) (*corev1.Service, error) {
	services, err := c.GetServicesFromSelector(selector)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get services for the selector: %v", selector)
	}

	numServices := len(services)
	if numServices == 0 {
		return nil, fmt.Errorf("no Service was found for the selector: %v", selector)
	} else if numServices > 1 {
		return nil, fmt.Errorf("multiple Services exist for the selector: %v. Only one must be present", selector)
	}

	return &services[0], nil
}

// AddEnvironmentVariablesToDeployment adds the given environment
// variables to the only container in the Deployment and updates in the
// cluster
func (c *Client) AddEnvironmentVariablesToDeployment(envs []corev1.EnvVar, dep *appsv1.Deployment) error {
	numContainers := len(dep.Spec.Template.Spec.Containers)
	if numContainers != 1 {
		return fmt.Errorf("expected exactly one container in Deployment %v, got %v", dep.Name, numContainers)
	}

	dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, envs...)

	_, err := c.KubeClient.AppsV1().Deployments(c.Namespace).Update(dep)
	if err != nil {
		return errors.Wrapf(err, "unable to update Deployment %v", dep.Name)
	}
	return nil
}

// ServerInfo contains the fields that contain the server's information like
// address, OpenShift and Kubernetes versions
type ServerInfo struct {
	Address           string
	KubernetesVersion string
}

// GetServerVersion will fetch the Server Host, OpenShift and Kubernetes Version
// It will be shown on the execution of odo version command
func (c *Client) GetServerVersion() (*ServerInfo, error) {
	var info ServerInfo

	// This will fetch the information about Server Address
	config, err := c.KubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get server's address")
	}
	info.Address = config.Host

	// checking if the server is reachable
	if !isServerUp(config.Host) {
		return nil, errors.New("Unable to connect to OpenShift cluster, is it down?")
	}

	// This will fetch the information about Kubernetes Version
	rawKubernetesVersion, err := c.KubeClient.CoreV1().RESTClient().Get().AbsPath("/version").Do().Raw()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get Kubernetes Version")
	}
	var kubernetesVersion version.Info
	if err := json.Unmarshal(rawKubernetesVersion, &kubernetesVersion); err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal Kubernetes Version: %v", string(rawKubernetesVersion))
	}
	info.KubernetesVersion = kubernetesVersion.GitVersion

	return &info, nil
}

// ExecCMDInContainer execute command in first container of a pod
func (c *Client) ExecCMDInContainer(podName string, cmd []string, stdout io.Writer, stderr io.Writer, stdin io.Reader, tty bool) error {

	req := c.KubeClient.CoreV1().RESTClient().
		Post().
		Namespace(c.Namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   stdin != nil,
			Stdout:  stdout != nil,
			Stderr:  stderr != nil,
			TTY:     tty,
		}, scheme.ParameterCodec)

	config, err := c.KubeConfig.ClientConfig()
	if err != nil {
		return errors.Wrapf(err, "unable to get Kubernetes client config")
	}

	// Connect to url (constructed from req) using SPDY (HTTP/2) protocol which allows bidirectional streams.
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return errors.Wrapf(err, "unable execute command via SPDY")
	}
	// initialize the transport of the standard shell streams
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
	if err != nil {
		return errors.Wrapf(err, "error while streaming command")
	}

	return nil
}

// GetVolumeMountsFromDeployment returns a list of all volume mounts in the given Deployment
func (c *Client) GetVolumeMountsFromDeployment(dep *appsv1.Deployment) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	for _, container := range dep.Spec.Template.Spec.Containers {
		volumeMounts = append(volumeMounts, container.VolumeMounts...)
	}
	return volumeMounts
}

// IsVolumeAnEmptyDir returns true if the volume is an EmptyDir, false if not
func (c *Client) IsVolumeAnEmptyDir(volumeMountName string, dep *appsv1.Deployment) bool {
	for _, volume := range dep.Spec.Template.Spec.Volumes {
		if volume.Name == volumeMountName {
			if volume.EmptyDir != nil {
				return true
			}
		}
	}
	return false
}

// GetPVCNameFromVolumeMountName returns the PVC associated with the given volume
// An empty string is returned if the volume is not found
func (c *Client) GetPVCNameFromVolumeMountName(volumeMountName string, deployment *appsv1.Deployment) string {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == volumeMountName {
			if volume.PersistentVolumeClaim != nil {
				return volume.PersistentVolumeClaim.ClaimName
			}
		}
	}
	return ""
}

// GetPVCFromName returns the PVC of the given name
func (c *Client) GetPVCFromName(pvcName string) (*corev1.PersistentVolumeClaim, error) {
	return c.KubeClient.CoreV1().PersistentVolumeClaims(c.Namespace).Get(pvcName, metav1.GetOptions{})
}

// FindContainer finds the container
func FindContainer(containers []corev1.Container, name string) (corev1.Container, error) {

	if name == "" {
		return corev1.Container{}, errors.New("Invalid parameter for FindContainer, unable to find a blank container")
	}

	for _, container := range containers {
		if container.Name == name {
			return container, nil
		}
	}

	return corev1.Container{}, errors.New("Unable to find container")
}

// GetInputEnvVarsFromStrings generates corev1.EnvVar values from the array of string key=value pairs
// envVars is the array containing the key=value pairs
func GetInputEnvVarsFromStrings(envVars []string) ([]corev1.EnvVar, error) {
	var inputEnvVars []corev1.EnvVar
	var keys = make(map[string]int)
	for _, env := range envVars {
		splits := strings.SplitN(env, "=", 2)
		if len(splits) < 2 {
			return nil, errors.New("invalid syntax for env, please specify a VariableName=Value pair")
		}
		_, ok := keys[splits[0]]
		if ok {
			return nil, errors.Errorf("multiple values found for VariableName: %s", splits[0])
		}

		keys[splits[0]] = 1

		inputEnvVars = append(inputEnvVars, corev1.EnvVar{
			Name:  splits[0],
			Value: splits[1],
		})
	}
	return inputEnvVars, nil
}

// GetEnvVarsFromDep retrieves the env vars from the Deployment
// deploymentName is the name of the deployment from which the env vars are retrieved
// projectName is the name of the project
func (c *Client) GetEnvVarsFromDep(deploymentName string) ([]corev1.EnvVar, error) {
	deployment, err := c.GetDeploymentFromName(deploymentName)
	if err != nil {
		return nil, errors.Wrap(err, "error occurred while retrieving the deployment")
	}

	numContainers := len(deployment.Spec.Template.Spec.Containers)
	if numContainers != 1 {
		return nil, fmt.Errorf("expected exactly one container in Deployment %v, got %v", deployment.Name, numContainers)
	}

	return deployment.Spec.Template.Spec.Containers[0].Env, nil
}

// PropagateDeletes deletes the watch detected deleted files from remote component pod from each of the paths in passed targetPaths
// Parameters:
//	targetPodName: Name of component pod
//	delSrcRelPaths: Paths to be deleted on the remote pod relative to component source base path ex: Compoent src: /abc/src, file deleted: abc/src/foo.lang => relative path: foo.lang
//	targetPaths: Slice of all container paths that contain the component source -- deployment dir, destination dir, working dir, etc..
func (c *Client) PropagateDeletes(targetPodName string, delSrcRelPaths []string, containerPaths []string) error {
	reader, writer := io.Pipe()
	var rmPaths []string
	if len(containerPaths) == 0 || len(delSrcRelPaths) == 0 {
		return fmt.Errorf("Failed to propagate deletions: targetPaths: %+v and delSrcRelPaths: %+v", containerPaths, delSrcRelPaths)
	}
	for _, targetPath := range containerPaths {
		for _, delRelPath := range delSrcRelPaths {
			rmPaths = append(rmPaths, filepath.Join(targetPath, delRelPath))
		}
	}
	glog.V(4).Infof("targetPaths marked  for deletion are %+v", rmPaths)
	cmdArr := []string{"rm", "-rf"}
	cmdArr = append(cmdArr, rmPaths...)

	err := c.ExecCMDInContainer(targetPodName, cmdArr, writer, writer, reader, false)
	if err != nil {
		return err
	}
	return err
}

// RemoveVolumeFromDeployment removes the volume associated with the
// given PVC from the Deployment. Both, the volume entry and the
// volume mount entry in the containers, are deleted.
func (c *Client) RemoveVolumeFromDeployment(pvc string, depName string) error {

	retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {

		dep, err := c.GetDeploymentFromName(depName)
		if err != nil {
			return errors.Wrapf(err, "unable to get Deployment: %v", depName)
		}

		volumeNames := c.getVolumeNamesFromPVC(pvc, dep)
		numVolumes := len(volumeNames)
		if numVolumes == 0 {
			return fmt.Errorf("no volume found for PVC %v in Deployment %v, expected one", pvc, dep.Name)
		} else if numVolumes > 1 {
			return fmt.Errorf("found more than one volume for PVC %v in Deployment %v, expected one", pvc, dep.Name)
		}
		volumeName := volumeNames[0]

		// Remove volume if volume exists in Deployment
		if !removeVolumeFromDeployment(volumeName, dep) {
			return fmt.Errorf("could not find volume '%v' in Deployment '%v'", volumeName, dep.Name)
		}
		glog.V(4).Infof("Found volume: %v in Deployment: %v", volumeName, dep.Name)

		// Remove volume mount if volume mount exists
		if !removeVolumeFromDeployment(volumeName, dep) {
			return fmt.Errorf("could not find volumeMount: %v in Deployment: %v", volumeName, dep)
		}

		_, updateErr := c.KubeClient.AppsV1().Deployments(c.Namespace).Update(dep)
		return updateErr
	})
	if retryErr != nil {
		return errors.Wrapf(retryErr, "updating Deployment %v failed", depName)
	}
	return nil
}

func getAppRootVolumeName(depName string) string {
	return fmt.Sprintf("%s-idpdata", depName)
}
