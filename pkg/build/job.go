package build

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateBuildTaskKubeJob creates a Kubernetes Job
func CreateBuildTaskKubeJob(buildTaskJobName string, buildTaskType string, namespace string, idpClaimName string, projectSubPath string, projectName string) (*batchv1.Job, error) {
	fmt.Printf("Creating job %s\n", buildTaskJobName)
	// Create a Kube job to run mvn package for a Liberty project
	command := []string{"/bin/sh", "-c"}
	commandArgs := []string{string(FullBuildTask)}

	if buildTaskType == string(Incremental) {
		commandArgs = []string{string(IncrementalBuildTask)}
	}

	fmt.Printf("Command: %s %s\n", command, commandArgs)
	backoffLimit := int32(1)
	parallelism := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildTaskJobName,
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
							Command:         command,
							Args:            commandArgs,
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
