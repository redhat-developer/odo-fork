package build

const (
	// FullBuildTask is the relative path of the IDP full build task in the Persistent Volume's project directory
	FullBuildTask string = "/.udo/build-container-full.sh"
	// IncrementalBuildTask is the relative path of the IDP incremental build task in the Persistent Volume's project directory
	IncrementalBuildTask string = "/.udo/build-container-update.sh"

	// FullRunTask is the relative path of the IDP full run task in the Runtime Container Empty Dir Volume's project directory
	FullRunTask string = "/.udo/runtime-container-full.sh"
	// IncrementalRunTask is the relative path of the IDP incremental run task in the Runtime Container Empty Dir Volume's project directory
	IncrementalRunTask string = "/.udo/runtime-container-update.sh"

	// ReusableBuildContainer is a BuildTaskKind where udo will reuse the build container to build projects
	ReusableBuildContainer string = "ReusableBuildContainer"
	// KubeJob is a BuildTaskKind where udo will kick off a Kube job to build projects
	KubeJob string = "KubeJob"
	// Component is a BuildTaskKind where udo will deploy a component
	Component string = "Component"

	// BuildContainerImage holds the image name of the build task container
	BuildContainerImage string = "docker.io/maven:3.6"
	// BuildContainerName holds the container name of the build task container
	BuildContainerName string = "maven-build"
	// BuildContainerMountPath  holds the mount path of the build task container
	BuildContainerMountPath string = "/data/idp/"

	// RuntimeContainerImage is the default WebSphere Liberty Runtime image
	RuntimeContainerImage string = "websphere-liberty:19.0.0.3-webProfile7"
	// RuntimeContainerImageWithBuildTools is the default WebSphere Liberty Runtime image with Maven and Java installed
	RuntimeContainerImageWithBuildTools string = "maysunfaisal/libertymvnjava"
	// RuntimeContainerName is the runtime container name
	RuntimeContainerName string = "libertyproject"
	// RuntimeContainerMountPathDefault  holds the default mount path of the runtime task container
	RuntimeContainerMountPathDefault string = "/config"
	// RuntimeContainerMountPathEmptyDir  holds the empty dir mount path of the runtime task container
	RuntimeContainerMountPathEmptyDir string = "/home/default/idp"

	// IDPVolumeName holds the IDP volume name
	IDPVolumeName string = "idp-volume"
)
