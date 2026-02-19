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

When run with no subcommand, launches the web UI and opens your browser.

Subcommands are available for CLI-based operation:
  fers-calc calculate config.yaml
  fers-calc monte-carlo config.yaml --runs 5000
  fers-calc serve --port 3000 --no-open
  fers-calc --help`,
	// When no subcommand is given, launch the web UI (same as "serve")
	Run: runServe,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")

	// Web server flags on the root command (used when no subcommand is given)
	rootCmd.Flags().StringP("port", "p", "8080", "Port for the web UI server")
	rootCmd.Flags().Bool("no-open", false, "Don't auto-open the browser")
	rootCmd.Flags().Bool("no-idle-shutdown", false, "Don't auto-exit when the browser is closed")
}
