package idp

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

const (
	IDPBuildTaskFile   = "idp-build-tasks.yaml"
	IDPRuntimeTaskFile = "idp-runtime-tasks.yaml"
)

func TestGetScenario(t *testing.T) {

	idp, err := loadIDP(IDPBuildTaskFile)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", IDPBuildTaskFile)
		t.Fatal(err)
	}

	fmt.Printf("IDP name: %s\n", idp.Metadata.Name)

	tests := []struct {
		testName     string
		scenarioName string
		taskCount    int
		want         string
		wantErr      bool
	}{
		{
			testName:     "Test get valid scenario",
			scenarioName: "full-build",
			want:         "full-build",
			taskCount:    2,
		},
		{
			testName:     "Test get invalid scenario 1",
			scenarioName: "foo:3.5",
			want:         "",
			wantErr:      true,
		},
		{
			testName:     "Test get invalid scenario 2",
			scenarioName: "",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {

			scenario, err := idp.GetScenario(tt.scenarioName)

			if tt.wantErr && err == nil {
				t.Errorf("Expected an error, got success")
			} else if tt.wantErr == false && err != nil {
				t.Errorf("Error not expected: %s", err)
			}

			if tt.want != scenario.Name {
				t.Errorf("Expected %s, got %s", tt.want, scenario.Name)
			}

			if tt.taskCount != len(scenario.Tasks) {
				t.Errorf("Expected %d, got %d", tt.taskCount, len(scenario.Tasks))
			}
		})
	}
}

func TestGetTasksByScenario(t *testing.T) {

	idp, err := loadIDP(IDPBuildTaskFile)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", IDPBuildTaskFile)
		t.Fatal(err)
	}

	fmt.Printf("IDP name: %s\n", idp.Metadata.Name)

	tests := []struct {
		testName string
		scenario SpecScenario
		want     []string
		wantErr  bool
	}{
		{
			testName: "Test get tasks for valid scenario",
			scenario: SpecScenario{
				Name:  "full-build",
				Tasks: []string{"full-maven-build", "start-server"},
			},
			want: []string{"full-maven-build"},
		},
		{
			testName: "Test get tasks for invalid scenario",
			scenario: SpecScenario{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {

			tasks := idp.GetTasks(tt.scenario)

			if len(tt.want) != len(tasks) {
				t.Errorf("Expected %d, got %d", len(tt.want), len(tasks))
				t.FailNow()
			}

			for i := range tt.want {
				if tt.want[i] != tasks[i].Name {
					t.Errorf("Expected %s, got %s", tt.want[i], tasks[i].Name)
				}
			}
		})
	}
}

func TestGetContainerForTask(t *testing.T) {

	buildTaskIDP, err := loadIDP(IDPBuildTaskFile)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", IDPBuildTaskFile)
		t.Fatal(err)
	}

	runtimeTaskIDP, err := loadIDP(IDPRuntimeTaskFile)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", IDPRuntimeTaskFile)
		t.Fatal(err)
	}

	fmt.Printf("IDP build task name: %s\n", buildTaskIDP.Metadata.Name)
	fmt.Printf("IDP runtime task name: %s\n", runtimeTaskIDP.Metadata.Name)

	tests := []struct {
		testName      string
		idp           *IDP
		task          SpecTask
		containerType string
		want          string
		wantErr       bool
	}{
		{
			testName: "Test get container for valid build task",
			idp:      buildTaskIDP,
			task:     buildTaskIDP.Spec.Tasks[0],
			want:     "idp.TaskContainerInfo",
		},
		{
			testName: "Test get container for valid runtime task",
			idp:      runtimeTaskIDP,
			task:     runtimeTaskIDP.Spec.Tasks[0],
			want:     "idp.TaskContainerInfo",
		},
		{
			testName: "Test get container for invalid container reference",
			idp:      buildTaskIDP,
			task: SpecTask{
				Name:             "full-maven-build",
				Type:             "Shared",
				Container:        "not-exist",
				Command:          []string{"/data/idp/src/.udo/bin/build-container-full.sh"},
				WorkingDirectory: "/data/idp/src",
				Logs: []Logs{{
					Type: "maven.build",
					Path: "/logs/(etc)",
				}},
				RepoMappings: []RepoMapping{},
				SourceMapping: SourceMapping{
					DestPath:      "/data/idp/src",
					SetExecuteBit: true,
				},
				Env: []EnvVar{},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {

			container, err := tt.idp.GetTaskContainerInfo(tt.task)
			if tt.wantErr {
				if err != nil {
					return
				} else {
					t.Fatalf("Expected an error, got success")
				}
			}
			r := reflect.TypeOf(container)
			fmt.Printf("Container type %s\n", r)

			if tt.want != r.String() {
				t.Fatalf("Expected %s, got %s", tt.want, r)
			}
			// switch c := container.(type) {
			// case SpecRuntime:
			// 	fmt.Printf("Using a runtime container: %v\n", c)
			// case SharedContainer:
			// 	fmt.Printf("Using a build task container: %v\n", c)
			// default:
			// 	fmt.Printf("Did not find one of the expected container types\n")
			// }
		})
	}
}

func loadIDP(idpName string) (*IDP, error) {
	cwd, err := os.Getwd()
	if err != nil {
		err = errors.Wrap(err, "Failed to get current working directory")
		return &IDP{}, err
	}

	testIDPPath := "../../tests/resources/" + idpName
	testIDPFile := path.Join(cwd, testIDPPath)

	idpBytes, err := readIDPYaml(testIDPFile)
	if err != nil {
		err = errors.Wrapf(err, "Failed to read the test IDP file %s", testIDPFile)
		return &IDP{}, err
		// t.Fatalf("Failed to read the test IDP file %s", testIDPFile)
	}

	idp, err := parseIDPYaml(idpBytes)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", testIDPFile)
		return &IDP{}, err
		// t.Fatal(err)
	}
	return idp, err
}
