package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/redhat-developer/odo-fork/pkg/catalog"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	"github.com/spf13/cobra"
)

const idpRecommendedCommandName = "idp"

var idpsExample = `  # Get the supported Iterative-dev Packs`

// ListIDPOptions encapsulates the options for the udo catalog list idp command
type ListIDPOptions struct {
	// list of known images
	localIDPRepo string
	catalogList  []catalog.CatalogEntry
}

// NewListIDPOptions creates a new ListIDPOptions instance
func NewListIDPOptions() *ListIDPOptions {
	return &ListIDPOptions{}
}

// Complete completes ListIDPOptions after they've been created
func (o *ListIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.catalogList, err = catalog.List(o.localIDPRepo)
	if err != nil {
		return err
	}

	return
}

// Validate validates the ListIDPOptions based on completed values
func (o *ListIDPOptions) Validate() (err error) {
	if len(o.catalogList) == 0 {
		return fmt.Errorf("no Iterative-Dev Packs found")
	}

	return err
}

// Run contains the logic for the command associated with ListIDPOptions
func (o *ListIDPOptions) Run() (err error) {
	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME", "\t", "LANGUAGE", "\t", "FRAMEWORK")
	for _, idp := range o.catalogList {
		idpName := idp.Name
		idpFramework := idp.Framework
		idpLanguage := idp.Language
		fmt.Fprintln(w, idpName, "\t", idpLanguage, "\t", idpFramework)
	}
	w.Flush()
	return
}

// NewCmdCatalogListIDPs implements the udo catalog list idps command
func NewCmdCatalogListIDPs(name, fullName string) *cobra.Command {
	o := NewListIDPOptions()

	cmd := cobra.Command{
		Use:     name,
		Short:   "List all IDPs available.",
		Long:    "List all available Iterative-Dev Packs.",
		Example: fmt.Sprintf(idpsExample, fullName),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	// Add the flags to the command
	genericclioptions.AddLocalRepoFlag(&cmd, &o.localIDPRepo)

	return &cmd

}
