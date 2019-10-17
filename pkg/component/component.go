package component

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	// "github.com/redhat-developer/odo-fork/pkg/catalog"
	"github.com/pkg/errors"

	applabels "github.com/redhat-developer/odo-fork/pkg/application/labels"
	"github.com/redhat-developer/odo-fork/pkg/catalog"
	componentlabels "github.com/redhat-developer/odo-fork/pkg/component/labels"

	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/idp"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/kdo/util/validation"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/preference"
	"github.com/redhat-developer/odo-fork/pkg/storage"

	// "github.com/redhat-developer/odo-fork/pkg/storage"
	urlpkg "github.com/redhat-developer/odo-fork/pkg/url"
	"github.com/redhat-developer/odo-fork/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// componentSourceURLAnnotation is an source url from which component was build
// it can be also file://
const componentSourceURLAnnotation = "app.kubernetes.io/url"
const ComponentSourceTypeAnnotation = "app.kubernetes.io/component-source-type"
const componentRandomNamePartsMaxLen = 12
const componentNameMaxRetries = 3
const componentNameMaxLen = -1

// Target defines a target image environment which can be based on an IDP or s2i image
type ContainerAttributes struct {
	// Path is the location to copy the source
	SrcPath      string
	WorkingPaths []string
}

// GetComponentDir returns source repo name
// Parameters:
//		path: git url or source path or binary path
//		paramType: One of CreateType as in GIT/LOCAL/BINARY
// Returns: directory name
func GetComponentDir(path string, paramType config.SrcType) (string, error) {
	retVal := ""
	switch paramType {
	case config.GIT:
		retVal = strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".git")
	case config.LOCAL:
		retVal = filepath.Base(path)
	case config.BINARY:
		filename := filepath.Base(path)
		var extension = filepath.Ext(filename)
		retVal = filename[0 : len(filename)-len(extension)]
	default:
		currDir, err := os.Getwd()
		if err != nil {
			return "", errors.Wrapf(err, "unable to generate a random name as getting current directory failed")
		}
		retVal = filepath.Base(currDir)
	}
	retVal = strings.TrimSpace(util.GetDNS1123Name(strings.ToLower(retVal)))
	return retVal, nil
}

//BuildTask is a struct of essential data
type BuildTask struct {
	UseRuntime         bool
	Kind               string
	Name               string
	Image              string
	ContainerName      string
	PodName            string
	Namespace          string
	WorkspaceID        string
	ServiceAccountName string
	PullSecret         string
	OwnerReferenceName string
	OwnerReferenceUID  types.UID
	Privileged         bool
	Ingress            string
	PVCName            []string
	MountPath          []string
	SubPath            []string
	Labels             map[string]string
	Command            []string
	SrcDestination     string
}

// GetDefaultComponentName generates a unique component name
// Parameters: desired default component name(w/o prefix) and slice of existing component names
// Returns: Unique component name and error if any
func GetDefaultComponentName(componentPath string, componentPathType config.SrcType, componentType string, existingComponentList ComponentList) (string, error) {
	var prefix string

	// Get component names from component list
	var existingComponentNames []string
	for _, component := range existingComponentList.Items {
		existingComponentNames = append(existingComponentNames, component.Name)
	}

	// Fetch config
	cfg, err := preference.New()
	if err != nil {
		return "", errors.Wrap(err, "unable to generate random component name")
	}

	// If there's no prefix in config file, or its value is empty string use safe default - the current directory along with component type
	if cfg.OdoSettings.NamePrefix == nil || *cfg.OdoSettings.NamePrefix == "" {
		prefix, err = GetComponentDir(componentPath, componentPathType)
		if err != nil {
			return "", errors.Wrap(err, "unable to generate random component name")
		}
		prefix = util.TruncateString(prefix, componentRandomNamePartsMaxLen)
	} else {
		// Set the required prefix into componentName
		prefix = *cfg.OdoSettings.NamePrefix
	}

	// Generate unique name for the component using prefix and unique random suffix
	componentName, err := util.GetRandomName(
		fmt.Sprintf("%s-%s", componentType, prefix),
		componentNameMaxLen,
		existingComponentNames,
		componentNameMaxRetries,
	)
	if err != nil {
		return "", errors.Wrap(err, "unable to generate random component name")
	}

	return util.GetDNS1123Name(componentName), nil
}

// validateSourceType check if given sourceType is supported
func validateSourceType(sourceType string) bool {
	validSourceTypes := []string{
		"git",
		"local",
		"binary",
	}
	for _, valid := range validSourceTypes {
		if valid == sourceType {
			return true
		}
	}
	return false
}

// // CreateFromGit inputPorts is the array containing the string port values
// // inputPorts is the array containing the string port values
// // envVars is the array containing the environment variables
// func CreateFromGit(client *kclient.Client, params kclient.CreateArgs) error {

// 	labels := componentlabels.GetLabels(params.Name, params.ApplicationName, true)

// 	// Parse componentImageType before adding to labels
// 	_, imageName, imageTag, _, err := kclient.ParseImageName(params.ImageName)
// 	if err != nil {
// 		return errors.Wrap(err, "unable to parse image name")
// 	}

// 	// save component type as label
// 	labels[componentlabels.ComponentTypeLabel] = imageName
// 	labels[componentlabels.ComponentTypeVersion] = imageTag

// 	// save source path as annotation
// 	annotations := map[string]string{componentSourceURLAnnotation: params.SourcePath}
// 	annotations[ComponentSourceTypeAnnotation] = "git"

// 	// Namespace the component
// 	namespacedOpenShiftObject, err := util.NamespaceOpenShiftObject(params.Name, params.ApplicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create namespaced name")
// 	}

// 	// Create CommonObjectMeta to be passed in
// 	commonObjectMeta := metav1.ObjectMeta{
// 		Name:        namespacedOpenShiftObject,
// 		Labels:      labels,
// 		Annotations: annotations,
// 	}

// 	err = client.NewAppS2I(params, commonObjectMeta)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create git component %s", namespacedOpenShiftObject)
// 	}

// 	// Trigger build
// 	if err = Build(client, params.Name, params.ApplicationName, params.Wait, params.StdOut, false); err != nil {
// 		return errors.Wrapf(err, "failed to build component with args %+v", params)
// 	}

// 	// deploy the component and wait for it to complete
// 	// desiredRevision is 1 as this is the first push
// 	if err = Deploy(client, params, 1); err != nil {
// 		return errors.Wrapf(err, "failed to deploy component with args %+v", params)
// 	}

// 	return nil
// }

// // GetComponentPorts provides slice of ports used by the component in the form port_no/protocol
// func GetComponentPorts(client *kclient.Client, componentName string, applicationName string) (ports []string, err error) {
// 	componentLabels := componentlabels.GetLabels(componentName, applicationName, false)
// 	componentSelector := util.ConvertLabelsToSelector(componentLabels)

// 	dc, err := client.GetOneDeploymentConfigFromSelector(componentSelector)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "unable to fetch deployment configs for the selector %v", componentSelector)
// 	}

// 	for _, container := range dc.Spec.Template.Spec.Containers {
// 		for _, port := range container.Ports {
// 			ports = append(ports, fmt.Sprintf("%v/%v", port.ContainerPort, port.Protocol))
// 		}
// 	}

// 	return ports, nil
// }

// GetComponentLinkedSecretNames provides a slice containing the names of the secrets that are present in envFrom
func GetComponentLinkedSecretNames(client *kclient.Client, componentName string, applicationName string) (secretNames []string, err error) {
	componentLabels := componentlabels.GetLabels(componentName, applicationName, false)
	componentSelector := util.ConvertLabelsToSelector(componentLabels)

	dc, err := client.GetOneDeploymentFromSelector(componentSelector)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch deployment for the selector %v", componentSelector)
	}

	for _, env := range dc.Spec.Template.Spec.Containers[0].EnvFrom {
		if env.SecretRef != nil {
			secretNames = append(secretNames, env.SecretRef.Name)
		}
	}

	return secretNames, nil
}

// CreateFromPath create new component with source or binary from the given local path
// sourceType indicates the source type of the component and can be either local or binary
// envVars is the array containing the environment variables
func (b *BuildTask) CreateFromPath(client *kclient.Client, params kclient.CreateArgs, devPack *idp.IDP) error {
	labels := componentlabels.GetLabels(params.Name, params.ApplicationName, true)

	// Parse componentImageType before adding to labels
	_, imageName, imageTag, _, err := kclient.ParseImageName(params.ImageName)
	if err != nil {
		return errors.Wrap(err, "unable to parse image name")
	}

	// save component type as label
	labels[componentlabels.ComponentTypeLabel] = imageName
	labels[componentlabels.ComponentTypeVersion] = imageTag

	// save source path as annotation
	sourceURL := util.GenFileURL(params.SourcePath)
	annotations := map[string]string{componentSourceURLAnnotation: sourceURL}
	annotations[ComponentSourceTypeAnnotation] = string(params.SourceType)

	// Namespace the component
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(params.Name, params.ApplicationName)
	if err != nil {
		return errors.Wrapf(err, "unable to create namespaced name")
	}

	// Create CommonObjectMeta to be passed in
	commonObjectMeta := metav1.ObjectMeta{
		Name:        namespacedKubernetesObject,
		Labels:      labels,
		Annotations: annotations,
	}

	// Create component resources
	err = client.CreateComponentResources(params, commonObjectMeta)
	if err != nil {
		return err
	}

	return nil
}

// Delete whole component
func Delete(client *kclient.Client, componentName string, applicationName string) error {

	// Loading spinner
	s := log.Spinnerf("Deleting component %s", componentName)
	defer s.End(false)

	labels := componentlabels.GetLabels(componentName, applicationName, false)
	err := client.Delete(labels)
	if err != nil {
		return errors.Wrapf(err, "error deleting component %s", componentName)
	}

	s.End(true)
	return nil
}

// // getEnvFromPodEnvs loops through the passed slice of pod#EnvVars and gets the value corresponding to the key passed, returns empty stirng if not available
// func getEnvFromPodEnvs(envName string, podEnvs []corev1.EnvVar) string {
// 	for _, podEnv := range podEnvs {
// 		if podEnv.Name == envName {
// 			return podEnv.Value
// 		}
// 	}
// 	return ""
// }

// // getS2IPaths returns slice of s2i paths of odo interest
// // Parameters:
// //	podEnvs: Slice of env vars extracted from pod template
// // Returns:
// //	Slice of s2i paths extracted from passed parameters
// func getS2IPaths(podEnvs []corev1.EnvVar) []string {
// 	retVal := []string{}
// 	// List of s2i Paths exported for use in container pod for working with source/binary
// 	s2iPathEnvs := []string{
// 		kclient.EnvS2IDeploymentDir,
// 		kclient.EnvS2ISrcOrBinPath,
// 		kclient.EnvS2IWorkingDir,
// 		kclient.EnvS2ISrcBackupDir,
// 	}
// 	// For each of the required env var
// 	for _, s2iPathEnv := range s2iPathEnvs {
// 		// try to fetch the value of required env from the ones set already in the component container like for the case of watch or multiple pushes
// 		envVal := getEnvFromPodEnvs(s2iPathEnv, podEnvs)
// 		isEnvValPresent := false
// 		if envVal != "" {
// 			for _, e := range retVal {
// 				if envVal == e {
// 					isEnvValPresent = true
// 					break
// 				}
// 			}
// 			if !isEnvValPresent {
// 				// If `src` not in path, append it
// 				if filepath.Base(envVal) != "src" {
// 					envVal = filepath.Join(envVal, "src")
// 				}
// 				retVal = append(retVal, envVal)
// 			}
// 		}
// 	}
// 	// Append binary backup path to s2i paths list
// 	retVal = append(retVal, kclient.DefaultS2IDeploymentBackupDir)
// 	return retVal
// }

// CreateComponent creates component as per the passed component settings
//	Parameters:
//		client: kclient instance
//		componentConfig: the component configuration that holds all details of component
//		context: the component context indicating the location of component config and hence its source as well
//		stdout: io.Writer instance to write output to
//	Returns:
//		err: errors if any
func (b *BuildTask) CreateComponent(client *kclient.Client, componentConfig config.LocalConfigInfo, devPack *idp.IDP, pvc []*corev1.PersistentVolumeClaim) (err error) {

	cmpName := componentConfig.GetName()
	// cmpType := componentConfig.GetType()
	cmpSrcType := componentConfig.GetSourceType()
	cmpPorts := componentConfig.GetPorts()
	appName := componentConfig.GetApplication()
	envVarsList := componentConfig.GetEnvVars()

	// create and get the storage to be created/mounted during the component creation
	// storageList := getStorageFromConfig(&componentConfig)
	// storageToBeMounted, _, err := storage.Push(client, storageList, componentConfig.GetName(), componentConfig.GetApplication(), false)

	// TODO-KDO: remove following line and implement storage handling properly for KDO
	imgName := devPack.Spec.Runtime.Image
	log.Successf("Initializing component")
	createArgs := kclient.CreateArgs{
		Name: cmpName,
		// ImageName:          cmpType,
		ImageName:       imgName,
		ApplicationName: appName,
		EnvVars:         envVarsList.ToStringSlice(),
		UseRunTime:      b.UseRuntime,
	}
	createArgs.SourceType = cmpSrcType
	createArgs.SourcePath = componentConfig.GetSourceLocation()

	if b.UseRuntime {
		glog.V(0).Info("MJF yes it is runtime")
		storageToBeMounted := make(map[string]*corev1.PersistentVolumeClaim)
		for i := range b.PVCName {
			storageToBeMounted[b.MountPath[i]+"#"+b.SubPath[i]] = pvc[i]
		}
		createArgs.StorageToBeMounted = storageToBeMounted
	}

	// If the user overrides ports in the udo config, set them as the component's ports instead (Rather than what the IDP specified)
	if len(cmpPorts) > 0 {
		createArgs.Ports = cmpPorts
	} else {
		createArgs.Ports = devPack.GetPorts()
	}

	createArgs.Resources, err = kclient.GetResourceRequirementsFromCmpSettings(componentConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create component")
	}

	switch cmpSrcType {
	// TODO-KDO: Decide whether to implement create component from git, possibly use tekton pipeline for this scenario
	// case config.GIT:
	// 	// Use Git
	// 	if cmpSrcRef != "" {
	// 		createArgs.SourceRef = cmpSrcRef
	// 	}

	// 	createArgs.Wait = true
	// 	createArgs.StdOut = stdout

	// 	if err = CreateFromGit(
	// 		client,
	// 		createArgs,
	// 	); err != nil {
	// 		return errors.Wrapf(err, "failed to create component with args %+v", createArgs)
	// 	}
	case config.LOCAL:
		fileInfo, err := os.Stat(createArgs.SourcePath)
		if err != nil {
			return errors.Wrapf(err, "failed to get info of path %+v of component %+v", createArgs.SourcePath, createArgs)
		}
		if !fileInfo.IsDir() {
			return fmt.Errorf("component creation with args %+v as path needs to be a directory", createArgs)
		}
		// Create
		if err = b.CreateFromPath(client, createArgs, devPack); err != nil {
			return errors.Wrapf(err, "failed to create component with args %+v", createArgs)
		}
	case config.BINARY:
		if err = b.CreateFromPath(client, createArgs, devPack); err != nil {
			return errors.Wrapf(err, "failed to create component with args %+v", createArgs)
		}
	default:
		// If the user does not provide anything (local, git or binary), use the current absolute path and deploy it
		createArgs.SourceType = config.LOCAL
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to create component with current directory as source for the component")
		}
		createArgs.SourcePath = dir
		if err = b.CreateFromPath(client, createArgs, devPack); err != nil {
			return errors.Wrapf(err, "")
		}
	}
	return
}

// CheckComponentMandatoryParams checks mandatory parammeters for component
func CheckComponentMandatoryParams(componentSettings config.ComponentSettings) error {
	var req_fields string

	if componentSettings.Name == nil {
		req_fields = fmt.Sprintf("%s name", req_fields)
	}

	if componentSettings.Application == nil {
		req_fields = fmt.Sprintf("%s application", req_fields)
	}

	if componentSettings.Project == nil {
		req_fields = fmt.Sprintf("%s project name", req_fields)
	}

	if componentSettings.SourceType == nil {
		req_fields = fmt.Sprintf("%s source type", req_fields)
	}

	if componentSettings.SourceLocation == nil {
		req_fields = fmt.Sprintf("%s source location", req_fields)
	}

	if componentSettings.Type == nil {
		req_fields = fmt.Sprintf("%s type", req_fields)
	}

	if len(req_fields) > 0 {
		return fmt.Errorf("missing mandatory parameters:%s", req_fields)
	}
	return nil
}

// ValidateComponentCreateRequest validates request for component creation and returns errors if any
// Parameters:
//	componentSettings: Component settings
//	isCmpExistsCheck: boolean to indicate whether or not error out if component with same name already exists
// Returns:
//	errors if any
func ValidateComponentCreateRequest(client *kclient.Client, componentSettings config.ComponentSettings, isCmpExistsCheck bool, localIndexJson string) (err error) {

	// Check the mandatory parameters first
	err = CheckComponentMandatoryParams(componentSettings)
	if err != nil {
		return err
	}

	// Parse the image name
	// _, componentType, _, componentVersion := util.ParseComponentImageName(*componentSettings.Type)
	_, componentType, _, _ := util.ParseComponentImageName(*componentSettings.Type)

	// Check to see if the catalog type actually exists
	exists, err := catalog.Exists(componentType, localIndexJson)
	if err != nil {
		return errors.Wrapf(err, "Failed to create component of type %s", componentType)
	}
	if !exists {
		log.Info("Run 'udo catalog list idp' for a list of supported Iterative-Dev packs")
		return fmt.Errorf("Failed to find component of type %s", componentType)
	}

	// Check to see if that particular version exists
	// versionExists, err := catalog.VersionExists(client, componentType, componentVersion)
	// if err != nil {
	// 	return errors.Wrapf(err, "Failed to create component of type %s of version %s", componentType, componentVersion)
	// }
	// if !versionExists {
	// 	log.Info("Run 'udo catalog list idp' to see a list of supported component type versions")
	// 	return fmt.Errorf("Invalid component version %s:%s", componentType, componentVersion)
	// }

	// Validate component name
	err = validation.ValidateName(*componentSettings.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to create component of name %s", *componentSettings.Name)
	}

	// If component does not exist, create it
	if isCmpExistsCheck {
		exists, err = Exists(client, *componentSettings.Name, *componentSettings.Application)
		if err != nil {
			return errors.Wrapf(err, "failed to check if component of name %s exists in application %s", *componentSettings.Name, *componentSettings.Application)
		}
		if exists {
			return fmt.Errorf("component with name %s already exists in application %s", *componentSettings.Name, *componentSettings.Application)
		}
	}

	// If component is of type local, check if the source path is valid
	if *componentSettings.SourceType == config.LOCAL {
		glog.V(4).Infof("Checking source location: %s", *(componentSettings.SourceLocation))
		srcLocInfo, err := os.Stat(*(componentSettings.SourceLocation))
		if err != nil {
			return errors.Wrap(err, "failed to create component. Please view the settings used using the command `odo config view`")
		}
		if !srcLocInfo.IsDir() {
			return fmt.Errorf("source path for component created for local source needs to be a directory")
		}
	}

	return
}

// ApplyConfig applies the component config onto component dc
// Parameters:
//	client: kclient instance
//	appName: Name of application of which the component is a part
//	componentName: Name of the component which is being patched with config
//	componentConfig: Component configuration
//  	cmpExist: true if components exists in the cluster
// Returns:
//	err: Errors if any else nil
func ApplyConfig(client *kclient.Client, componentConfig config.LocalConfigInfo, stdout io.Writer) (err error) {

	// s := log.Spinner("Applying configuration")
	// defer s.End(false)
	// // if component exist then only call the update function
	// if cmpExist {

	// 	if err = Update(client, componentConfig, componentConfig.GetSourceLocation(), stdout); err != nil {
	// 		return err
	// 	}
	// }
	// s.End(true)

	showChanges, err := checkIfURLChangesWillBeMade(client, componentConfig)
	if err != nil {
		return err
	}

	if showChanges {
		log.Info("\nApplying URL changes")
		// Create any URLs that have been added to the component
		err = ApplyConfigCreateURL(client, componentConfig)
		if err != nil {
			return err
		}

		// Delete any URLs
		err = applyConfigDeleteURL(client, componentConfig)
		if err != nil {
			return err
		}
	}

	return
}

// ApplyConfigDeleteURL applies url config deletion onto component
func applyConfigDeleteURL(client *kclient.Client, componentConfig config.LocalConfigInfo) (err error) {

	urlList, err := urlpkg.List(client, componentConfig.GetName(), componentConfig.GetApplication())
	if err != nil {
		return err
	}
	localUrlList := componentConfig.GetUrl()
	for _, u := range urlList.Items {
		if !checkIfUrlPresentInConfig(localUrlList, u.Name) {
			err = urlpkg.Delete(client, u.Name, componentConfig.GetApplication())
			if err != nil {
				return err
			}
			log.Successf("URL %s successfully deleted", u.Name)
		}
	}
	return nil
}

func checkIfUrlPresentInConfig(localUrl []config.ConfigUrl, url string) bool {
	for _, u := range localUrl {
		if u.Name == url {
			return true
		}
	}
	return false
}

// ApplyConfigCreateURL applies url config onto component
func ApplyConfigCreateURL(client *kclient.Client, componentConfig config.LocalConfigInfo) error {

	urls := componentConfig.GetUrl()
	for _, urlo := range urls {
		exist, err := urlpkg.Exists(client, urlo.Name, componentConfig.GetName(), componentConfig.GetApplication())
		if err != nil {
			return errors.Wrapf(err, "unable to check url")
		}
		if exist {
			log.Successf("URL %s already exists", urlo.Name)
		} else {
			host, err := urlpkg.Create(client, urlo.Name, urlo.Port, urlo.Host, componentConfig.GetName(), componentConfig.GetApplication())
			if err != nil {
				return errors.Wrapf(err, "unable to create url")
			}
			log.Successf("URL %s: %s created", urlo.Name, host)
		}
	}

	return nil
}

// PushLocal push local code to the cluster and trigger build there.
// During copying binary components, path represent base directory path to binary and files contains path of binary
// During copying local source components, path represent base directory path whereas files is empty
// During `odo watch`, path represent base directory path whereas files contains list of changed Files
// Parameters:
//	componentName is name of the component to update sources to
//	applicationName is the name of the application of which the component is a part
//	path is base path of the component source/binary
// 	files is list of changed files captured during `odo watch` as well as binary file path
// 	delFiles is the list of files identified as deleted
// 	isForcePush indicates if the sources to be updated are due to a push in which case its a full source directory push or only push of identified sources
// 	globExps are the glob expressions which are to be ignored during the push
//	show determines whether or not to show the log (passed in by po.show argument within /cmd)
// Returns
//	Error if any
func PushLocal(client *kclient.Client, componentName string, applicationName string, path string, out io.Writer, files []string, delFiles []string, isForcePush bool, globExps []string, show bool, containerAttr ContainerAttributes) error {
	glog.V(4).Infof("PushLocal: componentName: %s, applicationName: %s, path: %s, files: %s, delFiles: %s, isForcePush: %+v", componentName, applicationName, path, files, delFiles, isForcePush)

	// Edge case: check to see that the path is NOT empty.
	emptyDir, err := isEmpty(path)
	if err != nil {
		return errors.Wrapf(err, "Unable to check directory: %s", path)
	} else if emptyDir {
		return errors.New(fmt.Sprintf("Directory / file %s is empty", path))
	}

	// Find Deployment for component
	componentLabels := componentlabels.GetLabels(componentName, applicationName, false)
	componentSelector := util.ConvertLabelsToSelector(componentLabels)
	dep, err := client.GetOneDeploymentFromSelector(componentSelector)
	if err != nil {
		return errors.Wrap(err, "unable to get deployment for component")
	}
	// Find Pod for component
	podSelector := fmt.Sprintf("deployment=%s", dep.Name)
	watchOptions := metav1.ListOptions{
		LabelSelector: podSelector,
	}

	// Wait for Pod to be in running state otherwise we can't sync data to it.
	pod, err := client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start")
	if err != nil {
		return errors.Wrapf(err, "error while waiting for pod  %s", podSelector)
	}

	// If there are files identified as deleted, propagate them to the component pod
	if len(delFiles) > 0 {
		glog.V(4).Infof("propogating deletion of files %s to pod", strings.Join(delFiles, " "))
		/*
			Delete files observed by watch from each of the directories in the specified pod. The directories are an array because the source can
			be copied to more than one directory depending on the build controller. Eg. For s2i the following directories can contain the source:
				deployment dir: In interpreted runtimes like python, source is copied over to deployment dir so delete needs to happen here as well
				destination dir: This is the directory where s2i expects source to be copied for it be built and deployed
				working dir: Directory where, sources are copied over from deployment dir from where the s2i builds and deploys source.
							 Deletes need to happen here as well otherwise, even if the latest source is copied over, the stale source files remain
				source backup dir: Directory used for backing up source across multiple iterations of push and watch in component container
								   In case of python, s2i image moves sources from destination dir to workingdir which means sources are deleted from destination dir
								   So, during the subsequent watch pushing new diff to component pod, the source as a whole doesn't exist at destination dir and hence needs
								   to be backed up.
		*/
		err := client.PropagateDeletes(pod.Name, delFiles, containerAttr.WorkingPaths)
		if err != nil {
			return errors.Wrapf(err, "unable to propagate file deletions %+v", delFiles)
		}
	}

	// Copy the files to the pod
	s := log.Spinner("Copying files to component")
	defer s.End(false)

	if !isForcePush {
		if len(files) == 0 && len(delFiles) == 0 {
			return fmt.Errorf("pass files modifications/deletions to sync to component pod or force push")
		}
	}

	if isForcePush || len(files) > 0 {
		glog.V(4).Infof("Copying files %s to pod", strings.Join(files, " "))
		err = client.CopyFile(path, pod.Name, containerAttr.SrcPath, files, globExps)
		if err != nil {
			s.End(false)
			return errors.Wrap(err, "unable push files to pod")
		}
	}
	s.End(true)

	// TODO-KDO
	// Implement once we've added the build jobs/tasks to KDO
	// IMO, this should be in a separate function from `PushLocal()`
	/*if show {
		s = log.SpinnerNoSpin("Building component")
	} else {
		s = log.Spinner("Building component")
	}

	// use pipes to write output from ExecCMDInContainer in yellow  to 'out' io.Writer
	pipeReader, pipeWriter := io.Pipe()
	var cmdOutput string

	// This Go routine will automatically pipe the output from ExecCMDInContainer to
	// our logger.
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()

			if log.IsDebug() || show {
				_, err := fmt.Fprintln(out, line)
				if err != nil {
					log.Errorf("Unable to print to stdout: %v", err)
				}
			}

			cmdOutput += fmt.Sprintln(line)
		}
	}()

	err = client.ExecCMDInContainer(pod.Name,
		// We will use the assemble-and-restart script located within the supervisord container we've created
		[]string{"/var/lib/supervisord/bin/assemble-and-restart"},
		pipeWriter, pipeWriter, nil, false)

	if err != nil {
		// If we fail, log the output
		log.Errorf("Unable to build files\n%v", cmdOutput)
		s.End(false)
		return errors.Wrap(err, "unable to execute assemble script")
	}

	s.End(true)*/

	return nil
}

// // Build component from BuildConfig.
// // If 'wait' is true than it waits for build to successfully complete.
// // If 'wait' is false than this function won't return error even if build failed.
// // 'show' will determine whether or not the log will be shown to the user (while building)
// func Build(client *kclient.Client, componentName string, applicationName string, wait bool, stdout io.Writer, show bool) error {

// 	// Loading spinner
// 	// No loading spinner if we're showing the logging output
// 	s := log.Spinnerf("Triggering build from git")
// 	defer s.End(false)

// 	// Namespace the component
// 	namespacedOpenShiftObject, err := util.NamespaceOpenShiftObject(componentName, applicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create namespaced name")
// 	}

// 	buildName, err := client.StartBuild(namespacedOpenShiftObject)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to rebuild %s", componentName)
// 	}
// 	s.End(true)

// 	// Retrieve the Build Log and write to buffer if debug is disabled, else we we output to stdout / debug.

// 	var b bytes.Buffer
// 	if !log.IsDebug() && !show {
// 		stdout = bufio.NewWriter(&b)
// 	}

// 	if wait {

// 		if show {
// 			s = log.SpinnerNoSpin("Waiting for build to finish")
// 		} else {
// 			s = log.Spinner("Waiting for build to finish")
// 		}

// 		defer s.End(false)
// 		if err := client.FollowBuildLog(buildName, stdout); err != nil {
// 			return errors.Wrapf(err, "unable to follow logs for %s", buildName)
// 		}

// 		if err := client.WaitForBuildToFinish(buildName); err != nil {
// 			return errors.Wrapf(err, "unable to build %s, error: %s", buildName, b.String())
// 		}
// 		s.End(true)
// 	}

// 	return nil
// }

// // Deploy deploys the component
// // it starts a new deployment and wait for the new dc to be available
// // desiredRevision is the desired version of the deployment config to wait for
// func Deploy(client *kclient.Client, params kclient.CreateArgs, desiredRevision int64) error {

// 	// Loading spinner
// 	s := log.Spinnerf("Deploying component %s", params.Name)
// 	defer s.End(false)

// 	// Namespace the component
// 	namespacedKubernetesObject, err := util.NamespacedKubernetesObject(params.Name, params.ApplicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create namespaced name")
// 	}

// 	// start the deployment
// 	// the build must be finished before this call and the new image must be successfully updated
// 	_, err = client.StartDeployment(namespacedKubernetesObject)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create Deployment for %s", namespacedKubernetesObject)
// 	}

// 	// Watch / wait for deployment to update annotations
// 	_, err = client.WaitAndGetDC(namespacedKubernetesObject, desiredRevision, kclient.OcUpdateTimeout, kclient.IsDCRolledOut)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to wait for Deployment %s to update", namespacedKubernetesObject)
// 	}

// 	s.End(true)

// 	return nil
// }

// GetComponentType returns type of component in given application and project
func GetComponentType(client *kclient.Client, componentName string, applicationName string) (string, error) {

	// filter according to component and application name
	selector := fmt.Sprintf("%s=%s,%s=%s", componentlabels.ComponentLabel, componentName, applabels.ApplicationLabel, applicationName)
	componentImageTypes, err := client.GetDeploymentLabelValues(componentlabels.ComponentTypeLabel, selector)
	if err != nil {
		return "", errors.Wrapf(err, "unable to get type of %s component", componentName)
	}
	if len(componentImageTypes) < 1 {
		// no type returned
		return "", errors.Wrapf(err, "unable to find type of %s component", componentName)

	}
	// check if all types are the same
	// it should be as we are secting only exactly one component, and it doesn't make sense
	// to have one component labeled with different component type labels
	for _, componentImageType := range componentImageTypes {
		if componentImageTypes[0] != componentImageType {
			return "", errors.Wrap(err, "data mismatch: %s component has objects with different types")
		}

	}
	return componentImageTypes[0], nil
}

// // List lists components in active application
func List(client *kclient.Client, applicationName string) (ComponentList, error) {

	var applicationSelector string
	if applicationName != "" {
		applicationSelector = fmt.Sprintf("%s=%s", applabels.ApplicationLabel, applicationName)
	}

	// retrieve all the deployment that are associated with this application
	dcList, err := client.GetDeploymentsFromSelector(applicationSelector)
	if err != nil {
		return ComponentList{}, errors.Wrapf(err, "unable to list components")
	}

	var components []Component

	// extract the labels we care about from each component
	for _, elem := range dcList {
		component, err := GetComponent(client, elem.Labels[componentlabels.ComponentLabel], applicationName, client.Namespace)
		if err != nil {
			return ComponentList{}, errors.Wrap(err, "Unable to get component")
		}
		component.Status.State = "Pushed"
		components = append(components, component)

	}

	compoList := GetMachineReadableFormatForList(components)
	return compoList, nil
}

// // ListIfPathGiven lists all available component in given path directory
// func ListIfPathGiven(client *kclient.Client, paths []string) (ComponentList, error) {
// 	var components []Component
// 	var err error
// 	for _, path := range paths {
// 		err = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
// 			if f != nil && strings.Contains(f.Name(), ".udo") {
// 				data, err := config.NewLocalConfigInfo(filepath.Dir(path))
// 				if err != nil {
// 					return err
// 				}
// 				exist, err := Exists(client, data.GetName(), data.GetApplication())
// 				if err != nil {
// 					return err
// 				}
// 				con, _ := filepath.Abs(filepath.Dir(path))
// 				a := getMachineReadableFormat(data.GetName(), data.GetType())
// 				a.Namespace = data.GetProject()
// 				a.Spec.App = data.GetApplication()
// 				a.Spec.Source = data.GetSourceLocation()
// 				a.Spec.Ports = data.GetPorts()
// 				a.Status.Context = con
// 				state := "Not Pushed"
// 				if exist {
// 					state = "Pushed"
// 				}
// 				a.Status.State = state
// 				components = append(components, a)
// 			}
// 			return nil
// 		})

// 	}
// 	return GetMachineReadableFormatForList(components), err
// }

// GetComponentSource what source type given component uses
// The first returned string is component source type ("git" or "local" or "binary")
// The second returned string is a source (url to git repository or local path or path to binary)
// we retrieve the source type by looking up the Deployment that's deployed
func GetComponentSource(client *kclient.Client, componentName string, applicationName string) (string, string, error) {

	// Namespace the application
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(componentName, applicationName)
	if err != nil {
		return "", "", errors.Wrapf(err, "unable to create namespaced name")
	}

	deployment, err := client.GetDeploymentFromName(namespacedKubernetesObject)
	if err != nil {
		return "", "", errors.Wrapf(err, "unable to get source path for component %s", componentName)
	}

	sourcePath := deployment.ObjectMeta.Annotations[componentSourceURLAnnotation]
	sourceType := deployment.ObjectMeta.Annotations[ComponentSourceTypeAnnotation]

	if !validateSourceType(sourceType) {
		return "", "", fmt.Errorf("unsupported component source type %s", sourceType)
	}

	glog.V(4).Infof("Source for component %s is %s (%s)", componentName, sourcePath, sourceType)
	return sourceType, sourcePath, nil
}

// // Update updates the requested component
// // Parameters:
// //	client: kclient instance
// //	componentSettings: Component configuration
// //	newSource: Location of component source resolved to absolute path
// //	stdout: io pipe to write logs to
// // Returns:
// //	errors if any
// func Update(client *kclient.Client, componentSettings config.LocalConfigInfo, newSource string, stdout io.Writer) error {

// 	// STEP 1. Create the common Object Meta for updating.

// 	componentName := componentSettings.GetName()
// 	applicationName := componentSettings.GetApplication()
// 	newSourceType := componentSettings.GetSourceType()
// 	newSourceRef := componentSettings.GetRef()
// 	componentImageType := componentSettings.GetType()
// 	cmpPorts := componentSettings.GetPorts()
// 	envVarsList := componentSettings.GetEnvVars()

// 	// retrieve the list of storages to create/mount and unmount
// 	storageList := getStorageFromConfig(&componentSettings)
// 	storageToMount, storageToUnMount, err := storage.Push(client, storageList, componentSettings.GetName(), componentSettings.GetApplication(), true)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to get storage to mount and unmount")
// 	}

// 	// Retrieve the old source type
// 	oldSourceType, _, err := GetComponentSource(client, componentName, applicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to get source of %s component", componentName)
// 	}

// 	// Namespace the application
// 	namespacedOpenShiftObject, err := util.NamespaceOpenShiftObject(componentName, applicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create namespaced name")
// 	}

// 	// Create annotations
// 	annotations := map[string]string{componentSourceURLAnnotation: newSource}
// 	annotations[ComponentSourceTypeAnnotation] = string(newSourceType)

// 	// Parse componentImageType before adding to labels
// 	imageNS, imageName, imageTag, _, err := kclient.ParseImageName(componentImageType)
// 	if err != nil {
// 		return errors.Wrap(err, "unable to parse image name")
// 	}

// 	// Retrieve labels
// 	// Save component type as label
// 	labels := componentlabels.GetLabels(componentName, applicationName, true)
// 	labels[componentlabels.ComponentTypeLabel] = imageName
// 	labels[componentlabels.ComponentTypeVersion] = imageTag

// 	// ObjectMetadata are the same for all generated objects
// 	// Create common metadata that will be updated throughout all objects.
// 	commonObjectMeta := metav1.ObjectMeta{
// 		Name:        namespacedOpenShiftObject,
// 		Labels:      labels,
// 		Annotations: annotations,
// 	}

// 	// Retrieve the current DC in order to obtain what the current inputPorts are..
// 	currentDC, err := client.GetDeploymentFromName(commonObjectMeta.Name)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to get DeploymentConfig %s", commonObjectMeta.Name)
// 	}

// 	foundCurrentDCContainer, err := kclient.FindContainer(currentDC.Spec.Template.Spec.Containers, commonObjectMeta.Name)
// 	if err != nil {
// 		return errors.Wrapf(err, "Unable to find container %s", commonObjectMeta.Name)
// 	}

// 	ports := foundCurrentDCContainer.Ports
// 	if len(cmpPorts) > 0 {
// 		ports, err = util.GetContainerPortsFromStrings(cmpPorts)
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to apply component config %+v to component %s", componentSettings, commonObjectMeta.Name)
// 		}
// 	}

// 	commonImageMeta := kclient.CommonImageMeta{
// 		Namespace: imageNS,
// 		Name:      imageName,
// 		Tag:       imageTag,
// 		Ports:     ports,
// 	}

// 	// Generate the new DeploymentConfig
// 	resourceLimits := kclient.FetchContainerResourceLimits(foundCurrentDCContainer)
// 	resLts, err := kclient.GetResourceRequirementsFromCmpSettings(componentSettings)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to update component")
// 	}
// 	if resLts != nil {
// 		resourceLimits = *resLts
// 	}

// 	// we choose the env variables in the config over the one present in the DC
// 	// so the local config is reflected on the cluster
// 	evl, err := kclient.GetInputEnvVarsFromStrings(envVarsList.ToStringSlice())
// 	if err != nil {
// 		return err
// 	}
// 	updateComponentParams := kclient.UpdateComponentParams{
// 		CommonObjectMeta:     commonObjectMeta,
// 		ImageMeta:            commonImageMeta,
// 		ResourceLimits:       resourceLimits,
// 		DcRollOutWaitCond:    kclient.IsDCRolledOut,
// 		ExistingDC:           currentDC,
// 		StorageToBeMounted:   storageToMount,
// 		StorageToBeUnMounted: storageToUnMount,
// 		EnvVars:              evl,
// 	}

// 	// STEP 2. Determine what the new source is going to be

// 	glog.V(4).Infof("Updating component %s, from %s to %s (%s).", componentName, oldSourceType, newSource, newSourceType)

// 	if (oldSourceType == "local" || oldSourceType == "binary") && newSourceType == "git" {
// 		// Steps to update component from local or binary to git
// 		// 1. Create a BuildConfig
// 		// 2. Update DeploymentConfig with the new image
// 		// 3. Clean up
// 		// 4. Build the application

// 		// CreateBuildConfig here!
// 		glog.V(4).Infof("Creating BuildConfig %s using imageName: %s for updating", namespacedOpenShiftObject, imageName)
// 		bc, err := client.CreateBuildConfig(commonObjectMeta, componentImageType, newSource, newSourceRef, evl)
// 		if err != nil {
// 			return errors.Wrapf(err, "unable to update BuildConfig  for %s component", componentName)
// 		}

// 		// we need to retrieve and build the git repository before deployment for the git components
// 		// so we build before updating the deployment
// 		err = Build(client, componentName, applicationName, true, stdout, false)
// 		if err != nil {
// 			return errors.Wrapf(err, "unable to build the component %s", componentName)
// 		}

// 		// Update / replace the current DeploymentConfig with a Git one (not SupervisorD!)
// 		glog.V(4).Infof("Updating the DeploymentConfig %s image to %s", namespacedOpenShiftObject, bc.Spec.Output.To.Name)

// 		// Update the image for git deployment to the BC built component image
// 		updateComponentParams.ImageMeta.Name = bc.Spec.Output.To.Name
// 		isDeleteSupervisordVolumes := (oldSourceType != string(config.GIT))

// 		err = client.UpdateDCToGit(
// 			updateComponentParams,
// 			isDeleteSupervisordVolumes,
// 		)
// 		if err != nil {
// 			return errors.Wrapf(err, "unable to update DeploymentConfig image for %s component", componentName)
// 		}

// 	} else if oldSourceType == "git" && (newSourceType == "binary" || newSourceType == "local") {

// 		// Steps to update component from git to local or binary

// 		// Update the sourceURL since it is not a local/binary file.
// 		sourceURL := util.GenFileURL(newSource)
// 		annotations[componentSourceURLAnnotation] = sourceURL
// 		updateComponentParams.CommonObjectMeta.Annotations = annotations

// 		// Need to delete the old BuildConfig
// 		err = client.DeleteBuildConfig(commonObjectMeta)

// 		if err != nil {
// 			return errors.Wrapf(err, "unable to delete BuildConfig for %s component", componentName)
// 		}

// 		// Update the DeploymentConfig
// 		err = client.UpdateDCToSupervisor(
// 			updateComponentParams,
// 			newSourceType == config.LOCAL,
// 			true,
// 		)
// 		if err != nil {
// 			return errors.Wrapf(err, "unable to update DeploymentConfig for %s component", componentName)
// 		}

// 	} else {
// 		// save source path as annotation
// 		// this part is for updates where the source does not change or change from local to binary and vice versa

// 		if newSourceType == "git" {

// 			// Update the BuildConfig
// 			err = client.UpdateBuildConfig(namespacedOpenShiftObject, newSource, annotations)
// 			if err != nil {
// 				return errors.Wrapf(err, "unable to update the build config %v", componentName)
// 			}

// 			bc, err := client.GetBuildConfigFromName(namespacedOpenShiftObject)
// 			if err != nil {
// 				return errors.Wrap(err, "unable to get the BuildConfig file")
// 			}

// 			// we need to retrieve and build the git repository before deployment for git components
// 			// so we build it before running the deployment
// 			err = Build(client, componentName, applicationName, true, stdout, false)
// 			if err != nil {
// 				return errors.Wrapf(err, "unable to build the component: %v", componentName)
// 			}

// 			// Update the current DeploymentConfig with all config applied
// 			glog.V(4).Infof("Updating the DeploymentConfig %s image to %s", namespacedOpenShiftObject, bc.Spec.Output.To.Name)

// 			// Update the image for git deployment to the BC built component image
// 			updateComponentParams.ImageMeta.Name = bc.Spec.Output.To.Name
// 			isDeleteSupervisordVolumes := (oldSourceType != string(config.GIT))

// 			err = client.UpdateDCToGit(
// 				updateComponentParams,
// 				isDeleteSupervisordVolumes,
// 			)
// 			if err != nil {
// 				return errors.Wrapf(err, "unable to update DeploymentConfig image for %s component", componentName)
// 			}

// 		} else if newSourceType == "local" || newSourceType == "binary" {

// 			// Update the sourceURL
// 			sourceURL := util.GenFileURL(newSource)
// 			annotations[componentSourceURLAnnotation] = sourceURL
// 			updateComponentParams.CommonObjectMeta.Annotations = annotations

// 			// Update the DeploymentConfig
// 			err = client.UpdateDCToSupervisor(
// 				updateComponentParams,
// 				newSourceType == config.LOCAL,
// 				false,
// 			)
// 			if err != nil {
// 				return errors.Wrapf(err, "unable to update DeploymentConfig for %s component", componentName)
// 			}

// 		}

// 		if err != nil {
// 			return errors.Wrap(err, "unable to update the component")
// 		}
// 	}
// 	return nil
// }

// Exists checks whether a component with the given name exists in the current application or not
// componentName is the component name to perform check for
// The first returned parameter is a bool indicating if a component with the given name already exists or not
// The second returned parameter is the error that might occurs while execution
func Exists(client *kclient.Client, componentName, applicationName string) (bool, error) {
	deploymentName, err := util.NamespaceKubernetesObject(componentName, applicationName)
	if err != nil {
		return false, errors.Wrapf(err, "unable to create namespaced name")
	}
	deployment, _ := client.GetDeploymentFromName(deploymentName)
	if deployment != nil {
		return true, nil
	}
	return false, nil
}

// GetComponent provides component definition
func GetComponent(client *kclient.Client, componentName string, applicationName string, namespace string) (component Component, err error) {
	// Component Type
	componentType, err := GetComponentType(client, componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to get source type")
	}
	// Source
	_, path, err := GetComponentSource(client, componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to get source path")
	}
	// URL
	urlList, err := urlpkg.List(client, componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to get url list")
	}
	var urls []string
	for _, url := range urlList.Items {
		urls = append(urls, url.Name)
	}

	// Storage
	appStore, err := storage.List(client, componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to get storage list")
	}
	var storage []string
	for _, store := range appStore.Items {
		storage = append(storage, store.Name)
	}
	// Environment Variables
	Deployment, err := util.NamespaceKubernetesObject(componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to get DC list")
	}
	envVars, err := client.GetEnvVarsFromDep(Deployment)
	if err != nil {
		return component, errors.Wrap(err, "unable to get envVars list")
	}
	var filteredEnv []corev1.EnvVar
	for _, env := range envVars {
		if !strings.Contains(env.Name, "ODO") {
			filteredEnv = append(filteredEnv, env)
		}
	}

	if err != nil {
		return component, errors.Wrap(err, "unable to get envVars list")
	}

	linkedServices := make([]string, 0, 5)
	linkedComponents := make(map[string][]string)
	linkedSecretNames, err := GetComponentLinkedSecretNames(client, componentName, applicationName)
	if err != nil {
		return component, errors.Wrap(err, "unable to list linked secrets")
	}
	for _, secretName := range linkedSecretNames {
		secret, err := client.GetSecret(secretName, namespace)
		if err != nil {
			return component, errors.Wrapf(err, "unable to get info about secret %s", secretName)
		}
		componentName, containsComponentLabel := secret.Labels[componentlabels.ComponentLabel]
		if containsComponentLabel {
			if port, ok := secret.Annotations[kclient.ComponentPortAnnotationName]; ok {
				linkedComponents[componentName] = append(linkedComponents[componentName], port)
			}
		} else {
			linkedServices = append(linkedServices, secretName)
		}
	}

	component = getMachineReadableFormat(componentName, componentType)
	component.Namespace = client.Namespace
	component.Spec.App = applicationName
	component.Spec.Source = path
	component.Spec.URL = urls
	component.Spec.Storage = storage
	component.Spec.Env = filteredEnv
	component.Status.LinkedComponents = linkedComponents
	component.Status.LinkedServices = linkedServices
	component.Status.State = "Pushed"

	return component, nil
}

// // GetLogs follow the DeploymentConfig logs if follow is set to true
// func GetLogs(client *kclient.Client, componentName string, applicationName string, follow bool, stdout io.Writer) error {

// 	// Namespace the component
// 	namespacedOpenShiftObject, err := util.NamespaceOpenShiftObject(componentName, applicationName)
// 	if err != nil {
// 		return errors.Wrapf(err, "unable to create namespaced name")
// 	}

// 	// Retrieve the logs
// 	err = client.DisplayDeploymentConfigLog(namespacedOpenShiftObject, follow, stdout)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func getMachineReadableFormat(componentName, componentType string) Component {
	return Component{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Component",
			APIVersion: "udo.udo.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: componentName,
		},
		Spec: ComponentSpec{
			Type: componentType,
		},
		Status: ComponentStatus{},
	}

}

// GetMachineReadableFormatForList returns list of components in machine readable format
func GetMachineReadableFormatForList(components []Component) ComponentList {
	if len(components) == 0 {
		components = []Component{}
	}
	return ComponentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "udo.udo.io/v1alpha1",
		},
		ListMeta: metav1.ListMeta{},
		Items:    components,
	}

}

// isEmpty checks to see if a directory is empty
// shamelessly taken from: https://stackoverflow.com/questions/30697324/how-to-check-if-directory-on-path-is-empty
// this helps detect any edge cases where an empty directory is copied over
func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

// getStorageFromConfig gets all the storage from the config
// returns a list of storage in storage struct format
func getStorageFromConfig(localConfig *config.LocalConfigInfo) storage.StorageList {
	storageList := storage.StorageList{}
	for _, storageVar := range localConfig.GetStorage() {
		storageList.Items = append(storageList.Items, storage.GetMachineReadableFormat(storageVar.Name, storageVar.Size, storageVar.Path))
	}
	return storageList
}

// checkIfURLChangesWillBeMade checks to see if there are going to be any changes
// to the URLs when deploying and returns a true / false
func checkIfURLChangesWillBeMade(client *kclient.Client, componentConfig config.LocalConfigInfo) (bool, error) {

	urlList, err := urlpkg.List(client, componentConfig.GetName(), componentConfig.GetApplication())
	if err != nil {
		return false, err
	}

	// If config has URL(s) (since we check) or if the cluster has URL's but
	// componentConfig does not (deleting)
	if len(componentConfig.GetUrl()) > 0 || len(componentConfig.GetUrl()) == 0 && (len(urlList.Items) > 0) {
		return true, nil
	}

	return false, nil
}
