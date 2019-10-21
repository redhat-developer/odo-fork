package idp

import (
	"errors"
	"fmt"
)

//TaskContainerInfo is a struct that holds the basic necessary data to create a component
type TaskContainerInfo struct {
	Type           string
	Name           string
	Image          string
	VolumeMappings []VolumeMapping
	RuntimePorts   RuntimePorts
}

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

// GetTasks returns the tasks in the assigned order for a scenario
func (i *IDP) GetTasks(scenario SpecScenario) []SpecTask {
	var tasks []SpecTask

	for _, name := range scenario.Tasks {
		for _, t := range i.Spec.Tasks {
			if name == t.Name {
				tasks = append(tasks, t)
			}
		}
	}
	return tasks
}

// GetRuntimeInfo returns the container info for the Runtime
func (i *IDP) GetRuntimeInfo() TaskContainerInfo {
	taskContainerInfo := TaskContainerInfo{
		Type:           RuntimeTask,
		Name:           "",
		Image:          i.Spec.Runtime.Image,
		VolumeMappings: i.Spec.Runtime.VolumeMappings,
	}

	return taskContainerInfo
}

// GetTaskContainerInfo returns the container for a given task
func (i *IDP) GetTaskContainerInfo(task SpecTask) (TaskContainerInfo, error) {
	var taskContainerInfo TaskContainerInfo
	var err error
	if task.Type == RuntimeTask {
		taskContainerInfo = i.GetRuntimeInfo()
	} else {
		for _, c := range i.Spec.Shared.Containers {
			if c.Name == task.Container {
				taskContainerInfo.Type = SharedTask
				taskContainerInfo.Name = c.Name
				taskContainerInfo.Image = c.Image
				taskContainerInfo.VolumeMappings = c.VolumeMappings
			}
		}
	}
	if taskContainerInfo.Image == "" {
		err = errors.New("Task container not found")
	}
	return taskContainerInfo, err
}

// GetSharedVolumes returns the list of Shared Volumes
func (i *IDP) GetSharedVolumes() []SharedVolume {
	var sharedVolumes []SharedVolume

	for _, v := range i.Spec.Shared.Volumes {
		sharedVolumes = append(sharedVolumes, v)
	}
	return sharedVolumes
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
