package search

import (
	"fmt"

	"github.com/redhat-developer/odo-fork/pkg/catalog"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	"github.com/redhat-developer/odo-fork/pkg/log"
	"github.com/spf13/cobra"
)

const idpRecommendedCommandName = "idp"

var idpExample = `  # Search for an iterative-dev pack
  %[1]s python`

// SearchIDPOptions encapsulates the options for the udo catalog search idp command
type SearchIDPOptions struct {
	searchTerm   string
	localIDPRepo string
	idps         []string
}

// NewSearchIDPOptions creates a new SearchIDPOptions instance
func NewSearchIDPOptions() *SearchIDPOptions {
	return &SearchIDPOptions{}
}

// Complete completes SearchIDPOptions after they've been created
func (o *SearchIDPOptions) Complete(name string, cmd *cobra.Command, args []string) (err error) {
	o.searchTerm = args[0]
	o.idps, err = catalog.Search(o.searchTerm, o.localIDPRepo)
	return err
}

// Validate validates the SearchIDPOptions based on completed values
func (o *SearchIDPOptions) Validate() (err error) {
	if len(o.idps) == 0 {
		return fmt.Errorf("no iterative-dev pack matched the query: %s", o.searchTerm)
	}

	return
}

// Run contains the logic for the command associated with SearchIDPOptions
func (o *SearchIDPOptions) Run() (err error) {
	log.Infof("The following Iterative-Dev Packs were found:")
	for _, idp := range o.idps {
		fmt.Printf("- %v\n", idp)
	}
	return
}

// NewCmdCatalogSearchIDP implements the udo catalog search idp command
func NewCmdCatalogSearchIDP(name, fullName string) *cobra.Command {
	o := NewSearchIDPOptions()

	cmd := cobra.Command{
		Use:   name,
		Short: "Search Iterative-Dev Pack in catalog",
		Long: `Search Iterative-Dev Pack in catalog.

This searches for a partial match for the given search term in all the available
Iterative-Dev Packs.
`,
		Args:    cobra.ExactArgs(1),
		Example: fmt.Sprintf(idpExample, fullName),
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}

	// Adds flags to the command
	genericclioptions.AddLocalRepoFlag(&cmd, &o.localIDPRepo)

	return &cmd
}
