// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/util"
	"github.com/wso2/wum-uc/constant"
	"gopkg.in/yaml.v2"
)

var (
	validateCmdUse = "validate <update_loc> <dist_loc>"
	validateCmdShortDesc = "Validate update zip"
	validateCmdLongDesc = dedent.Dedent(`
		This command will validate the given update zip. Directory
		structure will be matched against the given distribution.`)
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use: validateCmdUse,
	Short: validateCmdShortDesc,
	Long: validateCmdLongDesc,
	Run: initializeValidateCommand,
}

func init() {
	RootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", util.EnableDebugLogs, "Enable debug logs")
	validateCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", util.EnableTraceLogs, "Enable trace logs")
}

func initializeValidateCommand(cmd *cobra.Command, args []string) {
	if len(args) < 2 || len(args) > 2 {
		util.PrintErrorAndExit("Invalid number of argumants. Run 'wum-uc validate --help' to view help.")
	}
	startValidation(args[0], args[1])
}

//Entry point of  the validate command
func startValidation(updateFilePath, distributionLocation string) {

	setLogLevel()
	logger.Debug("validate command called")

	updateFileMap := make(map[string]bool)
	distributionFileMap := make(map[string]bool)

	//Check update location
	//Check 1
	if !strings.HasSuffix(updateFilePath, ".zip") {
		util.PrintErrorAndExit("Entered update does not have a 'zip' extention.")
	} else {
		util.PrintInfo("Entered update file has a zip extension")
	}
	//Check 2
	exists, err := util.IsFileExists(updateFilePath)
	util.HandleError(err, "")
	if !exists {
		util.PrintErrorAndExit("Update file '" + updateFilePath + "' does not exist.")
	} else {
		util.PrintInfo("Entered update file exists")
	}

	if !strings.HasSuffix(distributionLocation, ".zip") {
		util.PrintErrorAndExit("Entered distribution does not have a 'zip' extention.")
	} else {
		util.PrintInfo("Entered distribution file has a zip extension")
	}

	exists, err = util.IsFileExists(distributionLocation)
	util.HandleError(err, "Error occurred while checking '" + distributionLocation + "'")
	if !exists {
		util.PrintErrorAndExit("Distribution does not exist at ", distributionLocation)
	} else {
		util.PrintInfo("Entered distribution exists")
	}

	//Check 3
	locationInfo, err := os.Stat(updateFilePath)
	util.HandleError(err, "")
	match, err := regexp.MatchString(constant.FILENAME_REGEX, locationInfo.Name())
	if !match {
		util.PrintErrorAndExit("Update file name(" + locationInfo.Name() + ") does not match '" + constant.FILENAME_REGEX + "' regular expression.")
	}
	updateName := strings.TrimSuffix(locationInfo.Name(), ".zip")
	viper.Set(constant.UPDATE_NAME, updateName)

	updateFileMap, updateDescriptor, err := readUpdateZip(updateFilePath)
	util.HandleError(err)
	logger.Debug(updateFileMap)

	//Check dist location
	if strings.HasSuffix(distributionLocation, ".zip") {
		locationInfo, err := os.Stat(distributionLocation)
		util.HandleError(err, "")
		viper.Set(constant.PRODUCT_NAME, strings.TrimSuffix(locationInfo.Name(), ".zip"))
		distributionFileMap, err = readDistributionZip(distributionLocation)
	} else {

	}
	err = compare(updateFileMap, distributionFileMap, updateDescriptor)
	util.HandleError(err)
	util.PrintInfo("'" + updateName + "' validation successfully finished.")
}

func compare(updateFileMap, distributionFileMap map[string]bool, updateDescriptor *util.UpdateDescriptor) error {
	updateName := viper.GetString(constant.UPDATE_NAME)
	for filePath := range updateFileMap {
		logger.Debug(fmt.Sprintf("Searching: %s", filePath))
		_, found := distributionFileMap[filePath]
		if !found {
			isInAddedFiles := util.IsStringIsInSlice(filePath, updateDescriptor.File_changes.Added_files)
			resourceFiles := getResourceFiles()
			logger.Debug(fmt.Sprintf("resourceFiles: %s", resourceFiles))
			fileName := strings.TrimPrefix(filePath, updateName + constant.PATH_SEPARATOR)
			logger.Debug(fmt.Sprintf("fileName: %s", fileName))
			_, foundInResources := resourceFiles[fileName]
			logger.Debug(fmt.Sprintf("found in resources: %s", foundInResources))
			if !isInAddedFiles && !foundInResources {
				return errors.New("File not found in the distribution: '" + filePath + "'. If this is a new file, add an entry to the 'added_files' sections in the '" + constant.UPDATE_DESCRIPTOR_FILE + "' file")
			} else {
				logger.Debug("'" + filePath + "' found in added files.")
			}
		}
	}
	return nil
}

func readUpdateZip(filename string) (map[string]bool, *util.UpdateDescriptor, error) {
	fileMap := make(map[string]bool)
	updateDescriptor := util.UpdateDescriptor{}

	isNotAContributionFileFound := false
	isASecPatch := false

	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, nil, err
	}
	defer zipReader.Close()

	updateName := viper.GetString(constant.UPDATE_NAME)
	logger.Debug("updateName:", updateName)
	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		if file.FileInfo().IsDir() {
			logger.Debug("dir:", file.Name)
			logger.Debug("dir:", file.FileInfo().Name())
			if file.FileInfo().Name() != updateName {
				logger.Debug("Checking:", file.FileInfo().Name())
				//Check 4
				prefix := filepath.Join(updateName, constant.CARBON_HOME)
				hasPrefix := strings.HasPrefix(file.Name, prefix)
				if !hasPrefix {
					return nil, nil, errors.New("Unknown directory found: '" + file.Name + "'")
				}
			}
		} else {
			logger.Debug("file:", file.Name)
			logger.Debug("file:", file.FileInfo().Name())
			fullPath := filepath.Join(updateName, file.FileInfo().Name())
			logger.Debug("fullPath:", fullPath)
			switch file.FileInfo().Name() {
			case constant.UPDATE_DESCRIPTOR_FILE:
				data, err := validateFile(file, constant.UPDATE_DESCRIPTOR_FILE, fullPath, updateName)
				if err != nil {
					return nil, nil, err
				}
				err = yaml.Unmarshal(data, &updateDescriptor)
				if err != nil {
					return nil, nil, err
				}
				//check
				err = util.ValidateUpdateDescriptor(&updateDescriptor)
				if err != nil {
					return nil, nil, errors.New("'" + constant.UPDATE_DESCRIPTOR_FILE + "' is invalid. " + err.Error())
				}
			case constant.LICENSE_FILE:
				data, err := validateFile(file, constant.LICENSE_FILE, fullPath, updateName)
				if err != nil {
					return nil, nil, err
				}
				dataString := string(data)
				if strings.Contains(dataString, "under Apache License 2.0") {
					isASecPatch = true
				}
			case constant.INSTRUCTIONS_FILE:
				_, err := validateFile(file, constant.INSTRUCTIONS_FILE, fullPath, updateName)
				if err != nil {
					return nil, nil, err
				}
			case constant.NOT_A_CONTRIBUTION_FILE:
				isNotAContributionFileFound = true
				_, err := validateFile(file, constant.NOT_A_CONTRIBUTION_FILE, fullPath, updateName)
				if err != nil {
					return nil, nil, err
				}
			default:
				resourceFiles := getResourceFiles()
				logger.Debug(fmt.Sprintf("resourceFiles: %s", resourceFiles))
				prefix := filepath.Join(updateName, constant.CARBON_HOME)
				hasPrefix := strings.HasPrefix(file.Name, prefix)
				_, foundInResources := resourceFiles[file.FileInfo().Name()]
				logger.Debug(fmt.Sprintf("foundInResources: %s", foundInResources))
				if !hasPrefix && !foundInResources {
					return nil, nil, errors.New("Unknown file found: '" + file.Name + "'")
				}
				relativePath := strings.TrimPrefix(file.Name, prefix + string(os.PathSeparator))
				fileMap[relativePath] = false
			}
		}
	}
	if !isASecPatch && !isNotAContributionFileFound {
		util.PrintWarning("This update is not a security update. But '" + constant.NOT_A_CONTRIBUTION_FILE + "' was not found. Please review and add '" + constant.NOT_A_CONTRIBUTION_FILE + "' file if necessary.")
	} else if isASecPatch && isNotAContributionFileFound {
		util.PrintWarning("This update is a security update. But '" + constant.NOT_A_CONTRIBUTION_FILE + "' was found. Please review and remove '" + constant.NOT_A_CONTRIBUTION_FILE + "' file if necessary.")
	}
	return fileMap, &updateDescriptor, nil
}

func validateFile(file *zip.File, fileName, fullPath, updateName string) ([]byte, error) {
	logger.Debug(fmt.Sprintf("Checking file: %s", fileName))
	if file.Name != fullPath {
		parent := strings.TrimSuffix(file.Name, file.FileInfo().Name())
		return nil, errors.New("'" + fileName + "' found at '" + parent + "'. It should be in the '" + updateName + "' directory")
	}
	zippedFile, err := file.Open()
	if err != nil {
		logger.Debug(fmt.Sprintf("Error occurred while opening the zip file: %v", err))
		return nil, err
	}
	data, err := ioutil.ReadAll(zippedFile)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error occurred while reading the zip file: %v", err))
		return nil, err
	}
	zippedFile.Close()

	dataString := string(data)
	//dataString = processData(dataString)
	dataString = processString(dataString, "\n", true)

	//check
	regex, err := regexp.Compile(constant.PATCH_REGEX)
	allMatches := regex.FindAllStringSubmatch(dataString, -1)
	logger.Debug(fmt.Sprintf("All matches: %v", allMatches))
	isPatchWordFound := false
	if len(allMatches) > 0 {
		isPatchWordFound = true
	}
	if isPatchWordFound {
		util.PrintWarning("'" + fileName + "' file contains the word 'patch' in following lines. Please review and change it to 'update' if possible.")
		//boldColor:=color.New(color.Bold)
		//boldColor.
		//red := color.New(color.FgRed).PrintfFunc()

		for i, line := range allMatches {
			util.PrintInfo(fmt.Sprintf("Matching Line #%d - %v", i + 1, line[0]))
		}
		fmt.Println()
	}
	return data, nil
}

func readDistributionZip(filename string) (map[string]bool, error) {
	fileMap := make(map[string]bool)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	productName := viper.GetString(constant.PRODUCT_NAME)
	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		logger.Trace(file.Name)
		relativePath := strings.TrimPrefix(file.Name, productName + string(os.PathSeparator))
		if !file.FileInfo().IsDir() {
			fileMap[relativePath] = false
		}
	}
	return fileMap, nil
}

//func processData(data string) string {
//	data = strings.TrimSpace(data)
//	data = strings.Replace(data, "\r", "\n", -1)
//	contains := strings.Contains(data, "\n")
//	if !contains {
//		return data
//	}
//	allLines := ""
//	lines := strings.Split(data, "\n")
//	for _, line := range lines {
//		allLines = allLines + strings.TrimRight(line, " ") + "\n"
//	}
//	return allLines
//}
