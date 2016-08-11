package cmd

import (
	"github.com/shan1024/wum-uc/util"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
		and usage of using your command. For example:

		Cobra is a CLI library for Go that empowers applications.
		This application is a tool to generate the needed files
		to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 1 {
			util.PrintErrorAndExit("Invalid number of argumants. Run with --help for more details about the argumants")
		}
		createUpdate(args[0], args[1], enableDebugLogsForCreateCommand, enableTraceLogsForCreateCommand)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	RootCmd.Flags().BoolVarP(&enableDebugLogsForCreateCommand, "debug", "d", false, "Enable debug logs")
	RootCmd.Flags().BoolVarP(&enableTraceLogsForCreateCommand, "trace", "t", false, "Enable trace logs")
}

func startInit() {

}