package cli

import (
	"github.com/spf13/cobra"
)

var (
	// Version information (will be set by build flags)
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fers-calc",
	Short: "FERS Retirement Planning Calculator",
	Long: `A comprehensive retirement planning calculator for federal employees.

This tool helps you model various retirement scenarios, including:
- Deterministic projections with fixed assumptions
- Monte Carlo simulations with market uncertainty
- Break-even analysis between scenarios

Examples:
  fers-calc calculate config.yaml
  fers-calc monte-carlo config.yaml --runs 5000
  fers-calc --help`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no command is specified, show help
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")
}
