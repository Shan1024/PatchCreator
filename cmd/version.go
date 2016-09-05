// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display wum-uc version information",
	Long: `Display wum-uc version information.`,
	Run: versionCommand,
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func versionCommand(cmd *cobra.Command, args []string) {
	fmt.Fprintf(os.Stdout, "wum version: %v\n", Version)
	fmt.Fprintf(os.Stdout, "Release date: %v\n", BuildDate)
	fmt.Fprintf(os.Stdout, "OS\\Arch: %v\\%v\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(os.Stdout, "Go version: %v\n\n", runtime.Version())
}
