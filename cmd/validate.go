// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"archive/zip"
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
	"errors"
)

var (
	validateCmdUse = "validate <update_loc> <dist_loc>"
	validateCmdShortDesc = "A brief description of your command"
	validateCmdLongDesc = dedent.Dedent(`
		A longer description that spans multiple lines and likely contains
		examples and usage of using your command.`)
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
	validateCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", false, "Enable debug logs")
	validateCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", false, "Enable trace logs")
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
		util.PrintErrorAndExit("Entered update location does not have a 'zip' extention.")
	}
	//Check 2
	exists, err := util.IsFileExists(updateFilePath)
	util.HandleError(err, "")
	if !exists {
		util.PrintErrorAndExit("Update file '" + updateFilePath + "' does not exist.")
	}

	exists, err = util.IsDistributionExists(distributionLocation)
	util.HandleError(err, "Error occurred while checking '" + distributionLocation + "'")
	if !exists {
		util.PrintErrorAndExit("Distribution does not exist at ", distributionLocation)
	}

	//Check 3
	locationInfo, err := os.Stat(updateFilePath)
	util.HandleError(err, "")
	match, err := regexp.MatchString(constant.FILENAME_REGEX, locationInfo.Name())
	if !match {
		util.PrintErrorAndExit("Update file name does not match '" + constant.FILENAME_REGEX + "' regular expression.")
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
	for filePath := range updateFileMap {
		logger.Debug("Searching:", filePath)
		_, found := distributionFileMap[filePath]
		if !found {
			isInAddedFiles := util.IsStringIsInSlice(filePath, updateDescriptor.File_changes.Added_files)
			if !isInAddedFiles {
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
				prefix := filepath.Join(updateName, constant.CARBON_HOME)
				hasPrefix := strings.HasPrefix(file.Name, prefix)
				if !hasPrefix {
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
	if file.Name != fullPath {
		parent := strings.TrimSuffix(file.Name, file.FileInfo().Name())
		return nil, errors.New("'" + fileName + "' found at '" + parent + "'. It should be in the '" + updateName + "' directory")
	}
	zippedFile, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(zippedFile)
	zippedFile.Close()
	if err != nil {
		return nil, err
	}
	dataString := string(data)
	//check
	isPatchWordFound := strings.Contains(dataString, "patch")
	if isPatchWordFound {
		util.PrintWarning("'" + fileName + "' file contains the word 'patch'. Please review and change it to 'update' if possible.")
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
		relativePath := strings.TrimPrefix(file.Name, productName + string(os.PathSeparator))
		if !file.FileInfo().IsDir() {
			fileMap[relativePath] = false
		}
	}
	return fileMap, nil
}