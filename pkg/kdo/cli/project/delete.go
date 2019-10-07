package project

import (
	"fmt"

	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/ui"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/redhat-developer/odo-fork/pkg/project"

	// "github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/spf13/cobra"

	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const deleteRecommendedCommandName = "delete"

var (
	deleteExample = ktemplates.Examples(`
	# Delete a project
	%[1]s myproject  
	`)

	deleteLongDesc = ktemplates.LongDesc(`Delete a project and all resources deployed in the project being deleted`)

	deleteShortDesc = `Delete a project`
)

// ProjectDeleteOptions encapsulates the options for the odo project delete command
type ProjectDeleteOptions struct {
	// name of the project
	projectName string

	// force delete doesn't ask the user for confirmation
	projectForceDeleteFlag bool

	// generic context options common to all commands
	*genericclioptions.Context
}

// NewProjectDeleteOptions creates a ProjectDeleteOptions instance
func NewProjectDeleteOptions() *ProjectDeleteOptions {
	return &ProjectDeleteOptions{}
}

// Complete completes ProjectDeleteOptions after they've been created
func (pdo *ProjectDeleteOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	pdo.projectName = args[0]
	pdo.Context = genericclioptions.NewContext(cmd)
	return
}

// Validate validates the parameters of the ProjectDeleteOptions
func (pdo *ProjectDeleteOptions) Validate() (err error) {
	// Validate existence of the project to be deleted
	isValidProject, err := project.Exists(pdo.Context.Client, pdo.projectName)
	if !isValidProject {
		return fmt.Errorf("The project %s does not exist. Please check the list of projects using `odo project list`", pdo.projectName)
	}
	return
}

// Run runs the project delete command
func (pdo *ProjectDeleteOptions) Run() (err error) {
	// this to set the project in the file and runtime
	err = project.SetCurrent(pdo.Context.Client, pdo.projectName)
	if err != nil {
		return
	}
	err = printDeleteProjectInfo(pdo.Context.Client, pdo.projectName)
	if err != nil {
		return err
	}
	if pdo.projectForceDeleteFlag || ui.Proceed(fmt.Sprintf("Are you sure you want to delete project %v", pdo.projectName)) {
		err := project.Delete(pdo.Context.Client, pdo.projectName)
		if err != nil {
			return err
		}

		log.Infof("Deleted project : %v", pdo.projectName)
		return nil
	}

	return fmt.Errorf("Aborting deletion of project: %v", pdo.projectName)
}

// NewCmdProjectDelete creates the project delete command
func NewCmdProjectDelete(name, fullName string) *cobra.Command {
	o := NewProjectDeleteOptions()

	projectDeleteCmd := &cobra.Command{
		Use:     name,
		Short:   deleteShortDesc,
		Long:    deleteLongDesc,
		Example: fmt.Sprintf(deleteExample, fullName),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	projectDeleteCmd.Flags().BoolVarP(&o.projectForceDeleteFlag, "force", "f", false, "Delete project without prompting")

	return projectDeleteCmd
}
