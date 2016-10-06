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
		This command will validate the given update zip. Files
		will be matched against the given distribution.`)
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
	if len(args) != 2 {
		util.HandleErrorAndExit(nil, "Invalid number of argumants. Run 'wum-uc validate --help' to view help.")
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
	//Check
	if !strings.HasSuffix(updateFilePath, ".zip") {
		util.HandleErrorAndExit(nil, "Entered update does not have a 'zip' extention.")
	}

	//Check
	exists, err := util.IsFileExists(updateFilePath)
	util.HandleErrorAndExit(err, "")
	if !exists {
		util.HandleErrorAndExit(nil, fmt.Sprintf("Update file '%s' does not exist.", updateFilePath))
	}

	if !strings.HasSuffix(distributionLocation, ".zip") {
		util.HandleErrorAndExit(nil, "Entered distribution does not have a 'zip' extention.")
	}

	lastIndex := strings.LastIndex(distributionLocation, "/")
	viper.Set(constant.PRODUCT_NAME, strings.TrimSuffix(distributionLocation[lastIndex + 1:], ".zip"))

	distributionFileMap, err = readDistributionZip(distributionLocation)

	//check
	exists, err = util.IsFileExists(distributionLocation)
	util.HandleErrorAndExit(err, "Error occurred while checking '" + distributionLocation + "'")
	if !exists {
		util.HandleErrorAndExit(nil, "Distribution does not exist at ", distributionLocation)
	}

	//Check
	locationInfo, err := os.Stat(updateFilePath)
	util.HandleErrorAndExit(err, "")
	match, err := regexp.MatchString(constant.FILENAME_REGEX, locationInfo.Name())
	if !match {
		util.HandleErrorAndExit(nil, fmt.Sprintf("Update file name(%s) does not match '%s' regular expression.", locationInfo.Name(), constant.FILENAME_REGEX))
	}

	updateName := strings.TrimSuffix(locationInfo.Name(), ".zip")
	viper.Set(constant.UPDATE_NAME, updateName)

	updateFileMap, updateDescriptor, err := readUpdateZip(updateFilePath)
	util.HandleErrorAndExit(err)
	logger.Debug(updateFileMap)

	err = compare(updateFileMap, distributionFileMap, updateDescriptor)
	util.HandleErrorAndExit(err)
	util.PrintInfo("'" + updateName + "' validation successfully finished.")
}

func compare(updateFileMap, distributionFileMap map[string]bool, updateDescriptor *util.UpdateDescriptor) error {
	updateName := viper.GetString(constant.UPDATE_NAME)
	for filePath := range updateFileMap {
		logger.Debug(fmt.Sprintf("Searching: %s", filePath))
		_, found := distributionFileMap[filePath]
		if !found {
			logger.Debug("Added files: ", updateDescriptor.File_changes.Added_files)
			isInAddedFiles := util.IsStringIsInSlice(filePath, updateDescriptor.File_changes.Added_files)
			logger.Debug(fmt.Sprintf("isInAddedFiles: %v", isInAddedFiles))
			resourceFiles := getResourceFiles()
			logger.Debug(fmt.Sprintf("resourceFiles: %v", resourceFiles))
			fileName := strings.TrimPrefix(filePath, updateName + constant.PATH_SEPARATOR)
			logger.Debug(fmt.Sprintf("fileName: %s", fileName))
			_, foundInResources := resourceFiles[fileName]
			logger.Debug(fmt.Sprintf("found in resources: %v", foundInResources))
			//check
			if !isInAddedFiles && !foundInResources {
				return errors.New("File not found in the distribution: '" + filePath + "'. If this is a new file, add an entry to the 'added_files' sections in the '" + constant.UPDATE_DESCRIPTOR_FILE + "' file")
			} else {
				logger.Debug("'" + filePath + "' found in added files.")
			}
		}
	}
	return nil
}

//This function will read the update zip at the the given location
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
				//Check
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
				logger.Debug(fmt.Sprintf("resourceFiles: %v", resourceFiles))
				prefix := filepath.Join(updateName, constant.CARBON_HOME)
				hasPrefix := strings.HasPrefix(file.Name, prefix)
				_, foundInResources := resourceFiles[file.FileInfo().Name()]
				logger.Debug(fmt.Sprintf("foundInResources: %v", foundInResources))
				if !hasPrefix && !foundInResources {
					return nil, nil, errors.New(fmt.Sprintf("Unknown file found: '%s'.", file.Name))
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

//This function will validate the provided file. If the word 'patch' is found, a warning smessage is printed.
func validateFile(file *zip.File, fileName, fullPath, updateName string) ([]byte, error) {
	logger.Debug(fmt.Sprintf("Validating '%s' at '%s' started.", fileName, fullPath))
	parent := strings.TrimSuffix(file.Name, file.FileInfo().Name())
	if file.Name != fullPath {
		return nil, errors.New(fmt.Sprintf("'%s' found at '%s'. It should be in the '%s' directory.", fileName, parent, updateName))
	} else {
		logger.Debug(fmt.Sprintf("'%s' found at '%s'.", fileName, parent))
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
	dataString = util.ProcessString(dataString, "\n", true)

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
		for i, line := range allMatches {
			util.PrintInfo(fmt.Sprintf("Matching Line #%d - %v", i + 1, line[0]))
		}
		fmt.Println()
	}
	logger.Debug(fmt.Sprintf("Validating '%s' finished.", fileName))
	return data, nil
}

//This function reads the product distribution at the given location
func readDistributionZip(filename string) (map[string]bool, error) {
	fileMap := make(map[string]bool)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	productName := viper.GetString(constant.PRODUCT_NAME)
	logger.Debug(fmt.Sprintf("productName: %s", productName))
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
