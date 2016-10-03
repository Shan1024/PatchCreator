// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
	"gopkg.in/yaml.v2"
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

	initCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", util.EnableDebugLogs, "Enable debug logs")
	initCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", util.EnableTraceLogs, "Enable trace logs")

	initCmd.Flags().BoolP("sample", "s", util.PrintSampleSelected, "Show sample file")
	viper.BindPFlag(constant.SAMPLE, initCmd.Flags().Lookup("sample"))

	initCmd.Flags().BoolP("process", "p", viper.GetBool(constant.PROCESS_README), "Process README.txt file")
	viper.BindPFlag(constant.PROCESS_README, initCmd.Flags().Lookup("process"))
}

func initializeInitCommand(cmd *cobra.Command, args []string) {
	logger.Debug("[Init] called")
	switch len(args) {
	case 0:
		if viper.GetBool(constant.SAMPLE) {
			logger.Debug("-s flag found. Printing sample.")
			fmt.Println(initCmdExample)
		} else {
			logger.Debug("-s flag not found. Initializing current working directory.")
			initCurrentDirectory()
		}
	case 1:
		logger.Debug("Initializing directory:", args[0])
		initDirectory(args[0])
	default:
		logger.Debug("Invalid number of argumants:", args)
		util.PrintErrorAndExit("Invalid number of argumants. Run 'wum-uc init --help' to view help.")
	}
}

func initCurrentDirectory() {
	currentDirectory := "./"
	initDirectory(currentDirectory)
}

func initDirectory(destination string) {

	logger.Debug("Initializing started.")
	exists, err := util.IsDirectoryExists(destination)
	logger.Debug(fmt.Sprintf("'%s' directory exists: %v", destination, exists))
	if !exists {
		util.PrintInfo(fmt.Sprintf("'%s' directory does not exist. Creating '%s' directory.", destination, destination))
		err := util.CreateDirectory(destination)
		util.HandleError(err)
	}
	updateDescriptorFile := filepath.Join(destination, constant.UPDATE_DESCRIPTOR_FILE)
	logger.Debug(fmt.Sprintf("updateDescriptorFile: %v", updateDescriptorFile))
	updateDescriptor := util.UpdateDescriptor{}

	if viper.GetBool(constant.PROCESS_README) {
		logger.Debug(constant.PROCESS_README + " enabled")
		processReadMe(destination, &updateDescriptor)
	} else {
		logger.Debug(constant.PROCESS_README + " disabled")
		setUpdateDescriptorDefaultValues(&updateDescriptor)
	}

	data, err := yaml.Marshal(&updateDescriptor)
	util.HandleError(err)

	dataString := string(data)
	//remove " enclosing the update number
	dataString = strings.Replace(dataString, "\"", "", -1)
	logger.Debug(fmt.Sprintf("update-descriptor:\n%s", dataString))

	file, err := os.OpenFile(
		updateDescriptorFile,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0600,
	)
	util.HandleError(err)
	defer file.Close()

	// Write bytes to file
	_, err = file.Write([]byte(dataString))
	if err != nil {
		util.HandleError(err)
	}
	util.PrintInfo(fmt.Sprintf("'%s' has been successfully created at '%s'.", constant.UPDATE_DESCRIPTOR_FILE, destination))

	util.PrintWhatsNext(fmt.Sprintf("run 'wum-uc init --sample' to view a sample '%s' file.", constant.UPDATE_DESCRIPTOR_FILE))
}

func setUpdateDescriptorDefaultValues(updateDescriptor *util.UpdateDescriptor) {
	logger.Debug("Setting default values:")
	logger.Debug(fmt.Sprintf("platform_name: %s", util.PlatformName_Default))
	updateDescriptor.Platform_name = util.PlatformName_Default
	logger.Debug(fmt.Sprintf("platform_version: %s", util.PlatformVersion_Default))
	updateDescriptor.Platform_version = util.PlatformVersion_Default
	bugFixes := map[string]string{
		util.BugFixes_Default: util.BugFixes_Default,
	}
	logger.Debug(fmt.Sprintf("bug_fixes: %v", bugFixes))
	updateDescriptor.Bug_fixes = bugFixes
}

func processReadMe(directory string, updateDescriptor *util.UpdateDescriptor) {
	logger.Debug("Processing README started")
	readMePath := path.Join(directory, constant.README_FILE)
	logger.Debug(fmt.Sprintf("README Path: %v", readMePath))
	_, err := os.Stat(readMePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug(fmt.Sprintf("%s not found", readMePath))
			setUpdateDescriptorDefaultValues(updateDescriptor)
			return
		}
	}
	data, err := ioutil.ReadFile(readMePath)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error occurred and processing README: %v", err))
		setUpdateDescriptorDefaultValues(updateDescriptor)
		return
	}

	logger.Debug("README.txt found")
	stringData := string(data)
	regex, err := regexp.Compile(constant.PATCH_ID_REGEX)
	if err == nil {
		result := regex.FindStringSubmatch(stringData)
		logger.Trace(fmt.Sprintf("PATCH_ID_REGEX result: %v", result))
		if len(result) == 3 {
			updateDescriptor.Update_number = result[2]
			updateDescriptor.Platform_version = result[1]
			platformsMap := viper.GetStringMapString(constant.PLATFORM_VERSIONS)
			logger.Trace(fmt.Sprintf("Platform Map: %v", platformsMap))
			platformName, found := platformsMap[result[1]]
			if found {
				logger.Debug("PlatformName found in configs")
				updateDescriptor.Platform_name = platformName
			} else {
				logger.Debug("No matching platform name found for:", result[1])
			}
		} else {
			logger.Debug("PATCH_ID_REGEX results incorrect:", result)
		}
	} else {
		logger.Debug(fmt.Sprintf("Error occurred while processing PATCH_ID_REGEX: %v", err))
	}

	regex, err = regexp.Compile(constant.APPLIES_TO_REGEX)
	if err == nil {
		result := regex.FindStringSubmatch(stringData)
		logger.Trace(fmt.Sprintf("APPLIES_TO_REGEX result: %v", result))
		if len(result) == 2 {
			updateDescriptor.Applies_to = util.ProcessString(result[1], ", ", true)
		} else if len(result) == 3 {
			updateDescriptor.Applies_to = util.ProcessString(strings.TrimSpace(result[1] + result[2]), ", ", true)
		} else {
			logger.Debug("No matching results found for APPLIES_TO_REGEX:", result)
		}
	} else {
		logger.Debug(fmt.Sprintf("Error occurred while processing APPLIES_TO_REGEX: %v", err))
	}

	regex, err = regexp.Compile(constant.ASSOCIATED_JIRAS_REGEX)
	if err == nil {
		allResult := regex.FindAllStringSubmatch(stringData, -1)
		logger.Trace(fmt.Sprintf("APPLIES_TO_REGEX result: %v", allResult))
		updateDescriptor.Bug_fixes = make(map[string]string)
		if len(allResult) == 0 {
			logger.Debug("No matching results found for ASSOCIATED_JIRAS_REGEX. Setting default values.")
			updateDescriptor.Bug_fixes[util.BugFixes_Default] = util.BugFixes_Default
		} else {
			logger.Debug("Matching results found for ASSOCIATED_JIRAS_REGEX")
			for i, match := range allResult {
				logger.Debug(fmt.Sprintf("%d: %s", i, match[1]))
				if len(match) == 2 {
					logger.Debug(fmt.Sprintf("ASSOCIATED_JIRAS_REGEX results is correct: %v", match))
					updateDescriptor.Bug_fixes[match[1]] = util.GetJiraSummary(match[1])
				} else {
					logger.Debug(fmt.Sprintf("ASSOCIATED_JIRAS_REGEX results incorrect: %v", match))
				}
			}
		}
	} else {
		logger.Debug(fmt.Sprintf("Error occurred while processing ASSOCIATED_JIRAS_REGEX: %v", err))
		logger.Debug("Setting defailt values to bug_fixes")
		updateDescriptor.Bug_fixes = make(map[string]string)
		updateDescriptor.Bug_fixes[util.BugFixes_Default] = util.BugFixes_Default
	}

	regex, err = regexp.Compile(constant.DESCRIPTION_REGEX)
	if err == nil {
		result := regex.FindStringSubmatch(stringData)
		logger.Trace(fmt.Sprintf("DESCRIPTION_REGEX result: %v", result))
		if len(result) == 2 {
			updateDescriptor.Description = util.ProcessString(result[1], "\n", false)
		} else {

		}
		logger.Debug(fmt.Sprintf("No matching results found for DESCRIPTION_REGEX: %v", result))
	} else {
		logger.Debug(fmt.Sprintf("Error occurred while processing DESCRIPTION_REGEX: %v", err))
	}
	logger.Debug("Processing README finished")
}
