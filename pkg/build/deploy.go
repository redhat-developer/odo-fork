package build

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//BuildTask is a struct of essential data
type BuildTask struct {
	Type               string
	Name               string
	Image              string
	ContainerName      string
	PodName            string
	Namespace          string
	WorkspaceID        string
	PVCName            string
	ServiceAccountName string
	PullSecret         string
	OwnerReferenceName string
	OwnerReferenceUID  types.UID
	Privileged         bool
	Ingress            string
	MountPath          string
	SubPath            string
}

// CreateComponentDeploy creates a Kubernetes deployment
func CreateComponentDeploy(buildtask BuildTask, projectName string, labels map[string]string) appsv1.Deployment {

	volumes, volumeMounts := setPFEVolumes(buildtask, projectName)
	envVars := setPFEEnvVars(buildtask)

	return generateDeployment(buildtask, volumes, volumeMounts, envVars, labels)
}

// CreateComponentService creates a Kubernetes service for Codewind, exposing port 9191
func CreateComponentService(buildtask BuildTask, labels map[string]string) corev1.Service {

	return generateService(buildtask, labels)
}

// setPFEVolumes returns the 3 volumes & corresponding volume mounts required by the PFE container:
// project workspace, buildah volume, and the docker registry secret (the latter of which is optional)
func setPFEVolumes(buildtask BuildTask, projectName string) ([]corev1.Volume, []corev1.VolumeMount) {

	volumes := []corev1.Volume{
		{
			Name: "idp-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: buildtask.PVCName,
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "idp-volume",
			MountPath: buildtask.MountPath,
			SubPath:   buildtask.SubPath,
		},
	}

	return volumes, volumeMounts
}

// setPFEEnvVars sets the env var for the component pod
func setPFEEnvVars(buildtask BuildTask) []corev1.EnvVar {
	booleanTrue := bool(true)

	envVars := []corev1.EnvVar{
		{
			Name:  "PORT",
			Value: "9080",
		},
		{
			Name:  "APPLICATION_NAME",
			Value: "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		},
		{
			Name:  "PROJECT_NAME",
			Value: "maysunliberty2",
		},
		{
			Name:  "LOG_FOLDER",
			Value: "maysunliberty2-6c1b1ce0-cb4c-11e9-be96-bfc50f05726d",
		},
		{
			Name:  "IN_K8",
			Value: "true",
		},
		{
			Name: "IBM_APM_SERVER_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_server_url",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_KEYFILE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_keyfile_password",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_INGRESS_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_ingress_url",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_KEYFILE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_keyfile_password",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_ACCESS_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_access_token",
					Optional: &booleanTrue,
				},
			},
		},
	}

	if buildtask.Type == "build" {
		envVars = []corev1.EnvVar{}
	}

	return envVars
}

// generateDeployment returns a Kubernetes deployment object with the given name for the given image.
// Additionally, volume/volumemounts and env vars can be specified.
func generateDeployment(buildtask BuildTask, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar, labels map[string]string) appsv1.Deployment {
	// blockOwnerDeletion := true
	// controller := true
	containerName := buildtask.ContainerName
	image := buildtask.Image
	replicas := int32(1)
	container := []corev1.Container{
		{
			Name:            containerName,
			Image:           image,
			ImagePullPolicy: corev1.PullAlways,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &buildtask.Privileged,
			},
			VolumeMounts: volumeMounts,
			Env:          envVars,
		},
	}
	if buildtask.Type == "build" {
		container = []corev1.Container{
			{
				Name:            containerName,
				Image:           image,
				ImagePullPolicy: corev1.PullAlways,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &buildtask.Privileged,
				},
				VolumeMounts: volumeMounts,
				Command:      []string{"tail"},
				Args:         []string{"-f", "/dev/null"},
				Env:          envVars,
			},
		}
	}

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildtask.Name,
			Namespace: buildtask.Namespace,
			Labels:    labels,
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion:         "apps/v1",
			// 		BlockOwnerDeletion: &blockOwnerDeletion,
			// 		Controller:         &controller,
			// 		Kind:               "ReplicaSet",
			// 		Name:               codewind.OwnerReferenceName,
			// 		UID:                codewind.OwnerReferenceUID,
			// 	},
			// },
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: buildtask.ServiceAccountName,
					Volumes:            volumes,
					Containers:         container,
				},
			},
		},
	}
	return deployment
}

// generateService returns a Kubernetes service object with the given name, exposed over the specified port
// for the container with the given labels.
func generateService(buildtask BuildTask, labels map[string]string) corev1.Service {
	// blockOwnerDeletion := true
	// controller := true

	port1 := 9080
	port2 := 9443

	ports := []corev1.ServicePort{
		{
			Port: int32(port1),
			Name: "http",
		},
		{
			Port: int32(port2),
			Name: "https",
		},
	}

	if buildtask.Type == "build" {
		ports = []corev1.ServicePort{
			{
				Port: int32(port1),
				Name: "http",
			},
		}
	}

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildtask.Name,
			Namespace: buildtask.Namespace,
			Labels:    labels,
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion:         "apps/v1",
			// 		BlockOwnerDeletion: &blockOwnerDeletion,
			// 		Controller:         &controller,
			// 		Kind:               "ReplicaSet",
			// 		Name:               codewind.OwnerReferenceName,
			// 		UID:                codewind.OwnerReferenceUID,
			// 	},
			// },
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Ports:    ports,
			Selector: labels,
		},
	}
	return service
}
