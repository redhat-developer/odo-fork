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
	Scenarios []SpecScenario `yaml:"scenario"`
}

// SpecDev represents the fields under `spec.dev` in the idp.yaml
type SpecDev struct {
	Watched       WatchedFiles         `yaml:"watched"`
	UploadFilter  UploadFilter         `yaml:"uploadFilter"`
	TypeDetection []TypeDetectionEntry `yaml:"typeDetection"`
}

// SpecRuntime defines the runtime image to be used in the IDP, contains info such as docker image, ports, env vars, etc.
type SpecRuntime struct {
	Image      string             `yaml:"image"`
	Endpoints  RuntimeEndpoints   `yaml:"endpoints"`
	Ports      RuntimePorts       `yaml:"ports"`
	Logs       []Logs             `yaml:"logs"`
	Env        []EnvVar           `yaml:"env"`
	Kubernetes KubernetesSettings `yaml:"kubernetes"`
}

// SpecShared represents shared settings and values to be used across the runtime and build containers
type SpecShared struct {
	Tasks   SharedTasks     `yaml:"tasks"`
	Volumes []SharedVolumes `yaml:"volumes"`
	Env     []EnvVar        `yaml:"env"`
}

// SpecTask represents an IDP build task/step
type SpecTask struct {
	Name             string             `yaml:"name"`
	BuildImage       string             `yaml:"buildImage"`
	Command          string             `yaml:"command"`
	WorkingDirectory string             `yaml:"workingDirectory"`
	Logs             []Logs             `yaml:"logs"`
	VolumeMappings   []Mappings         `yaml:"volumeMappings"`
	RepoMappings     []Mappings         `yaml:"repoMappings"`
	SourceMappings   []Mappings         `yaml:"sourceMappings"`
	RunAsUser        int                `yaml:"runAsUser"`
	Kubernetes       KubernetesSettings `yaml:"kubernetes"`
	Env              []EnvVar           `yaml:"env"`
}

type SpecScenario struct {
	Name  string   `yaml:"name"`
	Tasks []string `yaml:"tasks"`
}

// KubernetesSettings represents readiness/liveness probes for use on Kube
type KubernetesSettings struct {
	LivenessProbe  KubeProbe `yaml:"livenessProbe"`
	ReadinessProbe KubeProbe `yaml:"readinessProbe"`
	MemoryLimit    string    `yaml:"memoryLimit"`
	RunAsUser      int       `yaml:"runAsUser"`
}

// KubeProbe represents a kubernetes liveness / readiness probe
type KubeProbe struct {
	InitialDelaySeconds int `yaml:"initialDelay"`
	TimeoutSeconds      int `yaml:"timeoutSeconds"`
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

// SharedTasks describes common settings to be applied to all of the tasks in the IDP.yaml
type SharedTasks struct {
	DisposeOfSharedContainersOnTaskComplete bool   `yaml:"disposeOfSharedContainersOnTaskComplete"`
	DisposeOnScenarioComplete               string `yaml:"disposeOnScenarioComplete"`
	IdleTaskContainerTimeout                int    `yaml:"idleTaskContainerTimeout"`
}

// SharedVolumes tells udo what RWX volumes to create (and how many) for the specific IDP
// This does not override any volumes that the use may add with `udo volume ...`
type SharedVolumes struct {
	Name   string `yaml:"name"`
	Labels string `yaml:"labels"`
	size   int    `yaml:"size"`
}

// EnvVar represents a key/value mapping of environment vars to use in runtime and build containers
type EnvVar struct {
	key   string `yaml:"key"`
	value string `yaml:"value"`
}

type Mappings struct {
	SrcPath       string `yaml:"srcPath"`
	DestPath      string `yaml:"destPath"`
	SetExecuteBit bool   `yaml:"setExecuteBit"`
}
