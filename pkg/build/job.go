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
	mvnCommand := "echo listing /data/idp/src && ls -la /data/idp/src && echo copying /data/idp/src to /tmp/app && cp -rf /data/idp/src /tmp/app && echo chown, listing and running mvn in /tmp/app: && chown -fR 1001 /tmp/app && cd /tmp/app && ls -la && mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo copying target to output dir && rm -rf /data/idp/output && mkdir -p /data/idp/output && cp -rf /tmp/app/target /data/idp/output && chown -fR 1001 /data/idp/output && echo listing /data/idp/output after mvn and chown 1001 buildoutput && ls -la /data/idp/output/target && echo rm -rf /data/idp/buildartifacts and copying artifacts && rm -rf /data/idp/buildartifacts && mkdir -p /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/* /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/shared/resources/ /data/idp/buildartifacts/ && cp /data/idp/src/src/main/liberty/config/jvmbx.options /data/idp/buildartifacts/jvm.options && echo chown the buildartifacts dir && chown -fR 1001 /data/idp/buildartifacts"

	if taskName == "inc" {
		mvnCommand = "echo listing /data/idp/src && ls -la /data/idp/src && echo copying /data/idp/src to /tmp/app && cp -rf /data/idp/src /tmp/app && echo chown, listing and running mvn in /tmp/app: && chown -fR 1001 /tmp/app && cd /tmp/app && ls -la && mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo copying target to output dir && rm -rf /data/idp/output && mkdir -p /data/idp/output && cp -rf /tmp/app/target /data/idp/output && chown -fR 1001 /data/idp/output && echo listing /data/idp/output after mvn and chown 1001 buildoutput && ls -la /data/idp/output/target && echo copying artifacts && cp -rf /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/apps/* /data/idp/buildartifacts/apps/ && echo chown the buildartifacts apps dir && chown -fR 1001 /data/idp/buildartifacts/apps"
	}

	fmt.Printf("Mvn Command: %s\n", mvnCommand)
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
							Args:            []string{mvnCommand},
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
