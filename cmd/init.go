// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
	"gopkg.in/yaml.v2"
	"github.com/spf13/viper"
)

var (
	initCmdUse = "init"
	initCmdShortDesc = "Generate '" + constant.UPDATE_DESCRIPTOR_FILE + "' file template"
	initCmdLongDesc = dedent.Dedent(`
		This command will generate the 'update-descriptor.yaml' file. If the
		user does not specify a location, it will generate the template in
		the current working directory.`)

	initCmdExample = dedent.Dedent(`update_number: 0001
platform_version: 4.4.0
platform_name: wilkes
applies_to: All the products based on carbon 4.4.1
bug_fixes:
  CARBON-15395: Upgrade Hazelcast version to 3.5.2
  <MORE_JIRAS_HERE>
description: |
  This update contain the relavent fixes for upgrading Hazelcast version
  to its latest 3.5.2 version. When applying this update it requires a
  full cluster estart since if the nodes has multiple client versions of
  Hazelcast it can cause issues during connectivity.
file_changes:
  added_files: []
  removed_files: []
  modified_files: []`)

	isPrintSampleSelected bool
)

// initCmd represents the validate command
var initCmd = &cobra.Command{
	Use: initCmdUse,
	Short: initCmdShortDesc,
	Long: initCmdLongDesc,
	Run: initializeInitCommand,
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&isPrintSampleSelected, "sample", "s", false, "Show sample file")
	initCmd.Flags().BoolP("process", "p", false, "Process README.txt file")
	viper.BindPFlag("ProcessReadMe", initCmd.Flags().Lookup("process"))
}

func initializeInitCommand(cmd *cobra.Command, args []string) {
	switch len(args) {
	case 0:
		if isPrintSampleSelected {
			fmt.Println(initCmdExample)
		} else {
			initCurrentDirectory()
		}
	case 1:
		initDirectory(args[0])
	default:
		util.PrintErrorAndExit("Invalid number of argumants. Run 'wum-uc init --help' to view help.")
	}
}

func initCurrentDirectory() {
	currentDirectory := "./"
	initDirectory(currentDirectory)
}

func initDirectory(destination string) {
	err := util.CreateDirectory(destination)
	util.HandleError(err)

	updateDescriptorFile := filepath.Join(destination, constant.UPDATE_DESCRIPTOR_FILE)
	updateDescriptor := util.UpdateDescriptor{}

	if viper.GetBool(constant.PROCESS_README) {
		processReadMe(&updateDescriptor)
	}

	data, err := yaml.Marshal(&updateDescriptor)
	dataString := string(data)
	dataString = strings.Replace(dataString, "\"", "", -1)
	if err != nil {
		util.HandleError(err)
	}

	file, err := os.OpenFile(
		updateDescriptorFile,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0600,
	)
	if err != nil {
		util.HandleError(err)
	}
	defer file.Close()

	// Write bytes to file
	_, err = file.Write([]byte(dataString))
	if err != nil {
		util.HandleError(err)
	}
	util.PrintInfo("'" + constant.UPDATE_DESCRIPTOR_FILE + "' has been successfully created.")

	util.PrintWhatsNext("run 'wum-uc init --sample' to view sample file.")
}

func processReadMe(updateDescriptor *util.UpdateDescriptor) {

}
