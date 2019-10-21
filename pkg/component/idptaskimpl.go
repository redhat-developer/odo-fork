package component

import (
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/idp"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/storage"
	"github.com/redhat-developer/odo-fork/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskExec is the Build Task or the Runtime Task execution implementation of the IDP
func TaskExec(Client *kclient.Client, componentConfig config.LocalConfigInfo, fullBuild bool, devPack *idp.IDP) error {
	// clientset := Client.KubeClient
	namespace := Client.Namespace
	cmpName := componentConfig.GetName()
	appName := componentConfig.GetApplication()
	// Namespace the component
	namespacedKubernetesObject, err := util.NamespaceKubernetesObject(cmpName, appName)

	glog.V(0).Infof("Namespace: %s\n", namespace)

	// Get the IDP Scenario
	var idpScenario idp.SpecScenario
	if fullBuild {
		idpScenario, err = devPack.GetScenario("full-build")
	} else {
		idpScenario, err = devPack.GetScenario("incremental-build")
	}
	if err != nil {
		glog.V(0).Infof("Error occured while getting the scenarios from the IDP")
		err = errors.New("Error occured while getting the scenarios from the IDP: " + err.Error())
		return err
	}

	// Get the IDP Tasks
	var idpTasks []idp.SpecTask
	idpTasks = devPack.GetTasks(idpScenario)

	// Get the Shared Volumes
	// This may need to be updated to handle mount and unmount of PVCs,
	// if user updates idp.yaml, check storage.go's Push() func for ref
	idpPVC := make(map[string]*corev1.PersistentVolumeClaim)
	sharedVolumes := devPack.GetSharedVolumes()

	for _, vol := range sharedVolumes {
		PVCs, err := Client.GetPVCsFromSelector("app.kubernetes.io/component-name=" + cmpName + ",app.kubernetes.io/storage-name=" + vol.Name)
		if err != nil {
			glog.V(0).Infof("Error occured while getting the PVC")
			err = errors.New("Unable to get the PVC: " + err.Error())
			return err
		}
		if len(PVCs) == 1 {
			existingPVC := &PVCs[0]
			idpPVC[vol.Name] = existingPVC
		}
		if len(PVCs) == 0 {
			createdPVC, err := storage.Create(Client, vol.Name, vol.Size, cmpName, appName)
			idpPVC[vol.Name] = createdPVC
			if err != nil {
				glog.V(0).Infof("Error creating the PVC: " + err.Error())
				err = errors.New("Error creating the PVC: " + err.Error())
				return err
			}
		}

		glog.V(0).Infof("Using PVC: %s\n", idpPVC[vol.Name].GetName())
	}

	serviceAccountName := "default"
	glog.V(0).Infof("Service Account: %s\n", serviceAccountName)

	// cwd is the project root dir, where udo command will run
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	glog.V(0).Infof("CWD: %s\n", cwd)

	timeout := int64(10)

	for _, task := range idpTasks {
		useRuntime := false
		if task.Type == idp.RuntimeTask {
			useRuntime = true
		}

		taskContainerInfo, err := devPack.GetTaskContainerInfo(task)
		if err != nil {
			glog.V(0).Infof("Error occured while getting the Task Container Info for task " + task.Name)
			err = errors.New("Error occured while getting the Task Container Info for task " + task.Name + ": " + err.Error())
			return err
		}

		containerImage := taskContainerInfo.Image
		var containerName, trimmedNamespacedKubernetesObject, srcDestination string
		var pvcClaimName, mountPath, subPath []string
		var cmpPVC []*corev1.PersistentVolumeClaim

		if len(namespacedKubernetesObject) > 40 {
			trimmedNamespacedKubernetesObject = namespacedKubernetesObject[:40]
		} else {
			trimmedNamespacedKubernetesObject = namespacedKubernetesObject
		}

		if task.Type == idp.RuntimeTask {
			containerName = trimmedNamespacedKubernetesObject + "-runtime"
		} else if task.Type == idp.SharedTask {
			containerName = trimmedNamespacedKubernetesObject + task.Container
			if len(containerName) > 63 {
				containerName = containerName[:63]
			}
		}
		for _, vm := range taskContainerInfo.VolumeMappings {
			cmpPVC = append(cmpPVC, idpPVC[vm.VolumeName])
			pvcClaimName = append(pvcClaimName, idpPVC[vm.VolumeName].Name)
			mountPath = append(mountPath, vm.ContainerPath)
			subPath = append(subPath, vm.SubPath)
		}

		if len(task.SourceMapping.DestPath) > 0 {
			srcDestination = task.SourceMapping.DestPath
		}

		BuildTaskInstance := BuildTask{
			UseRuntime:         useRuntime,
			Name:               containerName,
			Image:              containerImage,
			ContainerName:      containerName,
			Namespace:          namespace,
			PVCName:            pvcClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged:     true,
			MountPath:      mountPath,
			SubPath:        subPath,
			Command:        task.Command,
			SrcDestination: srcDestination,
		}
		BuildTaskInstance.Labels = map[string]string{
			"app": BuildTaskInstance.Name,
		}

		var watchOptions metav1.ListOptions
		if task.Type == idp.SharedTask {
			watchOptions = metav1.ListOptions{
				LabelSelector:  "app=" + BuildTaskInstance.Name,
				TimeoutSeconds: &timeout,
			}
		} else if task.Type == idp.RuntimeTask {
			watchOptions = metav1.ListOptions{
				LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
				TimeoutSeconds: &timeout,
			}
		}

		glog.V(0).Infof("Checking if " + task.Type + " Container has already been deployed...\n")

		foundTaskContainer := false
		po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a "+task.Type+" Container has already been deployed")
		if po != nil {
			glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
			BuildTaskInstance.PodName = po.Name
			foundTaskContainer = true
		}

		if !foundTaskContainer {
			glog.V(0).Info("===============================")
			glog.V(0).Info("Creating a " + task.Type + " Container")

			if task.Type == idp.SharedTask {
				s := log.Spinner("Creating pod")
				defer s.End(false)
				_, err := Client.CreatePod(BuildTaskInstance.Name, BuildTaskInstance.ContainerName, BuildTaskInstance.Image, BuildTaskInstance.ServiceAccountName, BuildTaskInstance.Labels, BuildTaskInstance.PVCName, BuildTaskInstance.MountPath, BuildTaskInstance.SubPath, BuildTaskInstance.Privileged)
				if err != nil {
					glog.V(0).Info("Failed to create a pod: " + err.Error())
					err = errors.New("Failed to create a pod " + BuildTaskInstance.Name)
					return err
				}
				s.End(true)
			} else if task.Type == idp.RuntimeTask {
				s := log.Spinner("Creating component")
				defer s.End(false)
				if err = BuildTaskInstance.CreateComponent(Client, componentConfig, devPack, cmpPVC); err != nil {
					err = errors.New("Unable to create component deployment: " + err.Error())
					return err
				}
				s.End(true)
			}

			glog.V(0).Info("Successfully created a " + task.Type + " Container")
			glog.V(0).Info("===============================")
		}

		// Only sync to the Runtime if a Source Mapping is provided
		if len(srcDestination) > 0 {
			err = syncProjectToRunningContainer(Client, watchOptions, cwd, BuildTaskInstance.SrcDestination)
			if err != nil {
				glog.V(0).Infof("Error occured while syncing to the %s Container: %s\n", task.Type, err)
				err = errors.New("Unable to sync to the pod: " + err.Error())
				return err
			}
		}

		// Only execute tasks commands in Runtime if commands are provided
		if len(BuildTaskInstance.Command) > 0 {
			err = executetask(Client, strings.Join(BuildTaskInstance.Command, " "), watchOptions)
			if err != nil {
				glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(BuildTaskInstance.Command, " "), BuildTaskInstance.PodName, err)
				err = errors.New("Unable to exec command " + strings.Join(BuildTaskInstance.Command, " ") + " in the runtime container: " + err.Error())
				return err
			}
		}
	}

	// Finally, check if the Component has been deployed and start one
	// if absent. Because, there can be a IDP task without a Runtime type
	glog.V(0).Infof("Checking if the Component has already been deployed...\n")

	var taskContainerInfo idp.TaskContainerInfo
	var containerName, containerImage, trimmedNamespacedKubernetesObject, srcDestination string
	var pvcClaimName, mountPath, subPath []string
	var cmpPVC []*corev1.PersistentVolumeClaim
	var BuildTaskInstance BuildTask

	foundComponent := false
	watchOptions := metav1.ListOptions{
		LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
		TimeoutSeconds: &timeout,
	}
	po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Component has already been deployed")
	if po != nil {
		glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
		BuildTaskInstance.PodName = po.Name
		foundComponent = true
	}

	if !foundComponent {
		taskContainerInfo = devPack.GetRuntimeInfo()

		if len(namespacedKubernetesObject) > 40 {
			trimmedNamespacedKubernetesObject = namespacedKubernetesObject[:40]
		} else {
			trimmedNamespacedKubernetesObject = namespacedKubernetesObject
		}
		containerImage = taskContainerInfo.Image
		containerName = trimmedNamespacedKubernetesObject + "-runtime"

		for _, vm := range taskContainerInfo.VolumeMappings {
			cmpPVC = append(cmpPVC, idpPVC[vm.VolumeName])
			pvcClaimName = append(pvcClaimName, idpPVC[vm.VolumeName].Name)
			mountPath = append(mountPath, vm.ContainerPath)
			subPath = append(subPath, vm.SubPath)
		}

		BuildTaskInstance = BuildTask{
			UseRuntime:         true,
			Name:               containerName,
			Image:              containerImage,
			ContainerName:      containerName,
			Namespace:          namespace,
			PVCName:            pvcClaimName,
			ServiceAccountName: serviceAccountName,
			// OwnerReferenceName: ownerReferenceName,
			// OwnerReferenceUID:  ownerReferenceUID,
			Privileged:     true,
			MountPath:      mountPath,
			SubPath:        subPath,
			SrcDestination: srcDestination,
		}
		BuildTaskInstance.Labels = map[string]string{
			"app": BuildTaskInstance.Name,
		}

		glog.V(0).Info("===============================")
		glog.V(0).Info("Creating the Component")

		s := log.Spinner("Creating component")
		defer s.End(false)
		if err = BuildTaskInstance.CreateComponent(Client, componentConfig, devPack, cmpPVC); err != nil {
			err = errors.New("Unable to create component deployment: " + err.Error())
			return err
		}
		s.End(true)

		glog.V(0).Info("Successfully created the Container")
		glog.V(0).Info("===============================")
	}

	return nil
}
