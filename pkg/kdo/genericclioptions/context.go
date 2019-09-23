package genericclioptions

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/redhat-developer/odo-fork/pkg/component"
	"github.com/redhat-developer/odo-fork/pkg/config"
	"github.com/redhat-developer/odo-fork/pkg/kclient"
	"github.com/redhat-developer/odo-fork/pkg/kdo/util"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/project"
	pkgUtil "github.com/redhat-developer/odo-fork/pkg/util"
)

// DefaultAppName is the default name of the application when an application name is not provided
const DefaultAppName = "app"

// NewContext creates a new Context struct populated with the current state based on flags specified for the provided command
func NewContext(command *cobra.Command) *Context {
	return newContext(command, false)
}

// NewContextCreatingAppIfNeeded creates a new Context struct populated with the current state based on flags specified for the
// provided command, creating the application if none already exists
func NewContextCreatingAppIfNeeded(command *cobra.Command) *Context {
	return newContext(command, true)
}

// Client returns an oc client configured for this command's options
func Client(command *cobra.Command) *kclient.Client {
	return client(command)
}

// ClientWithConnectionCheck returns an oc client configured for this command's options but forcing the connection check status
// to the value of the provided bool, skipping it if true, checking the connection otherwise
func ClientWithConnectionCheck(command *cobra.Command, skipConnectionCheck bool) *kclient.Client {
	return client(command, skipConnectionCheck)
}

// client creates an oc client based on the command flags, overriding the skip connection check flag with the optionally
// specified shouldSkipConnectionCheck boolean.
// We use varargs to denote the optional status of that boolean.
func client(command *cobra.Command, shouldSkipConnectionCheck ...bool) *kclient.Client {
	var skipConnectionCheck bool
	switch len(shouldSkipConnectionCheck) {
	case 0:
		var err error
		skipConnectionCheck, err = command.Flags().GetBool(SkipConnectionCheckFlagName)
		util.LogErrorAndExit(err, "")
	case 1:
		skipConnectionCheck = shouldSkipConnectionCheck[0]
	default:
		// safeguard: fail if more than one optional bool is passed because it would be a programming error
		log.Errorf("client function only accepts one optional argument, was given: %v", shouldSkipConnectionCheck)
		os.Exit(1)
	}

	client, err := kclient.New(skipConnectionCheck)
	util.LogErrorAndExit(err, "")

	return client
}

// checkProjectCreateOrDeleteOnlyOnInvalidNamespace errors out if user is trying to create or delete something other than project
// errFormatforCommand must contain one %s
func checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command *cobra.Command, errFormatForCommand string) {
	if command.HasParent() && command.Parent().Name() != "project" && (command.Name() == "create" || command.Name() == "delete") {
		err := fmt.Errorf(errFormatForCommand, command.Root().Name())
		util.LogErrorAndExit(err, "")
	}
}

// getFirstChildOfCommand gets the first child command of the root command of command
func getFirstChildOfCommand(command *cobra.Command) *cobra.Command {
	// If command does not have a parent no point checking
	if command.HasParent() {
		// Get the root command and set current command and its parent
		r := command.Root()
		p := command.Parent()
		c := command
		for {
			// if parent is root, then we have our first child in c
			if p == r {
				return c
			}
			// Traverse backwards making current command as the parent and parent as the grandparent
			c = p
			p = c.Parent()
		}
	}
	return nil
}

func getValidConfig(command *cobra.Command) (*config.LocalConfigInfo, error) {

	// Get details from config file
	configFileName := FlagValueIfSet(command, ContextFlagName)
	if configFileName != "" {
		fAbs, err := pkgUtil.GetAbsPath(configFileName)
		util.LogErrorAndExit(err, "")
		configFileName = fAbs
	}
	lci, err := config.NewLocalConfigInfo(configFileName)
	// if we could not create local config for some reason, return it
	if err != nil {
		return nil, err
	}

	// Now we need to ensure that local config exists and not allow specific commands
	// if that is the case This block contains cases where the non existence of local
	// config is ignored
	// Only if command has parent as if it is just root command then cobra handles it
	if command.HasParent() {
		// Gather nessasary info
		p := command.Parent()
		r := command.Root()
		afs := FlagValueIfSet(command, ApplicationFlagName)
		// Find the first child of the command. As some groups are allowed even with non existent config
		fcc := getFirstChildOfCommand(command)
		// This should not happen but just to be safe
		if fcc == nil {
			return nil, fmt.Errorf("Unable to get first child of command")
		}
		// Case 1 : if command is create operation just allow it
		if command.Name() == "create" && (p.Name() == "component" || p.Name() == r.Name()) {
			return lci, nil
		}
		// Case 2 : if command is describe or delete and app flag is used just allow it
		if (fcc.Name() == "describe" || fcc.Name() == "delete") && len(afs) > 0 {
			return lci, nil
		}
		// Case 2 : if command is list, just allow it
		if fcc.Name() == "list" {
			return lci, nil
		}
		// Case 3 : Check if fcc is project. If so, skip validation of context
		if fcc.Name() == "project" {
			return lci, nil
		}
		// Case 4 : Check if specific flags are set for specific first child commands
		if fcc.Name() == "app" {
			return lci, nil
		}
		// Check if fcc is build
		if fcc.Name() == "build" {
			return lci, nil
		}
		// Case 5 : Check if fcc is catalog and request is to list
		if fcc.Name() == "catalog" && p.Name() == "list" {
			return lci, nil
		}
		// Check if fcc is component and  request is list
		if fcc.Name() == "component" && command.Name() == "list" {
			return lci, nil
		}
		// Case 6 : Check if fcc is component and app flag is used
		if fcc.Name() == "component" && len(afs) > 0 {
			return lci, nil
		}
		// Case 7 : Check if fcc is logout and app flag is used
		if fcc.Name() == "logout" {
			return lci, nil
		}
		if fcc.Name() == "url" {
			return lci, nil
		}

	} else {
		return lci, nil
	}
	// * Ignore error block ends

	// If file does not exist at this point, raise an error
	if !lci.ConfigFileExists() {
		return nil, fmt.Errorf("The current directory does not represent an odo component. Use 'odo create' to create component here or switch to directory with a component")
	}
	// else simply return the local config info
	return lci, nil
}

// resolveProject resolves project
func resolveProject(command *cobra.Command, client *kclient.Client, lci *config.LocalConfigInfo) string {
	var ns string
	projectFlag := FlagValueIfSet(command, ProjectFlagName)
	var err error
	if len(projectFlag) > 0 {
		// if project flag was set, check that the specified project exists and use it
		_, err := project.Exists(client, projectFlag)
		util.LogErrorAndExit(err, "")
		ns = projectFlag
	} else {
		ns = lci.GetProject()
		if ns == "" {
			ns = project.GetCurrent(client)
			if len(ns) <= 0 {
				errFormat := "Could not get current project. Please create or set a project\n\t%s project create|set <project_name>"
				checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command, errFormat)
			}
		}

		// check that the specified project exists
		_, err = project.Exists(client, ns)
		if err != nil {
			e1 := fmt.Sprintf("You dont have permission to project '%s' or it doesnt exist. Please create or set a different project\n\t", ns)
			errFormat := fmt.Sprint(e1, "%s project create|set <project_name>")
			checkProjectCreateOrDeleteOnlyOnInvalidNamespace(command, errFormat)
		}
	}
	client.Namespace = ns
	return ns
}

// resolveApp resolves the app
func resolveApp(command *cobra.Command, createAppIfNeeded bool, lci *config.LocalConfigInfo) string {
	var app string
	appFlag := FlagValueIfSet(command, ApplicationFlagName)
	if len(appFlag) > 0 {
		app = appFlag
	} else {
		app = lci.GetApplication()
		if app == "" {
			if createAppIfNeeded {
				return DefaultAppName
			}
		}
	}
	return app
}

// resolveComponent resolves component
func resolveComponent(command *cobra.Command, lci *config.LocalConfigInfo, context *Context) string {
	var cmp string
	cmpFlag := FlagValueIfSet(command, ComponentFlagName)
	if len(cmpFlag) == 0 {
		// retrieve the current component if it exists if we didn't set the component flag
		cmp = lci.GetName()
	} else {
		// if flag is set, check that the specified component exists
		context.checkComponentExistsOrFail(cmpFlag)
		cmp = cmpFlag
	}
	return cmp
}

// UpdatedContext returns a new context updated from config file
func UpdatedContext(context *Context) (*Context, *config.LocalConfigInfo, error) {
	lci, err := getValidConfig(context.command)
	return newContext(context.command, true), lci, err
}

// newContext creates a new context based on the command flags, creating missing app when requested
func newContext(command *cobra.Command, createAppIfNeeded bool) *Context {
	client := client(command)

	// Get details from config file
	configFileName := FlagValueIfSet(command, ContextFlagName)
	if configFileName != "" {
		fAbs, err := pkgUtil.GetAbsPath(configFileName)
		util.LogErrorAndExit(err, "")
		configFileName = fAbs
	}

	// Check for valid config
	lci, err := getValidConfig(command)
	if err != nil {
		util.LogErrorAndExit(err, "")
	}

	// resolve project
	ns := resolveProject(command, client, lci)

	// resolve application
	app := resolveApp(command, createAppIfNeeded, lci)

	// resolve output flag
	outputFlag := FlagValueIfSet(command, OutputFlagName)

	// create the internal context representation based on calculated values
	internalCxt := internalCxt{
		Client:      client,
		Namespace:   ns,
		Application: app,
		OutputFlag:  outputFlag,
		command:     command,
	}

	// create a context from the internal representation
	context := &Context{
		internalCxt: internalCxt,
	}
	// once the component is resolved, add it to the context
	context.cmp = resolveComponent(command, lci, context)

	return context
}

// FlagValueIfSet retrieves the value of the specified flag if it is set for the given command
func FlagValueIfSet(cmd *cobra.Command, flagName string) string {
	flag, err := cmd.Flags().GetString(flagName)

	// log the error for debugging purposes though an error should only occur if the flag hadn't been added to the command or
	// if the specified flag name doesn't match a string flag. This usually can be ignored.
	ignoreButLog(err)
	return flag
}

// Context holds contextual information useful to commands such as correctly configured client, target project and application
// (based on specified flag values) and provides for a way to retrieve a given component given this context
type Context struct {
	internalCxt
}

// internalCxt holds the actual context values and is not exported so that it cannot be instantiated outside of this package.
// This ensures that Context objects are always created properly via NewContext factory functions.
type internalCxt struct {
	Client      *kclient.Client
	command     *cobra.Command
	Namespace   string
	Application string
	cmp         string
	OutputFlag  string
}

// Component retrieves the optionally specified component or the current one if it is set. If no component is set, exit with
// an error
func (o *Context) Component(optionalComponent ...string) string {
	return o.ComponentAllowingEmpty(false, optionalComponent...)
}

// ComponentAllowingEmpty retrieves the optionally specified component or the current one if it is set, allowing empty
// components (instead of exiting with an error) if so specified
func (o *Context) ComponentAllowingEmpty(allowEmpty bool, optionalComponent ...string) string {
	switch len(optionalComponent) {
	case 0:
		// if we're not specifying a component to resolve, get the current one (resolved in NewContext as cmp)
		// so nothing to do here unless the calling context doesn't allow no component to be set in which case we exit with error
		if !allowEmpty && len(o.cmp) == 0 {
			log.Errorf("No component is set")
			os.Exit(1)
		}
	case 1:
		cmp := optionalComponent[0]
		// only check the component if we passed a non-empty string, otherwise return the current component set in NewContext
		if len(cmp) > 0 {
			o.checkComponentExistsOrFail(cmp)
			o.cmp = cmp // update context
		}
	default:
		// safeguard: fail if more than one optional string is passed because it would be a programming error
		log.Errorf("ComponentAllowingEmpty function only accepts one optional argument, was given: %v", optionalComponent)
		os.Exit(1)
	}

	return o.cmp
}

// existsOrExit checks if the specified component exists with the given context and exits the app if not.
func (o *Context) checkComponentExistsOrFail(cmp string) {
	exists, err := component.Exists(o.Client, cmp, o.Application)
	util.LogErrorAndExit(err, "")
	if !exists {
		log.Errorf("Component %v does not exist in application %s", cmp, o.Application)
		os.Exit(1)
	}
}

// ignoreButLog logs a potential error when trying to resolve a flag value.
func ignoreButLog(err error) {
	if err != nil {
		glog.V(4).Infof("Ignoring error as it usually means flag wasn't set: %v", err)
	}
}

// ApplyIgnore will take the current ignores []string and either ignore it (if .odoignore is used)
// or find the .gitignore file in the directory and use that instead.
func ApplyIgnore(ignores *[]string, sourcePath string) (err error) {
	if len(*ignores) == 0 {
		rules, err := pkgUtil.GetIgnoreRulesFromDirectory(sourcePath)
		if err != nil {
			util.LogErrorAndExit(err, "")
		}
		*ignores = append(*ignores, rules...)
	}
	return nil
}
