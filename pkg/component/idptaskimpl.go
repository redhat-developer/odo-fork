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

	// idpClaimName := ""
	// var cmpPVC *corev1.PersistentVolumeClaim

	// Get the Shared Volumes
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

	// PVCs, err := Client.GetPVCsFromSelector("app.kubernetes.io/component-name=" + cmpName + ",app.kubernetes.io/storage-name=" + cmpName)
	// if err != nil {
	// 	glog.V(0).Infof("Error occured while getting the PVC")
	// 	err = errors.New("Unable to get the PVC: " + err.Error())
	// 	return err
	// }
	// if len(PVCs) == 1 {
	// 	cmpPVC = &PVCs[0]
	// }

	// if len(PVCs) == 0 {
	// 	// sharedVolumes := devPack.GetSharedVolumes()
	// 	for _, v := range sharedVolumes {
	// 		cmpPVC, err = storage.Create(Client, v.Name, v.Size, cmpName, appName)
	// 		if err != nil {
	// 			glog.V(0).Infof("Error creating the PVC")
	// 			err = errors.New("Error creating the PVC: " + err.Error())
	// 			return err
	// 		}
	// 	}
	// }

	// idpClaimName = cmpPVC.GetName()

	// glog.V(0).Infof("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	glog.V(0).Infof("Service Account: %s\n", serviceAccountName)

	// cwd is the project root dir, where udo command will run
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.New("Unable to get the cwd" + err.Error())
		return err
	}
	glog.V(0).Infof("CWD: %s\n", cwd)

	// var TaskInstance map[string]BuildTask

	for _, task := range idpTasks {
		useRuntime := false
		if task.Type == idp.RuntimeTask {
			useRuntime = true
		}

		taskContainerInfo, err := devPack.GetContainer(task)
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

		if task.Type == idp.SharedTask {

			// Execute the Shared Tasks in Reusable Containers
			glog.V(0).Infof("Checking if Reusable Build Container has already been deployed...\n")
			foundReusableBuildContainer := false
			timeout := int64(10)
			watchOptions := metav1.ListOptions{
				LabelSelector:  "app=" + BuildTaskInstance.Name,
				TimeoutSeconds: &timeout,
			}
			po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Reusable Container is up")
			if po != nil {
				glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
				BuildTaskInstance.PodName = po.Name
				foundReusableBuildContainer = true
			}

			if !foundReusableBuildContainer {
				glog.V(0).Info("===============================")
				glog.V(0).Info("Creating a pod...")

				pod, err := Client.CreatePod(BuildTaskInstance.Name, BuildTaskInstance.ContainerName, BuildTaskInstance.Image, BuildTaskInstance.ServiceAccountName, BuildTaskInstance.Labels, BuildTaskInstance.PVCName, BuildTaskInstance.MountPath, BuildTaskInstance.SubPath, BuildTaskInstance.Privileged)
				if err != nil {
					glog.V(0).Info("Failed to create a pod: " + err.Error())
					err = errors.New("Failed to create a pod " + BuildTaskInstance.Name)
					return err
				}
				glog.V(0).Info("Created pod: " + pod.GetName())
				glog.V(0).Info("===============================")
				// Wait for pods to start and grab the pod name
				glog.V(0).Infof("Waiting for pod to run\n")
				watchOptions := metav1.ListOptions{
					LabelSelector: "app=" + BuildTaskInstance.Name,
				}
				po, err := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for the Reusable Build Container to run")
				if err != nil {
					err = errors.New("The Reusable Build Container failed to run")
					return err
				}

				BuildTaskInstance.PodName = po.Name
			}

			glog.V(0).Infof("The Reusable Build Container Pod Name: %s\n", BuildTaskInstance.PodName)

			watchOptions = metav1.ListOptions{
				LabelSelector: "app=" + BuildTaskInstance.Name,
			}
			err = syncProjectToRunningContainer(Client, watchOptions, cwd, BuildTaskInstance.SrcDestination)
			if err != nil {
				glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", BuildTaskInstance.PodName, err)
				err = errors.New("Unable to sync to the pod: " + err.Error())
				return err
			}

			err = executetask(Client, strings.Join(BuildTaskInstance.Command, " "), BuildTaskInstance.PodName)
			if err != nil {
				glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(task.Command, " "), BuildTaskInstance.PodName, err)
				err = errors.New("Unable to exec command " + strings.Join(task.Command, " ") + " in the runtime container: " + err.Error())
				return err
			}
		} else if task.Type == idp.RuntimeTask {

			// Execute the Runtime Tasks in the Component Pod
			foundRuntimeContainer := false
			timeout := int64(10)
			watchOptions := metav1.ListOptions{
				LabelSelector:  "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
				TimeoutSeconds: &timeout,
			}
			po, _ := Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if a Runtime Container has already been deployed")
			if po != nil {
				glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
				BuildTaskInstance.PodName = po.Name
				foundRuntimeContainer = true
			}

			if !foundRuntimeContainer {
				// Deploy the application if it is a full build type and a running pod is not found
				glog.V(0).Info("Deploying the application")

				BuildTaskInstance.Labels = map[string]string{
					"app":     BuildTaskInstance.Name + "-selector",
					"chart":   BuildTaskInstance.Name + "-1.0.0",
					"release": BuildTaskInstance.Name,
				}

				s := log.Spinner("Creating component")
				defer s.End(false)
				if err = BuildTaskInstance.CreateComponent(Client, componentConfig, devPack, cmpPVC); err != nil {
					err = errors.New("Unable to create component deployment: " + err.Error())
					return err
				}
				s.End(true)
			}

			// Only sync to the Runtime if a Source Mapping is provided
			if len(srcDestination) > 0 {
				watchOptions = metav1.ListOptions{
					LabelSelector: "app=" + namespacedKubernetesObject + ",deployment=" + namespacedKubernetesObject,
				}
				err = syncProjectToRunningContainer(Client, watchOptions, cwd, BuildTaskInstance.SrcDestination)
				if err != nil {
					glog.V(0).Infof("Error occured while syncing to the pod %s: %s\n", BuildTaskInstance.PodName, err)
					err = errors.New("Unable to sync to the pod: " + err.Error())
					return err
				}
			}

			// Only execute tasks commands in Runtime if commands are provided
			if len(BuildTaskInstance.Command) > 0 {
				po, _ = Client.WaitAndGetPod(watchOptions, corev1.PodRunning, "Checking to see if the Runtime Container is up before executing the Runtime Tasks")
				if po != nil {
					glog.V(0).Infof("Running pod found: %s...\n\n", po.Name)
					BuildTaskInstance.PodName = po.Name

					err = executetask(Client, strings.Join(task.Command, " "), BuildTaskInstance.PodName)
					if err != nil {
						glog.V(0).Infof("Error occured while executing command %s in the pod %s: %s\n", strings.Join(BuildTaskInstance.Command, " "), BuildTaskInstance.PodName, err)
						err = errors.New("Unable to exec command " + strings.Join(task.Command, " ") + " in the runtime container: " + err.Error())
						return err
					}
				}
			}
		}
	}

	return nil
}
