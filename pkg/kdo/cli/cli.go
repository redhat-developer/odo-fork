package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/redhat-developer/odo-fork/pkg/common"
	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/catalog"
	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/component"
	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/config"
	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/url"
	"github.com/redhat-developer/odo-fork/pkg/kdo/cli/version"
	"github.com/redhat-developer/odo-fork/pkg/kdo/genericclioptions"
	kdoutil "github.com/redhat-developer/odo-fork/pkg/kdo/util"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

// udoRecommendedName is the recommended udo command name
const UdoRecommendedName = "udo"

var (
	udoLong = ktemplates.LongDesc(`
(Universal Do) udo is a CLI tool for running cloud native applications in a fast and automated matter. Reducing the complexity of deployment, udo adds iterative development without the worry of deploying your source code.

Find more information at https://github.com/redhat-developer/odo-fork`)
	udoExample = ktemplates.Examples(`  # Creating and deploying a Node.js project
  git clone https://github.com/openshift/nodejs-ex && cd nodejs-ex
  %[1]s create nodejs
  %[1]s push

  # Accessing your Node.js component
  %[1]s url create`)
	rootUsageTemplate = `Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Shortcuts:{{range .Commands}}{{if eq .Annotations.command "component"}}
  {{rpad .Name .NamePadding }} {{.Short}} (shortcut for "component {{.Name}}") {{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Commands:{{range .Commands}}{{if eq .Annotations.command "main"}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Utility Commands:{{range .Commands}}{{if or (eq .Annotations.command "utility") (eq .Name "help") }}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{CapitalizeFlagDescriptions .LocalFlags | trimRightSpace }}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

// NewCmdudo creates a new root command for udo
func NewCmdUdo(name, fullName string) *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     name,
		Short:   "udo (Universal Do)",
		Long:    udoLong,
		Example: fmt.Sprintf(udoExample, fullName),
	}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.udo.yaml)")

	rootCmd.PersistentFlags().Bool(genericclioptions.SkipConnectionCheckFlagName, false, "Skip cluster check")

	// Here we add the necessary "logging" flags.. However, we choose to hide some of these from the user
	// as they are not necessarily needed and more for advanced debugging
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.Set("logtostderr", "true")
	pflag.CommandLine.MarkHidden("alsologtostderr")
	pflag.CommandLine.MarkHidden("log_backtrace_at")
	pflag.CommandLine.MarkHidden("log_dir")
	pflag.CommandLine.MarkHidden("logtostderr")
	pflag.CommandLine.MarkHidden("stderrthreshold")

	// Override the verbosity flag description
	verbosity := pflag.Lookup("v")
	verbosity.Usage += ". Level varies from 0 to 9 (default 0)."

	rootCmd.SetUsageTemplate(rootUsageTemplate)
	cobra.AddTemplateFunc("CapitalizeFlagDescriptions", kdoutil.CapitalizeFlagDescriptions)

	rootCmd.AddCommand(
		catalog.NewCmdCatalog(catalog.RecommendedCommandName, kdoutil.GetFullName(fullName, catalog.RecommendedCommandName)),
		common.CmdPrintUdo,
		component.NewCmdCreate(component.CreateRecommendedCommandName, kdoutil.GetFullName(fullName, component.CreateRecommendedCommandName)),
		component.NewCmdDelete(component.DeleteRecommendedCommandName, kdoutil.GetFullName(fullName, component.DeleteRecommendedCommandName)),
		component.NewCmdPush(component.PushRecommendedCommandName, kdoutil.GetFullName(fullName, component.PushRecommendedCommandName)),
		version.NewCmdVersion(version.RecommendedCommandName, kdoutil.GetFullName(fullName, version.RecommendedCommandName)),
		config.NewCmdConfiguration(config.RecommendedCommandName, kdoutil.GetFullName(fullName, config.RecommendedCommandName)),
		url.NewCmdURL(url.RecommendedCommandName, kdoutil.GetFullName(fullName, url.RecommendedCommandName)),
	)

	kdoutil.VisitCommands(rootCmd, reconfigureCmdWithSubcmd)

	return rootCmd
}

// reconfigureCmdWithSubcmd reconfigures each root command with a list of all subcommands and lists them
// beside the help output
// Adapted from: https://github.com/cppforlife/knctl/blob/612840d3c9729b1c57b20ca0450acab0d6eceeeb/pkg/knctl/cmd/knctl.go#L224
func reconfigureCmdWithSubcmd(cmd *cobra.Command) {
	if len(cmd.Commands()) == 0 {
		return
	}

	if cmd.Args == nil {
		cmd.Args = cobra.ArbitraryArgs
	}
	if cmd.RunE == nil {
		cmd.RunE = ShowSubcommands
	}

	var strs []string
	for _, subcmd := range cmd.Commands() {
		if !subcmd.Hidden {
			strs = append(strs, strings.Split(subcmd.Use, " ")[0])
		}
	}

	cmd.Short += " (" + strings.Join(strs, ", ") + ")"
}

// ShowSubcommands shows all available subcommands.
// Adapted from: https://github.com/cppforlife/knctl/blob/612840d3c9729b1c57b20ca0450acab0d6eceeeb/pkg/knctl/cmd/knctl.go#L224
func ShowSubcommands(cmd *cobra.Command, args []string) error {
	var strs []string
	for _, subcmd := range cmd.Commands() {
		if !subcmd.Hidden {
			strs = append(strs, subcmd.Name())
		}
	}
	return fmt.Errorf("Subcommand not found, use one of the available commands: %s", strings.Join(strs, ", "))
}
