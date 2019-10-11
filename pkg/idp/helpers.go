package idp

import (
	"errors"
	"fmt"
)

// GetScenario returns the scenario with the matching name or nil otherwise
func (i *IDP) GetScenario(name string) (SpecScenario, error) {
	for _, s := range i.Spec.Scenarios {
		if name == s.Name {
			return s, nil
		}
	}
	errMsg := fmt.Sprintf("No scenario found with the name %s", name)
	return SpecScenario{}, errors.New(errMsg)
}

// GetTasks returns the tasks for a scenario
func (i *IDP) GetTasks(scenario SpecScenario) []SpecTask {
	var tasks []SpecTask

	for _, t := range i.Spec.Tasks {
		for _, name := range scenario.Tasks {
			if name == t.Name {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks
}

// GetContainer returns the container for a given task
func (i *IDP) GetContainer(task SpecTask) (interface{}, error) {
	var taskContainer interface{}
	var err error
	if task.Type == RuntimeTask {
		taskContainer = i.Spec.Runtime
	} else {
		for _, c := range i.Spec.Shared.Containers {
			if c.Name == task.Container {
				taskContainer = c
			}
		}
	}
	if taskContainer == nil {
		err = errors.New("Task container not found")
	}
	return taskContainer, err
}

// GetPorts returns a list of ports that were set in the IDP. Unset ports will not be returned
func (i *IDP) GetPorts() []string {
	var portList []string
	if i.Spec.Runtime.Ports.InternalHTTPPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalHTTPPort)
	}
	if i.Spec.Runtime.Ports.InternalHTTPSPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalHTTPSPort)
	}
	if i.Spec.Runtime.Ports.InternalDebugPort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalDebugPort)
	}
	if i.Spec.Runtime.Ports.InternalPerformancePort != "" {
		portList = append(portList, i.Spec.Runtime.Ports.InternalPerformancePort)
	}

	return portList
}
