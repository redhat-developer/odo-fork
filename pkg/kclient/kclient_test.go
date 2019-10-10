package kclient

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/redhat-developer/odo-fork/pkg/util"

	// api resources

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	ktesting "k8s.io/client-go/testing"
)

func fakeResourceConsumption() []util.ResourceRequirementInfo {
	return []util.ResourceRequirementInfo{
		*util.FetchResourceQuantity(corev1.ResourceMemory, "100Mi", "350Mi", ""),
		*util.FetchResourceQuantity(corev1.ResourceCPU, "100m", "350m", ""),
	}
}

func fakePodStatus(status corev1.PodPhase, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Status: corev1.PodStatus{
			Phase: status,
		},
	}
}

func fakePlanExternalMetaDataRaw() ([][]byte, error) {
	planExternalMetaData1 := make(map[string]string)
	planExternalMetaData1["displayName"] = "plan-name-1"

	planExternalMetaData2 := make(map[string]string)
	planExternalMetaData2["displayName"] = "plan-name-2"

	planExternalMetaDataRaw1, err := json.Marshal(planExternalMetaData1)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	planExternalMetaDataRaw2, err := json.Marshal(planExternalMetaData2)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var data [][]byte
	data = append(data, planExternalMetaDataRaw1)
	data = append(data, planExternalMetaDataRaw2)

	return data, nil
}

func fakePlanServiceInstanceCreateParameterSchemasRaw() ([][]byte, error) {
	planServiceInstanceCreateParameterSchema1 := make(map[string][]string)
	planServiceInstanceCreateParameterSchema1["required"] = []string{"PLAN_DATABASE_URI", "PLAN_DATABASE_USERNAME", "PLAN_DATABASE_PASSWORD"}

	planServiceInstanceCreateParameterSchema2 := make(map[string][]string)
	planServiceInstanceCreateParameterSchema2["required"] = []string{"PLAN_DATABASE_USERNAME_2", "PLAN_DATABASE_PASSWORD"}

	planServiceInstanceCreateParameterSchemaRaw1, err := json.Marshal(planServiceInstanceCreateParameterSchema1)
	if err != nil {
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	planServiceInstanceCreateParameterSchemaRaw2, err := json.Marshal(planServiceInstanceCreateParameterSchema2)
	if err != nil {
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	var data [][]byte
	data = append(data, planServiceInstanceCreateParameterSchemaRaw1)
	data = append(data, planServiceInstanceCreateParameterSchemaRaw2)

	return data, nil
}

func TestAddLabelsToArgs(t *testing.T) {
	tests := []struct {
		name     string
		argsIn   []string
		labels   map[string]string
		argsOut1 []string
		argsOut2 []string
	}{
		{
			name:   "one label in empty args",
			argsIn: []string{},
			labels: map[string]string{
				"label1": "value1",
			},
			argsOut1: []string{
				"--labels", "label1=value1",
			},
		},
		{
			name: "one label with existing args",
			argsIn: []string{
				"--foo", "bar",
			},
			labels: map[string]string{
				"label1": "value1",
			},
			argsOut1: []string{
				"--foo", "bar",
				"--labels", "label1=value1",
			},
		},
		{
			name: "multiple label with existing args",
			argsIn: []string{
				"--foo", "bar",
			},
			labels: map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
			argsOut1: []string{
				"--foo", "bar",
				"--labels", "label1=value1,label2=value2",
			},
			argsOut2: []string{
				"--foo", "bar",
				"--labels", "label2=value2,label1=value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsGot := addLabelsToArgs(tt.labels, tt.argsIn)

			if !reflect.DeepEqual(argsGot, tt.argsOut1) && !reflect.DeepEqual(argsGot, tt.argsOut2) {
				t.Errorf("addLabelsToArgs() \ngot:  %#v \nwant: %#v or %#v", argsGot, tt.argsOut1, tt.argsOut2)
			}
		})
	}
}

func TestParseImageName(t *testing.T) {

	tests := []struct {
		arg     string
		want1   string
		want2   string
		want3   string
		want4   string
		wantErr bool
	}{
		{
			arg:     "nodejs:8",
			want1:   "",
			want2:   "nodejs",
			want3:   "8",
			want4:   "",
			wantErr: false,
		},
		{
			arg:     "nodejs@sha256:7e56ca37d1db225ebff79dd6d9fd2a9b8f646007c2afc26c67962b85dd591eb2",
			want2:   "nodejs",
			want1:   "",
			want3:   "",
			want4:   "sha256:7e56ca37d1db225ebff79dd6d9fd2a9b8f646007c2afc26c67962b85dd591eb2",
			wantErr: false,
		},
		{
			arg:     "nodejs@sha256:asdf@",
			wantErr: true,
		},
		{
			arg:     "nodejs@@",
			wantErr: true,
		},
		{
			arg:     "nodejs::",
			wantErr: true,
		},
		{
			arg:     "nodejs",
			want1:   "",
			want2:   "nodejs",
			want3:   "latest",
			want4:   "",
			wantErr: false,
		},
		{
			arg:     "",
			wantErr: true,
		},
		{
			arg:     ":",
			wantErr: true,
		},
		{
			arg:     "myproject/nodejs:8",
			want1:   "myproject",
			want2:   "nodejs",
			want3:   "8",
			want4:   "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("image name: '%s'", tt.arg)
		t.Run(name, func(t *testing.T) {
			got1, got2, got3, got4, err := ParseImageName(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got1 != tt.want1 {
				t.Errorf("ParseImageName() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("ParseImageName() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("ParseImageName() got3 = %v, want %v", got3, tt.want3)
			}
			if got4 != tt.want4 {
				t.Errorf("ParseImageName() got4 = %v, want %v", got4, tt.want4)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name       string
		secretNS   string
		secretName string
		wantErr    bool
		want       *corev1.Secret
	}{
		{
			name:       "Case: Valid request for retrieving a secret",
			secretNS:   "",
			secretName: "foo",
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			wantErr: false,
		},
		{
			name:       "Case: Invalid request for retrieving a secret",
			secretNS:   "",
			secretName: "foo2",
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			// Fake getting Secret
			fakeClientSet.Kubernetes.PrependReactor("get", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
				if tt.want.Name != tt.secretName {
					return true, nil, fmt.Errorf("'get' called with a different secret name")
				}
				return true, tt.want, nil
			})

			returnValue, err := fakeClient.GetSecret(tt.secretName, tt.secretNS)

			// Check for validating return value
			if err == nil && returnValue != tt.want {
				t.Errorf("error in return value got: %v, expected %v", returnValue, tt.want)
			}

			if !tt.wantErr == (err != nil) {
				t.Errorf("\nclient.GetSecret(secretNS, secretName) unexpected error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListSecrets(t *testing.T) {

	tests := []struct {
		name       string
		secretList corev1.SecretList
		output     []corev1.Secret
		wantErr    bool
	}{
		{
			name: "Case 1: Ensure secrets are properly listed",
			secretList: corev1.SecretList{
				Items: []corev1.Secret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "secret1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "secret2",
						},
					},
				},
			},
			output: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "secret1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "secret2",
					},
				},
			},

			wantErr: false,
		},
	}

	for _, tt := range tests {
		client, fakeClientSet := FakeNew()

		fakeClientSet.Kubernetes.PrependReactor("list", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, &tt.secretList, nil
		})

		secretsList, err := client.ListSecrets("")

		if !reflect.DeepEqual(tt.output, secretsList) {
			t.Errorf("expected output: %#v,got: %#v", tt.secretList, secretsList)
		}

		if err == nil && !tt.wantErr {
			if len(fakeClientSet.Kubernetes.Actions()) != 1 {
				t.Errorf("expected 1 action in ListSecrets got: %v", fakeClientSet.Kubernetes.Actions())
			}
		} else if err == nil && tt.wantErr {
			t.Error("test failed, expected: false, got true")
		} else if err != nil && !tt.wantErr {
			t.Errorf("test failed, expected: no error, got error: %s", err.Error())
		}
	}
}

func TestWaitAndGetPod(t *testing.T) {

	tests := []struct {
		name    string
		podName string
		status  corev1.PodPhase
		wantErr bool
	}{
		{
			name:    "phase: running",
			podName: "ruby",
			status:  corev1.PodRunning,
			wantErr: false,
		},

		{
			name:    "phase: failed",
			podName: "ruby",
			status:  corev1.PodFailed,
			wantErr: true,
		},

		{
			name: "phase:	unknown",
			podName: "ruby",
			status:  corev1.PodUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()
			fkWatch := watch.NewFake()

			// Change the status
			go func() {
				fkWatch.Modify(fakePodStatus(tt.status, tt.podName))
			}()

			fkclientset.Kubernetes.PrependWatchReactor("pods", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				return true, fkWatch, nil
			})

			podSelector := fmt.Sprintf("deployment=%s", tt.podName)
			timeout := int64(10)
			watchOptions := metav1.ListOptions{
				LabelSelector:  podSelector,
				TimeoutSeconds: &timeout,
			}
			pod, err := fkclient.WaitAndGetPod(watchOptions, corev1.PodRunning, "Waiting for component to start")

			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.WaitAndGetPod(string) unexpected error %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(fkclientset.Kubernetes.Actions()) != 1 {
				t.Errorf("expected 1 action in WaitAndGetPod got: %v", fkclientset.Kubernetes.Actions())
			}

			if err == nil {
				if pod.Name != tt.podName {
					t.Errorf("pod name is not matching to expected name, expected: %s, got %s", tt.podName, pod.Name)
				}
			}

		})
	}
}

func TestWaitAndGetSecret(t *testing.T) {

	tests := []struct {
		name       string
		secretName string
		namespace  string
		wantErr    bool
	}{
		{
			name:       "Case 1: no error expected",
			secretName: "ruby",
			namespace:  "dummy",
			wantErr:    false,
		},

		{
			name:       "Case 2: error expected",
			secretName: "",
			namespace:  "dummy",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fkclient, fkclientset := FakeNew()
			fkWatch := watch.NewFake()

			// Change the status
			go func() {
				fkWatch.Modify(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.secretName,
					},
				})
			}()

			fkclientset.Kubernetes.PrependWatchReactor("secrets", func(action ktesting.Action) (handled bool, ret watch.Interface, err error) {
				if len(tt.secretName) == 0 {
					return true, nil, fmt.Errorf("error watching secret")
				}
				return true, fkWatch, nil
			})

			pod, err := fkclient.WaitAndGetSecret(tt.secretName, tt.namespace)

			if !tt.wantErr == (err != nil) {
				t.Errorf(" client.WaitAndGetSecret(string, string) unexpected error %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(fkclientset.Kubernetes.Actions()) != 1 {
				t.Errorf("expected 1 action in WaitAndGetSecret got: %v", fkclientset.Kubernetes.Actions())
			}

			if err == nil {
				if pod.Name != tt.secretName {
					t.Errorf("secret name is not matching to expected name, expected: %s, got %s", tt.secretName, pod.Name)
				}
			}
		})
	}
}

func TestCreateService(t *testing.T) {
	tests := []struct {
		name             string
		commonObjectMeta metav1.ObjectMeta
		containerPorts   []corev1.ContainerPort
		wantErr          bool
	}{
		{
			name: "Test case: with valid commonObjectName and containerPorts",
			commonObjectMeta: metav1.ObjectMeta{
				Name: "nodejs",
				Labels: map[string]string{
					"app":                              "apptmp",
					"app.kubernetes.io/component-name": "ruby",
					"app.kubernetes.io/component-type": "ruby",
					"app.kubernetes.io/name":           "apptmp",
				},
				Annotations: map[string]string{
					"app.kubernetes.io/url":                   "https://github.com/openshift/ruby",
					"app.kubernetes.io/component-source-type": "git",
				},
			},
			containerPorts: []corev1.ContainerPort{
				{
					Name:          "8080-tcp",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "9100-udp",
					ContainerPort: 9100,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fkclient, fkclientset := FakeNew()

			_, err := fkclient.CreateService(tt.commonObjectMeta, tt.containerPorts)

			if err == nil && !tt.wantErr {
				if len(fkclientset.Kubernetes.Actions()) != 1 {
					t.Errorf("expected 1 Kubernetes.Actions() in CreateService, got: %v", fkclientset.Kubernetes.Actions())
				}
				createdSvc := fkclientset.Kubernetes.Actions()[0].(ktesting.CreateAction).GetObject().(*corev1.Service)
				if !reflect.DeepEqual(tt.commonObjectMeta, createdSvc.ObjectMeta) {
					t.Errorf("ObjectMeta does not match the expected name, expected: %v, got: %v", tt.commonObjectMeta, createdSvc.ObjectMeta)
				}
				if !reflect.DeepEqual(tt.commonObjectMeta.Name, createdSvc.Spec.Selector["deployment"]) {
					t.Errorf("selector value does not match the expected name, expected: %s, got: %s", tt.commonObjectMeta.Name, createdSvc.Spec.Selector["deployment"])
				}
				for _, port := range tt.containerPorts {
					found := false
					for _, servicePort := range createdSvc.Spec.Ports {
						if servicePort.Port == port.ContainerPort {
							found = true
							if servicePort.Protocol != port.Protocol {
								t.Errorf("service protocol does not match the expected name, expected: %s, got: %s", port.Protocol, servicePort.Protocol)
							}
							if servicePort.Name != port.Name {
								t.Errorf("service name does not match the expected name, expected: %s, got: %s", port.Name, servicePort.Name)
							}
							if servicePort.TargetPort != intstr.FromInt(int(port.ContainerPort)) {
								t.Errorf("target port does not match the expected name, expected: %v, got: %v", intstr.FromInt(int(port.ContainerPort)), servicePort.TargetPort)
							}
						}
					}
					if found == false {
						t.Errorf("expected service port %s not found in the created Service", tt.name)
						break
					}
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func TestGetDeploymentFromSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		label    map[string]string
		wantErr  bool
	}{
		{
			name:     "true case",
			selector: "app.kubernetes.io/name=app",
			label: map[string]string{
				"app.kubernetes.io/name": "app",
			},
			wantErr: false,
		},
		{
			name:     "true case",
			selector: "app.kubernetes.io/name=app1",
			label: map[string]string{
				"app.kubernetes.io/name": "app",
			},
			wantErr: false,
		},
	}

	listOfDep := appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "app",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, fakeClientSet := FakeNew()

			fakeClientSet.Kubernetes.PrependReactor("list", "deployment", func(action ktesting.Action) (bool, runtime.Object, error) {
				if !reflect.DeepEqual(action.(ktesting.ListAction).GetListRestrictions().Labels.String(), tt.selector) {
					return true, nil, fmt.Errorf("labels not matching with expected values, expected:%s, got:%s", tt.selector, action.(ktesting.ListAction).GetListRestrictions())
				}
				return true, &listOfDep, nil
			})
			dep, err := fakeClient.GetDeploymentsFromSelector(tt.selector)

			if len(fakeClientSet.Kubernetes.Actions()) != 1 {
				t.Errorf("expected 1 AppsClientset.Actions() in GetDeploymentsFromSelector, got: %v", fakeClientSet.Kubernetes.Actions())
			}

			if tt.wantErr == false && err != nil {
				t.Errorf("test failed, %#v", dep[0].Labels)
			}

			for _, dep1 := range dep {
				if !reflect.DeepEqual(dep1.Labels, tt.label) {
					t.Errorf("labels are not matching with expected labels, expected: %s, got %s", tt.label, dep1.Labels)
				}
			}

		})
	}
}

func TestUniqueAppendOrOverwriteEnvVars(t *testing.T) {
	tests := []struct {
		name            string
		existingEnvVars []corev1.EnvVar
		envVars         []corev1.EnvVar
		want            []corev1.EnvVar
	}{
		{
			name: "Case: Overlapping env vars appends",
			existingEnvVars: []corev1.EnvVar{
				{
					Name:  "key1",
					Value: "value1",
				},
				{
					Name:  "key2",
					Value: "value2",
				},
			},
			envVars: []corev1.EnvVar{
				{
					Name:  "key1",
					Value: "value3",
				},
				{
					Name:  "key2",
					Value: "value4",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "key1",
					Value: "value3",
				},
				{
					Name:  "key2",
					Value: "value4",
				},
			},
		},
		{
			name: "New env vars append",
			existingEnvVars: []corev1.EnvVar{
				{
					Name:  "key1",
					Value: "value1",
				},
				{
					Name:  "key2",
					Value: "value2",
				},
			},
			envVars: []corev1.EnvVar{
				{
					Name:  "key3",
					Value: "value3",
				},
				{
					Name:  "key4",
					Value: "value4",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "key1",
					Value: "value1",
				},
				{
					Name:  "key2",
					Value: "value2",
				},
				{
					Name:  "key3",
					Value: "value3",
				},
				{
					Name:  "key4",
					Value: "value4",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnvVars := uniqueAppendOrOverwriteEnvVars(tt.existingEnvVars, tt.envVars...)
			if len(tt.want) != len(gotEnvVars) {
				t.Errorf("Tc: %s, expected %+v, got %+v", tt.name, tt.want, gotEnvVars)
			}
			matchFound := false
			for _, wantEnv := range tt.want {
				for _, gotEnv := range gotEnvVars {
					if reflect.DeepEqual(wantEnv, gotEnv) {
						matchFound = true
					}
				}
				if !matchFound {
					t.Errorf("Tc: %s, expected %+v, got %+v", tt.name, tt.want, gotEnvVars)
				}
			}
		})
	}
}

func TestDeleteEnvVars(t *testing.T) {
	tests := []struct {
		name           string
		existingEnvs   []corev1.EnvVar
		envTobeDeleted string
		want           []corev1.EnvVar
	}{
		{
			name: "Case 1: valid case of delete",
			existingEnvs: []corev1.EnvVar{
				{
					Name:  "abc",
					Value: "123",
				},
				{
					Name:  "def",
					Value: "456",
				},
			},
			envTobeDeleted: "def",
			want: []corev1.EnvVar{
				{
					Name:  "abc",
					Value: "123",
				},
			},
		},
		{
			name: "Case 2: valid case of delete non-existant env",
			existingEnvs: []corev1.EnvVar{
				{
					Name:  "abc",
					Value: "123",
				},
				{
					Name:  "def",
					Value: "456",
				},
			},
			envTobeDeleted: "ghi",
			want: []corev1.EnvVar{
				{
					Name:  "abc",
					Value: "123",
				},
				{
					Name:  "def",
					Value: "456",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deleteEnvVars(tt.existingEnvs, tt.envTobeDeleted)
			// Verify the passed param is not changed after call to function
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %+v, want: %+v", got, tt.want)
			}
		})
	}
}

func Test_GetInputEnvVarsFromStrings(t *testing.T) {
	tests := []struct {
		name          string
		envVars       []string
		wantedEnvVars []corev1.EnvVar
		wantErr       bool
	}{
		{
			name:    "Test case 1: with valid two key value pairs",
			envVars: []string{"key=value", "key1=value1"},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "value1",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 2: one env var with missing value",
			envVars: []string{"key=value", "key1="},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 3: one env var with no value and no =",
			envVars: []string{"key=value", "key1"},
			wantErr: true,
		},
		{
			name:    "Test case 4: one env var with multiple values",
			envVars: []string{"key=value", "key1=value1=value2"},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "value1=value2",
				},
			},
			wantErr: false,
		},
		{
			name:    "Test case 5: two env var with same key",
			envVars: []string{"key=value", "key=value1"},
			wantErr: true,
		},
		{
			name:    "Test case 6: one env var with base64 encoded value",
			envVars: []string{"key=value", "key1=SSd2ZSBnb3QgYSBsb3ZlbHkgYnVuY2ggb2YgY29jb251dHMhCg=="},
			wantedEnvVars: []corev1.EnvVar{
				{
					Name:  "key",
					Value: "value",
				},
				{
					Name:  "key1",
					Value: "SSd2ZSBnb3QgYSBsb3ZlbHkgYnVuY2ggb2YgY29jb251dHMhCg==",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars, err := GetInputEnvVarsFromStrings(tt.envVars)

			if err == nil && !tt.wantErr {
				if !reflect.DeepEqual(tt.wantedEnvVars, envVars) {
					t.Errorf("corev1.Env values are not matching with expected values, expected: %v, got %v", tt.wantedEnvVars, envVars)
				}
			} else if err == nil && tt.wantErr {
				t.Error("error was expected, but no error was returned")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, no error was expected, but got unexpected error: %s", err)
			}
		})
	}
}

func Test_findContainer(t *testing.T) {
	type args struct {
		name       string
		containers []corev1.Container
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Case 1 - Find the container",
			args: args{
				name: "foo",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Case 2 - Error if container not found",
			args: args{
				name: "foo2",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Case 3 - Error when passing in blank container name",
			args: args{
				name: "",
				containers: []corev1.Container{
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Case 4 - Check against multiple containers (rather than one)",
			args: args{
				name: "foo",
				containers: []corev1.Container{
					{
						Name: "bar",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
					{
						Name: "foo",
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/tmp",
								Name:      "test-pvc",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Run function findContainer
			container, err := FindContainer(tt.args.containers, tt.args.name)

			// Check that the container matches the name
			if err == nil && container.Name != tt.args.name {
				t.Errorf("Wrong container returned, wanted container %v, got %v", tt.args.name, container.Name)
			}

			if err == nil && tt.wantErr {
				t.Error("test failed, expected: false, got true")
			} else if err != nil && !tt.wantErr {
				t.Errorf("test failed, expected: no error, got error: %s", err.Error())
			}

		})
	}
}
