package component

import (
	corev1 "k8s.io/api/core/v1"
)

const (

	// ReusableBuildContainerType is a BuildTaskKind where udo will reuse the build container to build projects
	ReusableBuildContainerType string = "ReusableBuildContainer"
	// ComponentType is a BuildTaskKind where udo will deploy a component
	ComponentType string = "Component"

	// IDPVolumeName holds the IDP volume name
	IDPVolumeName string = "idp-volume"
)

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
