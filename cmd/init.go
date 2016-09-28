// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
	"regexp"
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

	initCmd.Flags().BoolP("debug", "d", util.EnableDebugLogs, "Enable debug logs")
	viper.BindPFlag(constant.IS_DEBUG_ENABLED, initCmd.Flags().Lookup("debug"))

	initCmd.Flags().BoolP("trace", "t", util.EnableTraceLogs, "Enable trace logs")
	viper.BindPFlag(constant.IS_TRACE_ENABLED, initCmd.Flags().Lookup("trace"))

	initCmd.Flags().BoolP("sample", "s", util.PrintSampleSelected, "Show sample file")
	viper.BindPFlag(constant.SAMPLE, initCmd.Flags().Lookup("sample"))

	initCmd.Flags().BoolP("process", "p", viper.GetBool(constant.PROCESS_README), "Process README.txt file")
	viper.BindPFlag(constant.PROCESS_README, initCmd.Flags().Lookup("process"))
}

func initializeInitCommand(cmd *cobra.Command, args []string) {
	switch len(args) {
	case 0:
		if viper.GetBool(constant.SAMPLE) {
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
		fmt.Println("Process readme enabled")
		err := processReadMe(destination, &updateDescriptor)
		util.HandleError(err)
	} else {
		setUpdateDescriptorDefaultValues(&updateDescriptor)
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

func setUpdateDescriptorDefaultValues(updateDescriptor *util.UpdateDescriptor) {
	updateDescriptor.Platform_name = util.PlatformName_Default
	updateDescriptor.Platform_version = util.PlatformVersion_Default
	updateDescriptor.Bug_fixes = map[string]string{
		util.BugFixes_Default: util.BugFixes_Default,
	}
}

func processReadMe(directory string, updateDescriptor *util.UpdateDescriptor) error {
	fmt.Println("Processing readme started")
	readMePath := path.Join(directory, constant.README_FILE)
	_, err := os.Stat(readMePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(directory + " not found")
			return nil
		}
	}
	data, err := ioutil.ReadFile(readMePath)
	if err != nil {
		fmt.Println("error occurred and processing readme failed")
		return nil
	}

	stringData := string(data)

	fmt.Println("-----------------------------")
	regex, err := regexp.Compile(constant.PATCH_ID_REGEX)
	result := regex.FindStringSubmatch(stringData)
	if len(result) == 3 {
		updateDescriptor.Update_number = result[2]
		updateDescriptor.Platform_version = result[1]

		platformsMap := viper.GetStringMapString(constant.PLATFORM_VERSIONS)

		platformName, found := platformsMap[result[1]]
		if found {
			updateDescriptor.Platform_name = platformName
		} else {
			fmt.Println("No matching platform name found for:", result[1])
		}

	} else {
		fmt.Println("PATCH_ID_REGEX has incorrect results:", result)
	}

	fmt.Println("-----------------------------")
	regex, err = regexp.Compile(constant.APPLIES_TO_REGEX)
	result = regex.FindStringSubmatch(stringData)

	if len(result) == 2 {
		updateDescriptor.Applies_to = result[1]
	} else {
		fmt.Println("No matching results found for APPLIES_TO_REGEX:", result)
	}

	fmt.Println("-----------------------------")
	regex, err = regexp.Compile(constant.ASSOCIATED_JIRAS_REGEX)
	allResult := regex.FindAllStringSubmatch(stringData, -1)
	fmt.Println("ASSOCIATED_JIRAS_REGEX:")
	updateDescriptor.Bug_fixes = make(map[string]string)
	for _, match := range allResult {
		fmt.Println("match:", match[1])
		if len(match) == 2 {
			updateDescriptor.Bug_fixes[match[1]] = ""
		} else {
			fmt.Println("incorrect length for ASSOCIATED_JIRAS_REGEX:", match)
		}
	}
	fmt.Println("-----------------------------")
	regex, err = regexp.Compile(constant.DESCRIPTION_REGEX)
	result = regex.FindStringSubmatch(stringData)

	if len(result) == 2 {
		updateDescriptor.Description = result[1]
	} else {
		fmt.Println("No matching results found for DESCRIPTION_REGEX:", result)
	}

	fmt.Println("-----------------------------")

	fmt.Println("Processing readme finished")

	return nil
}
