package idp

// IDP represents an idp.yaml file
type IDP struct {
	APIVersion string   `yaml:"apiVersion"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata contains the metadata for an iterative-dev pack, such as language, framework and who maintains it
type Metadata struct {
	Name        string       `yaml:"name"`
	Language    string       `yaml:"language"`
	Framework   string       `yaml:"framework"`
	Version     string       `yaml:"version"`
	Maintainers []Maintainer `yaml:"maintainers"`
}

type Spec struct {
	Dev       SpecDev        `yaml:"dev"`
	Runtime   SpecRuntime    `yaml:"runtime"`
	Shared    SpecShared     `yaml:"shared"`
	Tasks     []SpecTask     `yaml:"tasks"`
	Scenarios []SpecScenario `yaml:"scenarios"`
}

// SpecDev represents the fields under `spec.dev` in the idp.yaml
type SpecDev struct {
	Watched       WatchedFiles         `yaml:"watched"`
	UploadFilter  UploadFilter         `yaml:"uploadFilter"`
	TypeDetection []TypeDetectionEntry `yaml:"typeDetection"`
}

// SpecRuntime defines the runtime image to be used in the IDP, contains info such as docker image, ports, env vars, etc.
type SpecRuntime struct {
	Image                        string             `yaml:"image"`
	OverrideEntrypointTailToFile string             `yaml:"overrideEntrypointTailToFile"`
	Endpoints                    RuntimeEndpoints   `yaml:"endpoints"`
	Ports                        RuntimePorts       `yaml:"ports"`
	Logs                         []Logs             `yaml:"logs"`
	Env                          []EnvVar           `yaml:"env"`
	VolumeMappings               []VolumeMapping    `yaml:"volumeMappings"`
	Kubernetes                   KubernetesSettings `yaml:"kubernetes"`
}

// SpecShared represents shared settings and values to be used across the runtime and build containers
type SpecShared struct {
	Containers []SharedContainer `yaml:"containers"`
	Volumes    []SharedVolume    `yaml:"volumes"`
	Env        []EnvVar          `yaml:"env"`
}

// SpecTask represents an IDP build task/step
type SpecTask struct {
	Name             string        `yaml:"name"`
	Type             string        `yaml:"type"`
	Container        string        `yaml:"container"`
	Command          []string      `yaml:"command"`
	WorkingDirectory string        `yaml:"workingDirectory"`
	Logs             []Logs        `yaml:"logs"`
	RepoMappings     []RepoMapping `yaml:"repoMappings"`
	SourceMapping    SourceMapping `yaml:"sourceMapping"`
	Env              []EnvVar      `yaml:"env"`
}

type SharedContainer struct {
	Name           string             `yaml:"name"`
	Image          string             `yaml:"image"`
	VolumeMappings []VolumeMapping    `yaml:"volumeMappings"`
	Env            []EnvVar           `yaml:"env"`
	Privileged     bool               `yaml:"privileged"`
	Kubernetes     KubernetesSettings `yaml:"kubernetes"`
}

type SpecScenario struct {
	Name  string   `yaml:"name"`
	Tasks []string `yaml:"tasks"`
}

// KubernetesSettings represents readiness/liveness probes for use on Kube
type KubernetesSettings struct {
	LivenessProbe  KubeProbe     `yaml:"livenessProbe"`
	ReadinessProbe KubeProbe     `yaml:"readinessProbe"`
	Requests       KubeResources `yaml:"requests"`
	Limits         KubeResources `yaml:"limits"`
}

// KubeProbe represents a kubernetes liveness / readiness probe
type KubeProbe struct {
	InitialDelaySeconds int `yaml:"initialDelaySeconds"`
	TimeoutSeconds      int `yaml:"timeoutSeconds"`
}

type KubeResources struct {
	Memory string `yaml:"memory"`
	CPU    string `yaml:"cpu"`
}

// Maintainer represents the maintainer(s) of an Iterative-Dev Pack
type Maintainer struct {
	Name     string `yaml:"name"`
	Email    string `yaml:"email"`
	GithubID string `yaml:"githubID"`
}

// WatchedFiles defines which files udo should watch for changes in for this IDP
type WatchedFiles struct {
	WatchDir     string   `yaml:"watchDir"`
	IgnoredPaths []string `yaml:"ignoredPaths"`
}

// UploadFilter specifies any file paths to be ignored when syncing files (like target/)
type UploadFilter struct {
	IgnoredPaths []string `yaml:"ignoredPaths"`
}

// TypeDetectionEntry defines the rules used to determine if a project is compatible with a specific IDP
type TypeDetectionEntry struct {
	PathSelector       PathSelector `yaml:"pathSelector"`
	TextStringsToMatch []string     `yaml:"textStringsToMatch"`
}

type PathSelector struct {
	RootPath         string `yaml:"rootPath"`
	FilenameWildcard string `yaml:"filenameWildcard"`
}

type RuntimeEndpoints struct {
	ContainerHealth string `yaml:"containerHealth"`
	AppRoot         string `yaml:"appRoot"`
}

// RuntimePorts specifies an optional set of ports to expose the user's application at for this IDP.
type RuntimePorts struct {
	InternalHTTPPort        string `yaml:"internalHttpPort"`
	InternalHTTPSPort       string `yaml:"internalHttpsPort"`
	InternalDebugPort       string `yaml:"internalDebugPort"`
	InternalPerformancePort string `yaml:"internalPerformancePort"`
}

// Logs describes a type of log file at a given path
type Logs struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

// SharedVolume tells udo what RWX volumes to create (and how many) for the specific IDP
// This does not override any volumes that the use may add with `udo volume ...`
type SharedVolume struct {
	Name string `yaml:"name"`
	Size string `yaml:"size"`
}

// EnvVar represents a key/value mapping of environment vars to use in runtime and build containers
type EnvVar struct {
	key   string `yaml:"key"`
	value string `yaml:"value"`
}

type RepoMapping struct {
	SrcPath       string `yaml:"srcPath"`
	DestPath      string `yaml:"destPath"`
	SetExecuteBit bool   `yaml:"setExecuteBit"`
}

type SourceMapping struct {
	DestPath      string `yaml:"destPath"`
	SetExecuteBit bool   `yaml:"setExecuteBit"`
}

type VolumeMapping struct {
	VolumeName    string `yaml:"volumeName"`
	ContainerPath string `yaml:"containerPath"`
	SubPath       string `yaml:"subPath"`
}
