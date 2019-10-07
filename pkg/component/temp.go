package component

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// FullBuildTask is the relative path of the IDP full build task in the Persistent Volume's project directory
	FullBuildTask string = "/.udo/build-container-full.sh"
	// IncrementalBuildTask is the relative path of the IDP incremental build task in the Persistent Volume's project directory
	IncrementalBuildTask string = "/.udo/build-container-update.sh"

	// FullRunTask is the relative path of the IDP full run task in the Runtime Container Empty Dir Volume's project directory
	FullRunTask string = "/.udo/runtime-container-full.sh"
	// IncrementalRunTask is the relative path of the IDP incremental run task in the Runtime Container Empty Dir Volume's project directory
	IncrementalRunTask string = "/.udo/runtime-container-update.sh"

	// ReusableBuildContainerType is a BuildTaskKind where udo will reuse the build container to build projects
	ReusableBuildContainerType string = "ReusableBuildContainer"
	// ComponentType is a BuildTaskKind where udo will deploy a component
	ComponentType string = "Component"

	// BuildContainerImage holds the image name of the build task container
	BuildContainerImage string = "docker.io/maven:3.6"
	// BuildContainerName holds the container name of the build task container
	BuildContainerName string = "maven-build"
	// BuildContainerMountPath  holds the mount path of the build task container
	BuildContainerMountPath string = "/data/idp/"

	// RuntimeContainerImage is the default WebSphere Liberty Runtime image
	RuntimeContainerImage string = "websphere-liberty:19.0.0.3-webProfile7"
	// RuntimeContainerImageWithBuildTools is the default WebSphere Liberty Runtime image with Maven and Java installed
	RuntimeContainerImageWithBuildTools string = "maysunfaisal/libertymvnjava"
	// RuntimeContainerName is the runtime container name
	RuntimeContainerName string = "libertyproject"
	// RuntimeContainerMountPathDefault  holds the default mount path of the runtime task container
	RuntimeContainerMountPathDefault string = "/config"
	// RuntimeContainerMountPathEmptyDir  holds the empty dir mount path of the runtime task container
	RuntimeContainerMountPathEmptyDir string = "/home/default/idp"

	// IDPVolumeName holds the IDP volume name
	IDPVolumeName string = "idp-volume"
)

// CreateDeploy creates a Kubernetes deployment
func CreateDeploy(b BuildTask) appsv1.Deployment {

	volumes, volumeMounts := SetVolumes(b)
	envVars := SetEnvVars(b)

	return generateDeployment(b, volumes, volumeMounts, envVars)
}

// CreateService creates a Kubernetes service for Codewind, exposing port 9191
func CreateService(b BuildTask) corev1.Service {

	return generateService(b)
}

// SetVolumes sets the IDP task volumes with either the PVC or the Empty Dir depending on the task
func SetVolumes(b BuildTask) ([]corev1.Volume, []corev1.VolumeMount) {

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
func SetEnvVars(b BuildTask) []corev1.EnvVar {

	envVars := []corev1.EnvVar{}

	return envVars
}

// generateDeployment returns a Kubernetes deployment object with the given name for the given image.
// Additionally, volume/volumemounts and env vars can be specified.
func generateDeployment(b BuildTask, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar) appsv1.Deployment {
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
	if b.Kind == ReusableBuildContainerType || b.UseRuntime {
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
func generateService(b BuildTask) corev1.Service {
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

	if b.Kind == ReusableBuildContainerType {
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
