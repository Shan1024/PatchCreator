// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"github.com/wso2/wum-uc/util"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate <update_loc> <dist_loc>",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 || len(args) > 2 {
			util.PrintErrorAndExit("Invalid number of argumants. Run with --help for more details about the argumants")
		}
		startValidation(args[0], args[1])
	},
}

func init() {
	RootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", false, "Enable debug logs")
	validateCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", false, "Enable trace logs")
}

//Entry point of the validate command
func startValidation(updateLocation, distributionLocation string) {

}

