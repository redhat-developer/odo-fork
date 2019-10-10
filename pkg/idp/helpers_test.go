package idp

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
)

func TestGetScenario(t *testing.T) {

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory")
	}

	testIDPFile := path.Join(cwd, "../../tests/resources/idp-build-tasks.yaml")

	idpBytes, err := readIDPYaml(testIDPFile)
	if err != nil {
		t.Fatalf("Failed to read the test IDP file %s", testIDPFile)
	}

	idp, err := parseIDPYaml(idpBytes)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", testIDPFile)
		t.Fatal(err)
	}

	fmt.Printf("IDP name: %s", idp.Metadata.Name)

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

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory")
	}

	testIDPFile := path.Join(cwd, "../../tests/resources/idp-build-tasks.yaml")

	idpBytes, err := readIDPYaml(testIDPFile)
	if err != nil {
		t.Fatalf("Failed to read the test IDP file %s", testIDPFile)
	}

	idp, err := parseIDPYaml(idpBytes)
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse the test IDP file %s", testIDPFile)
		t.Fatal(err)
	}

	fmt.Printf("IDP name: %s", idp.Metadata.Name)

	tests := []struct {
		testName string
		scenario SpecScenario
		want     []string
		wantErr  bool
	}{
		{
			testName: "Test get tasks for valid scenario",
			scenario: SpecScenario{},
			want:     []string{"full-maven-build"},
		},
		{
			testName: "Test get tasks for invalid scenario 1",
			scenario: SpecScenario{},
			want:     []string{},
			wantErr:  true,
		},
		{
			testName: "Test get tasks for invalid scenario 2",
			scenario: SpecScenario{},
			want:     []string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Log("Running test: ", tt.testName)
		t.Run(tt.testName, func(t *testing.T) {

			tasks := idp.GetTasksByScenario(tt.scenario)

			if tt.wantErr && err == nil {
				t.Errorf("Expected an error, got success")
			} else if tt.wantErr == false && err != nil {
				t.Errorf("Error not expected: %s", err)
			}

			if len(tt.want) != len(tasks) {
				t.Errorf("Expected %d, got %d", len(tt.want), len(tasks))
			}

			for i := range tt.want {
				if tt.want[i] != tasks[i].Name {
					t.Errorf("Expected %s, got %s", tt.want[i], tasks[i].Name)
				}
			}
		})
	}
}
