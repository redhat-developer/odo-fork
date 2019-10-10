package kclient

import (
	"github.com/pkg/errors"

	"github.com/redhat-developer/odo-fork/pkg/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CommonImageMeta has all the most common image data that is passed around within Odo
type CommonImageMeta struct {
	Name      string
	Tag       string
	Namespace string
	Ports     []corev1.ContainerPort
}

// GetResourceRequirementsFromCmpSettings converts the cpu and memory request info from component configuration into format usable in deployment
// Parameters:
//	cfg: Compoennt configuration/settings
// Returns:
//	*corev1.ResourceRequirements: component configuration converted into format usable in deployment
func GetResourceRequirementsFromCmpSettings(cfg config.LocalConfigInfo) (*corev1.ResourceRequirements, error) {
	var resourceRequirements corev1.ResourceRequirements
	requests := make(corev1.ResourceList)
	limits := make(corev1.ResourceList)

	cfgMinCPU := cfg.GetMinCPU()
	cfgMaxCPU := cfg.GetMaxCPU()
	cfgMinMemory := cfg.GetMinMemory()
	cfgMaxMemory := cfg.GetMaxMemory()

	if cfgMinCPU != "" {
		minCPU, err := parseResourceQuantity(cfgMinCPU)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse the min cpu")
		}
		requests[corev1.ResourceCPU] = minCPU
	}

	if cfgMaxCPU != "" {
		maxCPU, err := parseResourceQuantity(cfgMaxCPU)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse max cpu")
		}
		limits[corev1.ResourceCPU] = maxCPU
	}

	if cfgMinMemory != "" {
		minMemory, err := parseResourceQuantity(cfgMinMemory)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse min memory")
		}
		requests[corev1.ResourceMemory] = minMemory
	}

	if cfgMaxMemory != "" {
		maxMemory, err := parseResourceQuantity(cfgMaxMemory)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse max memory")
		}
		limits[corev1.ResourceMemory] = maxMemory
	}

	if len(limits) > 0 {
		resourceRequirements.Limits = limits
	}

	if len(requests) > 0 {
		resourceRequirements.Requests = requests
	}

	return &resourceRequirements, nil
}

// parseResourceQuantity takes a string representation of quantity/amount of a resource and returns kubernetes representation of it and errors if any
// This is a wrapper around the kube client provided ParseQuantity added to in future support more units and make it more readable
func parseResourceQuantity(resQuantity string) (resource.Quantity, error) {
	return resource.ParseQuantity(resQuantity)
}

// generateDeployment generates a deployment for local and binary components
// Parameters:
//	commonObjectMeta: Contains annotations and labels for deployment
//	commonImageMeta: Contains details like image NS, name, tag and ports to be exposed
//	envVar: env vars to be exposed
//	resourceRequirements: Container cpu and memory resource requirements
// Returns:
//	deployment generated using above parameters
func generateDeployment(commonObjectMeta metav1.ObjectMeta, commonImageMeta CommonImageMeta,
	envVar []corev1.EnvVar, envFrom []corev1.EnvFromSource, resourceRequirements *corev1.ResourceRequirements,
	useRuntime bool) appsv1.Deployment {

	labels := map[string]string{
		"app":        commonObjectMeta.Name,
		"deployment": commonObjectMeta.Name,
	}

	imageRef := commonImageMeta.Name + ":" + commonImageMeta.Tag
	if len(commonImageMeta.Namespace) > 0 {
		imageRef = commonImageMeta.Namespace + "/" + imageRef
	}

	containers := []corev1.Container{
		{
			Name:  commonObjectMeta.Name,
			Image: imageRef,
			Env:   envVar,
			Ports: commonImageMeta.Ports,
		},
	}

	if useRuntime {
		containers[0].Command = []string{"/bin/sh", "-c", "--"}
		containers[0].Args = []string{"tail -f /dev/null"}
	}

	replicas := int32(1)
	// Generates and deploys a Deployment with an InitContainer to copy over the SupervisorD binary.
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: commonObjectMeta,
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
					Containers: containers,
				},
			},
		},
	}

	containerIndex := -1
	if resourceRequirements != nil {
		for index, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == commonObjectMeta.Name {
				containerIndex = index
				break
			}
		}
		if containerIndex != -1 {
			deployment.Spec.Template.Spec.Containers[containerIndex].Resources = *resourceRequirements
		}
	}
	return deployment
}
