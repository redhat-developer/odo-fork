package build

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateBuildTaskKubeJob creates a Kubernetes Job
func CreateBuildTaskKubeJob(buildTaskJob string, taskName string, namespace string, idpClaimName string, projectSubPath string, projectName string) (*batchv1.Job, error) {
	fmt.Printf("Creating job %s\n", buildTaskJob)
	// Create a Kube job to run mvn package for a Liberty project
	command := "/data/idp/bin/build-container-full.sh"

	if taskName == "inc" {
		command = "/data/idp/bin/build-container-update.sh"
	}

	fmt.Printf("Command: %s\n", command)
	backoffLimit := int32(1)
	parallelism := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildTaskJob,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "idp-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: idpClaimName,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "maven-build",
							Image:           "docker.io/maven:3.6",
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{command},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "idp-volume",
									MountPath: "/data/idp/",
									SubPath:   projectSubPath,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backoffLimit,
			Parallelism:  &parallelism,
		},
	}

	return job, nil
}
