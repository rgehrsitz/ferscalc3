package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// exampleCmd represents the example command
var exampleCmd = &cobra.Command{
	Use:   "example [output-file]",
	Short: "Generate an example configuration file",
	Long: `Generate an example configuration file to help you get started.
If no output file is specified, the example will be written to stdout.

Example:
  fers-calc example config.yaml
  fers-calc example > my_config.yaml`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exampleConfig := `# FERS Retirement Planning Calculator - Example Configuration
# This is a sample configuration file showing all available options.

personal_details:
  person_a:
    name: "Alice Johnson"
    birth_date: "1975-03-15T00:00:00Z"
    hire_date: "2010-06-01T00:00:00Z"
    current_salary: 95000
    ss_benefit_62: 1800
    ss_benefit_fra: 2400
    ss_benefit_70: 3000
    high_3_salary: 92000
    tsp_balance: 350000
    tsp_allocation:
      c_fund: 40
      s_fund: 30
      i_fund: 20
      f_fund: 10
      g_fund: 0

  person_b:
    name: "Bob Johnson"
    birth_date: "1973-08-20T00:00:00Z"
    hire_date: "2008-09-15T00:00:00Z"
    current_salary: 85000
    ss_benefit_62: 1600
    ss_benefit_fra: 2200
    ss_benefit_70: 2750
    high_3_salary: 83000
    tsp_balance: 280000
    tsp_allocation:
      c_fund: 35
      s_fund: 25
      i_fund: 25
      f_fund: 15
      g_fund: 0

global_assumptions:
  inflation_rate: 0.025
  tsp_return_pre_retirement: 0.07
  tsp_return_post_retirement: 0.05
  cola_general_rate: 0.025
  projection_years: 30
  current_location:
    state: "Virginia"
    county: "Fairfax"

scenarios:
  - name: "Early Retirement at 62"
    person_a:
      employee_name: "person_a"
      retirement_date: "2037-03-15"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    person_b:
      employee_name: "person_b"
      retirement_date: "2037-03-15"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"

  - name: "Full Retirement at 67"
    person_a:
      employee_name: "person_a"
      retirement_date: "2042-03-15"
      ss_start_age: 67
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: 4000
    person_b:
      employee_name: "person_b"
      retirement_date: "2042-03-15"
      ss_start_age: 67
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: 3500
`

		if len(args) > 0 {
			// Write to file
			outputFile := args[0]
			if err := os.WriteFile(outputFile, []byte(exampleConfig), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing example config: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Example configuration written to %s\n", outputFile)
		} else {
			// Write to stdout
			fmt.Print(exampleConfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)
}
