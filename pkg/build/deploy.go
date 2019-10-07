package build

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
	PVCName            string
	ServiceAccountName string
	PullSecret         string
	OwnerReferenceName string
	OwnerReferenceUID  types.UID
	Privileged         bool
	Ingress            string
	MountPath          string
	SubPath            string
	Labels             map[string]string
}

// CreateDeploy creates a Kubernetes deployment
func (b *BuildTask) CreateDeploy() appsv1.Deployment {

	volumes, volumeMounts := b.SetVolumes()
	envVars := b.SetEnvVars()

	return b.generateDeployment(volumes, volumeMounts, envVars)
}

// CreateService creates a Kubernetes service for Codewind, exposing port 9191
func (b *BuildTask) CreateService() corev1.Service {

	return b.generateService()
}

// SetVolumes sets the IDP task volumes with either the PVC or the Empty Dir depending on the task
func (b *BuildTask) SetVolumes() ([]corev1.Volume, []corev1.VolumeMount) {

	volumes := []corev1.Volume{
		{
			Name: IDPVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: b.PVCName,
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      IDPVolumeName,
			MountPath: b.MountPath,
			SubPath:   b.SubPath,
		},
	}

	if b.UseRuntime {
		volumes = []corev1.Volume{
			{
				Name: IDPVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						Medium: corev1.StorageMediumMemory,
					},
				},
			},
		}
	}

	return volumes, volumeMounts
}

// SetEnvVars sets the env var for the component pod
func (b *BuildTask) SetEnvVars() []corev1.EnvVar {

	envVars := []corev1.EnvVar{}

	return envVars
}

// generateDeployment returns a Kubernetes deployment object with the given name for the given image.
// Additionally, volume/volumemounts and env vars can be specified.
func (b *BuildTask) generateDeployment(volumes []corev1.Volume, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar) appsv1.Deployment {
	// blockOwnerDeletion := true
	// controller := true
	containerName := b.ContainerName
	image := b.Image
	labels := b.Labels
	replicas := int32(1)
	container := []corev1.Container{
		{
			Name:            containerName,
			Image:           image,
			ImagePullPolicy: corev1.PullAlways,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &b.Privileged,
			},
			VolumeMounts: volumeMounts,
			Env:          envVars,
		},
	}
	if b.Kind == ReusableBuildContainer || b.UseRuntime {
		container = []corev1.Container{
			{
				Name:            containerName,
				Image:           image,
				ImagePullPolicy: corev1.PullAlways,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &b.Privileged,
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
			Name:      b.Name,
			Namespace: b.Namespace,
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
					ServiceAccountName: b.ServiceAccountName,
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
func (b *BuildTask) generateService() corev1.Service {
	// blockOwnerDeletion := true
	// controller := true

	port1 := 9080
	port2 := 9443

	labels := b.Labels

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

	if b.Kind == ReusableBuildContainer {
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
			Name:      b.Name,
			Namespace: b.Namespace,
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
