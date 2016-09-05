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
	////Set the logger level. If the logger level is not given, set the logger level to WARN
	//if (debugLogsEnabled) {
	//	logger.SetLevel(levels.DEBUG)
	//	logger.Debug("loggers enabled")
	//} else if (traceLogsEnabled) {
	//	logger.SetLevel(levels.TRACE)
	//	logger.Debug("loggers enabled")
	//} else {
	//	logger.SetLevel(levels.WARN)
	//}
	//logger.Debug("validate command called")
	//
	////Initialize variables
	//initialize()
	//
	////Update location should be a zip file
	//logger.Debug("Update Loc: %s", updateLocation)
	//if !isAZipFile(updateLocation) {
	//	printFailureAndExit("Update file should be a zip file.")
	//}
	////Check whether the update location exists
	//updateLocationExists := isFileExists(updateLocation)
	//if updateLocationExists {
	//	logger.Debug("Update location exists.")
	//} else {
	//	printFailureAndExit("Update location does not exist. Enter a valid file location.")
	//}
	//
	//logger.Debug("Reading update zip...")
	//readUpdateZip(updateLocation, debugLogsEnabled || traceLogsEnabled)
	//logger.Debug("Update zip successfully read.")
	//logger.Debug("Entries in update zip: %s", updatedFilesMap)
	//
	//logger.Debug("Distribution Loc: " + distributionLocation)
	////Check whether the distribution is a zip or a directory
	//if isAZipFile(distributionLocation) {
	//	//Check whether the distribution zip exists
	//	zipFileExists := isFileExists(distributionLocation)
	//	if zipFileExists {
	//		logger.Debug("Distribution location exists.")
	//		readDistZip(distributionLocation, debugLogsEnabled || traceLogsEnabled)
	//	} else {
	//		printFailureAndExit("Distribution zip does not exist. Enter a valid location.")
	//	}
	//} else {
	//	//Check whether the distribution location exists
	//	distributionLocationExists := isDirectoryExists(distributionLocation)
	//	if distributionLocationExists {
	//		logger.Debug("Distribution location exists.")
	//		readDistDir(distributionLocation, debugLogsEnabled || traceLogsEnabled)
	//	} else {
	//		printFailureAndExit("Distribution location does not exist. Enter a valid location.")
	//	}
	//}
	////Validate files
	//validate()
}

////This initializes the variables
//func initialize() {
//	allResFiles = make(map[string]bool)
//	allResFiles[constant.LICENSE_FILE] = true
//	allResFiles[constant.NOT_A_CONTRIBUTION_FILE] = true
//	allResFiles[constant.README_FILE] = true
//	allResFiles[constant.UPDATE_DESCRIPTOR_FILE] = true
//	allResFiles[constant.INSTRUCTIONS_FILE] = true
//
//	updatedFilesMap = make(map[string]bool)
//	distFileMap = make(map[string]bool)
//	addedFilesMap = make(map[string]bool)
//}

//This method validates the files
func validate() {
	////Iterate through all the files in the update. All files should be in the distribution unless they are newly
	//// added files
	//for updateLoc := range updatedFilesMap {
	//	logger.Trace("Checking location: %s", updateLoc)
	//	//Check whether the distribution has a file with the same name
	//	_, found := distFileMap[updateLoc]
	//	//If there is a file
	//	if found {
	//		logger.Trace(updateLoc, "found in distFileMap")
	//	} else {
	//		//If there is no file
	//		logger.Trace("%s not found in distFileMap", updateLoc)
	//		//Check whether it is a newly added file
	//		_, found := addedFilesMap[updateLoc]
	//		//if it is a newly added file
	//		if found {
	//			logger.Trace("%s found in addedFilesMap", updateLoc)
	//		} else {
	//			//If it is not a newly added file, print an error
	//			logger.Trace("%s not found in addedFilesMap", updateLoc)
	//			logger.Trace("addedFilesMap: %s", addedFilesMap)
	//			util.PrintError(updateLoc, "not found in distribution and it is not a newly added file.")
	//			fmt.Println("If it is a new file, please add an entry in", constant.UPDATE_DESCRIPTOR_FILE, "file.")
	//			printValidationFailureMessage()
	//			os.Exit(1)
	//		}
	//	}
	//}
	//color.Set(color.FgGreen)
	//fmt.Println("\n[INFO] Validation SUCCESSFUL\n")
	//color.Unset()
}

////This function reads the files of the given update zip
//func readUpdateZip(zipLocation string, loggersEnabled bool) {
//	logger.Debug("Zip file reading started: %s", zipLocation)
//
//	updateName := strings.TrimSuffix(zipLocation, ".zip")
//	if lastIndex := strings.LastIndex(updateName, string(os.PathSeparator)); lastIndex > -1 {
//		updateName = updateName[lastIndex + 1:]
//	}
//	logger.Debug("Update name: %s", updateName)
//
//	//Check whether the update name has the required prefix
//	if !strings.HasPrefix(updateName, constant.UPDATE_NAME_PREFIX) {
//		util.PrintErrorAndExit("Update file does not have", constant.UPDATE_NAME_PREFIX, "prefix")
//	} else {
//		logger.Debug("Update file does have %s prefix", constant.UPDATE_NAME_PREFIX)
//	}
//
//	// Create a reader out of the zip archive
//	zipReader, err := zip.OpenReader(zipLocation)
//	if err != nil {
//		util.PrintErrorAndExit("Error occurred while reading zip: %s", err)
//	}
//	defer zipReader.Close()
//
//	totalFiles := len(zipReader.Reader.File)
//	logger.Trace("File count in zip: %s", totalFiles)
//
//	fileCount := 0
//	//Create a new writer to show the progress
//	writer := uilive.New()
//	//start listening for updates and render
//	writer.Start()
//
//	// Iterate through each file/dir found in the zip
//	for _, file := range zipReader.Reader.File {
//		fileCount++
//		if (!loggersEnabled) {
//			fmt.Fprintf(writer, "Reading files from update zip: (%d/%d)\n", fileCount, totalFiles)
//			time.Sleep(time.Millisecond * 2)
//		}
//		logger.Trace("Checking file: %s", file.Name)
//
//		//Ignore directories. Directories will also be listed here if the update zip is unzipped and zipped again
//		if strings.HasSuffix(file.Name, "/") {
//			continue
//		}
//
//		//Every file should be in a root folder. Check for the os.PathSeparator character to identify this
//		index := strings.Index(file.Name, "/")//string(os.PathSeparator) removed because it does not work properly in windows
//		if index == -1 {
//			util.PrintErrorAndExit("Update zip file should have a root folder called", updateName)
//		} else {
//			rootFolder := file.Name[:index]
//			logger.Trace("RootFolder: %s", rootFolder)
//			if rootFolder != updateName {
//				util.PrintErrorAndExit(file.Name, "should be in", updateName, "root directory. But it is in ", rootFolder, "directory.")
//			}
//		}
//
//		//todo: rearrange the flow
//		//Check whether the file is a resource file
//		_, found := allResFiles[file.FileInfo().Name()]
//		//If it is not a resource file
//		if !found {
//			//It should have a PATCH_NAME/carbon.home prefix
//			pathPrefix := updateName + "/" + constant.CARBON_HOME
//			if !strings.Contains(file.Name, pathPrefix) {
//				//string(os.PathSeparator) removed because it does not work properly in windows
//				util.PrintErrorAndExit("'" + file.Name + "' is not a known resource file. It should be in '" + pathPrefix + "/" + "' folder")
//			}
//			logger.Debug("Have a %s folder.", constant.CARBON_HOME)
//			//string(os.PathSeparator) removed because it does not work properly in windows
//			temp := strings.TrimPrefix(file.Name, updateName + "/" + constant.CARBON_HOME)
//			logger.Trace("Entry: %s", temp)
//			updatedFilesMap[temp] = true
//		} else {
//			//If the file is a resource file, delete the entry from allResFiles. This map is later used
//			// to track missing resource files
//			logger.Trace(file.FileInfo().Name(), "was found in resource map")
//			delete(allResFiles, file.FileInfo().Name())
//			logger.Trace(file.FileInfo().Name(), "was removed from the map")
//			//If the file is update-descriptor.yaml file, we need to read the newly added files.
//			// Otherwise there will be no match for these files and validation will be failed
//			if file.FileInfo().Name() == constant.UPDATE_DESCRIPTOR_FILE {
//				//Open the file
//				yamlFile, err := file.Open()
//				if err != nil {
//					util.PrintErrorAndExit("Error occurred while reading the", constant.UPDATE_DESCRIPTOR_FILE, "file:", err)
//				}
//				//Get the byte array
//				data, err := ioutil.ReadAll(yamlFile)
//				if err != nil {
//					util.PrintErrorAndExit("Error occurred while reading the", constant.UPDATE_DESCRIPTOR_FILE, "file:", err)
//				}
//				descriptor := update_descriptor{}
//				//Get the values
//				err = yaml.Unmarshal(data, &descriptor)
//				if err != nil {
//					util.PrintErrorAndExit("Error occurred while unmarshalling the yaml:", err)
//				}
//				logger.Debug("descriptor:", descriptor)
//				//Add all files to addedFilesMap
//				for _, addedFile := range descriptor.File_changes.Added_files {
//					addedFilesMap[addedFile] = true
//				}
//			}
//		}
//	}
//	//Stop the writer
//	writer.Stop()
//
//	//Delete instructions.txt file if it is left in the map because it is optional
//	_, found := allResFiles[constant.INSTRUCTIONS_FILE]
//	if found {
//		logger.Debug("%s was not found in the zip file.", constant.INSTRUCTIONS_FILE)
//		delete(allResFiles, constant.INSTRUCTIONS_FILE)
//		logger.Trace("Resource map: %s", allResFiles)
//		logger.Trace(updatedFilesMap)
//		color.Set(color.FgYellow)
//		fmt.Println("[INFO]", constant.INSTRUCTIONS_FILE, "was not found in the zip file.")
//		color.Unset()
//	} else {
//		logger.Debug("%s was found in the zip file.", constant.INSTRUCTIONS_FILE)
//	}
//
//	//Delete NOT_A_CONTRIBUTION.txt file if it is left in the map because it is optional
//	_, found = allResFiles[constant.NOT_A_CONTRIBUTION_FILE]
//	if found {
//		logger.Debug("%s was not found in the zip file.", constant.NOT_A_CONTRIBUTION_FILE)
//		delete(allResFiles, constant.NOT_A_CONTRIBUTION_FILE)
//		logger.Trace("Resource map: %s", allResFiles)
//		logger.Trace(updatedFilesMap)
//		color.Set(color.FgYellow)
//		fmt.Println("[INFO]", constant.NOT_A_CONTRIBUTION_FILE, "was not found in the zip file.")
//		color.Unset()
//	} else {
//		logger.Debug("%s was found in the zip file.", constant.NOT_A_CONTRIBUTION_FILE)
//	}
//
//	//At this point, the size of the allResFiles should be zero. If one or more files are not found, that means
//	// that some required files are missing
//	if (len(allResFiles) != 0) {
//		//Print the missing files
//		util.PrintErrorAndExit("Following resource file(s) were not found in the update zip: ")
//		for key := range allResFiles {
//			fmt.Println("\t", "-", key)
//		}
//	}
//
//	//Check whether all files are read
//	logger.Debug("Zip file reading finished")
//	logger.Debug("Total files read: ", fileCount)
//	if totalFiles == fileCount {
//		logger.Debug("All files read")
//	} else {
//		util.PrintErrorAndExit("All files not read from zip file")
//	}
//}
//
////This function reads the files of the given distribution zip
//func readDistZip(zipLocation string, loggersEnabled bool) {
//	logger.Debug("Zip file reading started: ", zipLocation)
//
//	//Get the distribution name
//	distName := strings.TrimSuffix(zipLocation, ".zip")
//	if lastIndex := strings.LastIndex(distName, string(os.PathSeparator)); lastIndex > -1 {
//		distName = distName[lastIndex + 1:]
//	}
//
//	// Create a reader out of the zip archive
//	zipReader, err := zip.OpenReader(zipLocation)
//	if err != nil {
//		util.PrintErrorAndExit("Error occurred while reading zip:", err)
//	}
//	defer zipReader.Close()
//
//	totalFiles := len(zipReader.Reader.File)
//	logger.Debug("File count in zip: %s", totalFiles)
//
//	fileCount := 0
//	//Create a writer to show the progress
//	writer := uilive.New()
//	//start listening for updates and render
//	writer.Start()
//
//	// Iterate through each file/dir found in
//	for _, file := range zipReader.Reader.File {
//		fileCount++
//		if (!loggersEnabled) {
//			fmt.Fprintf(writer, "Reading files from distribution zip: (%d/%d)\n", fileCount, totalFiles)
//			time.Sleep(time.Millisecond * 2)
//		}
//
//		logger.Debug("Checking file: %s", file.Name)
//
//		//Get the relative path in the zip
//		temp := strings.TrimPrefix(file.Name, distName)
//		logger.Debug("Entry: %s", temp)
//		//Add to the map
//		distFileMap[temp] = true
//
//	}
//	//Stop the writer
//	writer.Stop()
//
//	//Check whether all files are read
//	logger.Debug("Zip file reading finished")
//	logger.Debug("Total files read: %s", fileCount)
//	if totalFiles == fileCount {
//		logger.Debug("All files read.")
//	} else {
//		util.PrintErrorAndExit("All files not read from zip file.")
//	}
//}
//
//func readDistDir(distributionLocation string, loggersEnabled bool) {
//	//Create a writer to show the progress
//	writer := uilive.New()
//	//start listening for updates and render
//	writer.Start()
//	fileCount := 0
//	//Start the walk
//	err := filepath.Walk(distributionLocation, func(path string, fileInfo os.FileInfo, err error) error {
//		fileCount++;
//		if (!loggersEnabled) {
//			fmt.Fprintf(writer, "Reading files from distribution directory: %d\n", fileCount)
//			time.Sleep(time.Millisecond * 2)
//		}
//		logger.Trace("Walking: %s", path)
//		//Get the relative path
//		temp := strings.TrimPrefix(path, distributionLocation)
//		logger.Trace("Entry: %s", temp)
//		//Add to the map
//		distFileMap[temp] = true
//		return nil
//	})
//	if err != nil {
//		util.PrintErrorAndExit("Error occurred while reading the zip file: ", err)
//	}
//	logger.Debug("Total files read: %s", fileCount)
//	//stop the writer
//	writer.Stop()
//}
//
//func printValidationFailureMessage() {
//	util.PrintInRed("\nValidation FAILED\n")
//}
