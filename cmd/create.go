// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"strconv"

	"github.com/ian-kent/go-log/layout"
	"github.com/ian-kent/go-log/levels"
	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
	"gopkg.in/yaml.v2"
)

var (
	createCmdUse = "create <update_dir> <dist_loc>"
	createCmdShortDesc = "A brief description of your command"
	createCmdLongDesc = dedent.Dedent(`
		A longer description that spans multiple lines and likely contains
		examples and usage of using your command.`)
)

//createCmd represents the create command
var createCmd = &cobra.Command{
	Use: createCmdUse,
	Short: createCmdShortDesc,
	Long: createCmdLongDesc,
	Run: initializeCreateCommand,
}

var isValidateSelected bool

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", false, "Enable debug logs")
	createCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", false, "Enable trace logs")
	createCmd.Flags().BoolVarP(&isValidateSelected, "validate", "v", false, "Validate the content of the created update zip")
}

func initializeCreateCommand(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		util.PrintErrorAndExit("Invalid number of argumants. Run 'wum-uc create --help' to view help.")
	}
	createUpdate(args[0], args[1])
}

//main execution path
func createUpdate(updateDirectoryPath, distributionPath string) {

	//set debug level
	setLogLevel()
	logger.Debug("<create> command called")

	//Flow - First check whether the given locations exists and required files exists. Then start processing.
	//If one step fails, print error message and exit.

	//1) Check whether the given update directory exists
	exists, err := util.IsDirectoryExists(updateDirectoryPath)
	//todo: look for best practice
	util.HandleError(err, "Error occurred while reading the update directory")
	if !exists {
		util.PrintErrorAndExit("Update directory '" + updateDirectoryPath + "' does not exist.")
	}
	updateRoot := strings.TrimSuffix(updateDirectoryPath, "/")
	updateRoot = strings.TrimSuffix(updateRoot, "\\")
	logger.Debug("updateRoot: %s", updateRoot)
	viper.Set(constant.UPDATE_ROOT, updateRoot)

	//2) Check whether the update-descriptor.yaml file exists
	//Construct the update-descriptor.yaml file location
	updateDescriptorPath := path.Join(updateDirectoryPath, constant.UPDATE_DESCRIPTOR_FILE)
	exists, err = util.IsFileExists(updateDescriptorPath)
	util.HandleError(err, "")
	if !exists {
		util.PrintError("'" + constant.UPDATE_DESCRIPTOR_FILE + "' not found at '" + updateDirectoryPath + "'.")
		util.PrintWhatsNext("Run 'wum-uc init " + updateDirectoryPath + "' to generate the '" + constant.UPDATE_DESCRIPTOR_FILE + "' template in the update location.")
		os.Exit(1)
	}
	logger.Debug("Descriptor Exists. Location %s", updateDescriptorPath)

	//3) Check whether the given distribution exists
	exists, err = util.IsDistributionExists(distributionPath)
	util.HandleError(err, "Distribution path '" + distributionPath + "' does not exist")
	if !exists {
		util.PrintErrorAndExit("Distribution does not exist at ", updateDirectoryPath)
	}
	//4) Read update-descriptor.yaml and set the update name which will be used when creating the update zip file.
	//This is used to read the update-descriptor.yaml file
	updateDescriptor, err := util.LoadUpdateDescriptor(constant.UPDATE_DESCRIPTOR_FILE, updateDirectoryPath)
	util.HandleError(err, "Error occurred when reading '" + constant.UPDATE_DESCRIPTOR_FILE + "' file")

	//Validate the file format
	err = util.ValidateUpdateDescriptor(updateDescriptor)
	util.HandleError(err, "'" + constant.UPDATE_DESCRIPTOR_FILE + "' format is not correct")

	//set the update name
	updateName := util.GetUpdateName(updateDescriptor, constant.UPDATE_NAME_PREFIX)
	viper.Set(constant.UPDATE_NAME, updateName)

	//5) Traverse and read the update
	ignoredFiles := getIgnoredFilesInUpdate()
	updateLocationInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}
	err = readDirectoryStructure(updateDirectoryPath, &updateLocationInfo, ignoredFiles, true)
	util.HandleError(err, "")
	logger.Debug("updateLocationInfo:", updateLocationInfo)

	//6) Traverse and read distribution
	distributionLocationInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}

	if strings.HasSuffix(distributionPath, ".zip") {
		distributionRoot := GetDistributionRootDirectory(distributionPath)
		logger.Debug("distributionRoot: %s", distributionRoot)
		viper.Set(constant.DISTRIBUTION_ROOT, distributionRoot)
		unzipDirectory := util.GetParentDirectory(distributionPath)
		logger.Debug("unzipDirectory: %s", unzipDirectory)

		cleanupChannel := util.HandleInterrupts(func() {
			logger.Debug("Cleaning up distributionRoot directory")
			util.CleanUpDirectory(distributionRoot)
		})

		util.PrintInfo("Extracting zip file. Please wait...")
		err := archiver.Unzip(distributionPath, unzipDirectory)
		util.HandleError(err, "")
		util.PrintInfo("Extracting successfully finished.")

		signal.Stop(cleanupChannel)

		util.PrintInfo("Reading files started...")
		err = readDirectoryStructure(distributionRoot, &distributionLocationInfo, nil, false)
		util.HandleError(err, "")
		util.PrintInfo("Reading files successfully finished.")

		//Delete the extracted distribution directory after function is finished
		defer util.CleanUpDirectory(strings.TrimSuffix(distributionPath, ".zip"))
	} else {
		distributionRoot := strings.TrimSuffix(distributionPath, "/")
		distributionRoot = strings.TrimSuffix(distributionRoot, "\\")
		logger.Debug("distributionRoot: %s", distributionRoot)
		viper.Set(constant.DISTRIBUTION_ROOT, distributionRoot)

		err = readDirectoryStructure(distributionPath, &distributionLocationInfo, nil, false)
		util.HandleError(err, "")
	}
	logger.Debug("distributionLocationInfo:", distributionLocationInfo)

	//7) Find matches
	diff, err := getDiff(&updateLocationInfo, &distributionLocationInfo, true)
	util.HandleError(err, "Error occurred while getting the diff.")
	logger.Debug("diff: ", diff)

	//8) Copy files to the temp
	err = populateZipDirectoryStructure(diff, updateDescriptor)
	util.HandleError(err, "Error occurred while creating the folder structure.")

	//9) Copy resource files (update-descriptor.yaml, etc)
	resourceFiles := getResourceFiles()
	err = copyResourceFiles(resourceFiles)
	util.HandleError(err, &util.CustomError{What: "Error occurred while copying resource files."})

	//Save the update-descriptor with the updated, newly added files to the temp directory
	data, err := marshalUpdateDescriptor(updateDescriptor)
	util.HandleError(err)

	err = saveUpdateDescriptor(constant.UPDATE_DESCRIPTOR_FILE, data)
	util.HandleError(err)

	cleanupChannel := util.HandleInterrupts(func() {
		logger.Debug("Cleaning up temp directory")
		util.CleanUpDirectory(constant.TEMP_DIR)
	})

	updateZipName := updateName + ".zip"
	//10) Create the update zip file
	//todo: what should be the destination directory for the zip file? current working directory?
	err = archiver.Zip(updateZipName, []string{filepath.Join(constant.TEMP_DIR, updateName)})
	util.HandleError(err)

	signal.Stop(cleanupChannel)

	//Remove the temp directories
	util.CleanUpDirectory(constant.TEMP_DIR)

	util.PrintInfo("'" + updateZipName + "' successfully created.")

	if isValidateSelected {
		//fmt.Println("\n\nValidating '" + updateZipName + "'\n")
		startValidation(updateZipName, distributionPath)
	} else {
		util.PrintWhatsNext("Validate the update zip after any manual modification by running 'wum-uc validate " + updateName + ".zip " + distributionPath + "'")
	}
}

//This will return a map of files which would be ignored when reading the update directory
func getIgnoredFilesInUpdate() map[string]bool {
	return map[string]bool{
		constant.UPDATE_DESCRIPTOR_FILE: true,
		constant.LICENSE_FILE: true,
		constant.README_FILE: true,
		constant.NOT_A_CONTRIBUTION_FILE: true,
		constant.INSTRUCTIONS_FILE: true,
	}
}

//This will return a map of files which would be copied to the temp directory before creating the update zip. Key is the
// file name and value is whether the file is mandatory or not.
func getResourceFiles() map[string]bool {
	return map[string]bool{
		constant.LICENSE_FILE: true,
		constant.NOT_A_CONTRIBUTION_FILE: false,
		constant.INSTRUCTIONS_FILE: false,
	}
}

func marshalUpdateDescriptor(updateDescriptor *util.UpdateDescriptor) ([]byte, error) {
	data, err := yaml.Marshal(&updateDescriptor)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func saveUpdateDescriptor(updateDescriptorFilename string, data []byte) error {
	updateName := viper.GetString(constant.UPDATE_NAME)
	destination := filepath.Join(constant.TEMP_DIR, updateName, updateDescriptorFilename)
	// Open a new file for writing only
	file, err := os.OpenFile(
		destination,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0600,
	)
	defer file.Close()
	if err != nil {
		return err
	}

	updatedData := strings.Replace(string(data), "\"", "", 2)
	modifiedData := []byte(updatedData)
	// Write bytes to file
	_, err = file.Write(modifiedData)
	if err != nil {
		return err
	}
	return nil
}

func copyResourceFiles(resourceFilesMap map[string]bool) error {
	//Create the directories if they are not available
	updateName := viper.GetString(constant.UPDATE_NAME)
	destination := filepath.Join(constant.TEMP_DIR, updateName, constant.CARBON_HOME)
	util.CreateDirectory(destination)
	for filename, isMandatory := range resourceFilesMap {
		updateRoot := viper.GetString(constant.UPDATE_ROOT)
		updateName := viper.GetString(constant.UPDATE_NAME)
		source := filepath.Join(updateRoot, filename)
		destination := filepath.Join(constant.TEMP_DIR, updateName, filename)
		err := util.CopyFile(source, destination)
		if err != nil {
			if isMandatory {
				return err
			} else {
				util.PrintWarning("'" + filename + "' not found.")
			}
		}
	}
	return nil
}

//This will return the diff of the given FileLocationInfo structs
func getDiff(updateLocationMap, distributionLocationMap *FileLocationInfo, inspectRootOnlyInUpdate bool) (*Diff, error) {
	diff := Diff{
		files: make(map[string]LocationData),
	}
	distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
	logger.Debug("updateLocationMap:", updateLocationMap)
	for filename, updateFileLocationInfo := range updateLocationMap.nameLocationInfoMap {
		logger.Trace("[UPDATE FILE INFO]:", filename, ":", updateFileLocationInfo)
		//todo: add inspectRootOnly value support to get complete diff of two directories
		updateFilePath, updateFileInfo, err := getUpdateFilePathAndInfo(&updateFileLocationInfo)
		util.HandleError(err, "Error occurred while getting location of a file in update directory.")
		logger.Trace("[UPDATE FILE INFO] updateFilePath:", updateFilePath)
		logger.Trace("[UPDATE FILE INFO] updateFileInfo:", updateFileInfo)
		distributionLocationInfo, foundMatchInDistribution := distributionLocationMap.nameLocationInfoMap[filename]
		locationData := LocationData{
			locationsInUpdate:make(map[string]bool),
			locationsInDistribution:make(map[string]bool),
		}
		if foundMatchInDistribution {
			logger.Debug("[MATCH] Match found in distribution: ", distributionLocationInfo)
			//Add
			locationData.locationsInUpdate[updateFilePath] = updateFileInfo.isDir
			for distributionFilepath, info := range distributionLocationInfo.filepathInfoMap {
				logger.Trace("[DIST FILE INFO] filepath:", distributionFilepath, ",Info:", info)
				if !updateFileInfo.isDir && !info.isDir && updateFileInfo.md5 == info.md5 {
					message := filename + " found in both update, distribution locations. But have the same md5 hash(" + info.md5 + ")" +
						"\n\tLocation in update: " + updateFilePath + filename +
						"\n\tLocation in dist  : CARBON_HOME" + strings.TrimPrefix(distributionFilepath, distributionRoot) + filename +
						"\nIt is possible that the old file was copied to the update location instead of the new file."
					return nil, &util.CustomError{What: message }
				} else if updateFileInfo.isDir != info.isDir {
					//Has same type, but different types. Ignore these matches
					continue
				} else {
					//Add
					locationData.locationsInDistribution[distributionFilepath] = info.isDir
				}
			}
			logger.Trace("[LOCATION DATA] locationData:", locationData)
		} else {
			logger.Debug("[404] No match found in distribution.")
			//Add only the location in the update directory. This will be considered as a new file. Later we can check the locations in the distribution to identify new files.
			locationData.locationsInUpdate[updateFilePath] = updateFileInfo.isDir
		}
		diff.Add(filename, locationData)
	}
	return &diff, nil
}

//This will return the values of the given map only if the map contains single key, value pair
func getUpdateFilePathAndInfo(updateFileLocationInfo *LocationInfo) (string, *Info, error) {
	//Check for duplicate filename. A File and A Directory in the root level of the update directory might have same name(it is highly unlikely). But this is not possible in Ubuntu. Need to check on other OSs
	if len(updateFileLocationInfo.filepathInfoMap) > 1 {
		logger.Debug(updateFileLocationInfo.filepathInfoMap)
		return "", nil, &util.CustomError{What: "Duplicate files found in the update directory. Possible reason for this error is that there are a file and a directory with the same name."}
	}
	var updateFilepath string
	var locationInfo Info
	for updateFilepath, locationInfo = range updateFileLocationInfo.filepathInfoMap {
		//do nothing
	}
	return updateFilepath, &locationInfo, nil
}

//This will start populating the directory structure of the update zip and copy files
func populateZipDirectoryStructure(diff *Diff, updateDescriptor *util.UpdateDescriptor) error {
	for filename, locationData := range diff.files {
		logger.Debug("[CREATE STRUCTURE] filename:", filename)
		logger.Debug("[CREATE STRUCTURE] locationData:", locationData)
		switch len(locationData.locationsInDistribution) {
		case 0:
			err := handleNoMatch(filename, &locationData, updateDescriptor)
			util.HandleError(err)
		case 1:
			err := handleSingleMatch(filename, &locationData, updateDescriptor)
			util.HandleError(err)
		default:
			err := handleMultipleMatches(filename, &locationData, updateDescriptor)
			util.HandleError(err)
		}
	}
	return nil
}

//This function will handle the copy process if no match is found in the distribution
func handleNoMatch(filename string, locationData *LocationData, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug("[NO MATCH]", filename)
	util.PrintInBold("'" + filename + "' not found in distribution.")
	for {
		util.PrintInBold("Do you want to add it as a new file? [(Y)es/(N)o]: ")
		preference, err := util.GetUserInput()
		util.HandleError(err, "Error occurred while getting input from the user.")
		if util.IsYes(preference) {
			err = handleNewFile(filename, locationData, updateDescriptor)
			util.HandleError(err)
			//If no error, return nil
			return nil
		} else if util.IsNo(preference) {
			util.PrintWarning("Skipping copying", filename)
			return nil
		} else {
			util.PrintError("Invalid preference. Enter Y for Yes or N for No.")
		}
	}
	return nil
}

//This function will handle the copy process if only one match is found in the distribution
func handleSingleMatch(filename string, locationData *LocationData, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug("[SINGLE MATCH]", filename)
	locationInUpdate, isDir, err := getSingleLocationFromMap(locationData.locationsInUpdate)
	util.HandleError(err)
	logger.Debug("[SINGLE MATCH] Location in Update:", locationInUpdate)

	locationInDistribution, _, err := getSingleLocationFromMap(locationData.locationsInDistribution)
	util.HandleError(err)
	locationInDistribution = strings.TrimPrefix(locationInDistribution, viper.GetString(constant.DISTRIBUTION_ROOT))
	logger.Debug("[SINGLE MATCH] Matching location in the Distribution:", locationInDistribution)

	if isDir {
		distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
		updateUpdateDescriptor(filepath.Join(locationInUpdate, filename), filepath.Join(distributionRoot, locationInDistribution, filename), updateDescriptor)
	} else {
		modifiedFile := strings.TrimPrefix(filepath.Join(locationInDistribution, filename), constant.PATH_SEPARATOR)
		updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, modifiedFile)
	}
	err = copyFile(filename, isDir, locationInUpdate, locationInDistribution)
	util.HandleError(err, "Error occurred while copying the '" + filename + "' ; From " + locationInUpdate + " ; To: " + locationInDistribution)
	return nil
}

//This function will handle the copy process if multiple matches are found in the distribution
func handleMultipleMatches(filename string, locationData *LocationData, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug("[MULTIPLE MATCHES]", filename)
	util.PrintInBold("Multiple matches for '" + filename + "' found.\n")
	locationTable, indexMap := generateLocationTable(filename, locationData.locationsInDistribution)
	locationTable.Render()
	logger.Debug("indexMap:", indexMap)
	var selectedIndices []string
	for {
		util.PrintInBold("Enter preference(s)[Multiple selections separated by commas]: ")
		preferences, err := util.GetUserInput()
		util.HandleError(err)
		logger.Debug("preferences: %s", preferences)
		//Remove the new line at the end
		preferences = strings.TrimSpace(preferences)
		//Split the indices
		selectedIndices = strings.Split(preferences, ",");
		//Sort the locations
		sort.Strings(selectedIndices)
		logger.Debug("sorted: %s", preferences)

		length := len(indexMap)
		isValid, err := util.IsUserPreferencesValid(selectedIndices, length)
		if err != nil {
			util.PrintError("Invalid preferences. Please select indices where 1 <= index <= " + strconv.Itoa(length))
			continue
		}

		if !isValid {
			util.PrintError("Invalid preferences. Please select indices where 1 <= index <= " + strconv.Itoa(length))
		} else {
			logger.Debug("Entered preferences are valid.")
			break
		}
	}
	for _, selectedIndex := range selectedIndices {
		pathInDistribution := indexMap[selectedIndex]
		logger.Debug("[MULTIPLE MATCHES] Selected path:", selectedIndex, ";", pathInDistribution)

		locationInUpdate, isDir, err := getSingleLocationFromMap(locationData.locationsInUpdate)
		util.HandleError(err)
		logger.Debug("[SINGLE MATCH] Location in Update:", locationInUpdate)
		relativeLocationInDistribution := strings.TrimPrefix(pathInDistribution, viper.GetString(constant.DISTRIBUTION_ROOT))

		//todo: save the preferences to generate the final summary map
		if isDir {
			distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
			updateUpdateDescriptor(filepath.Join(locationInUpdate, filename), filepath.Join(distributionRoot, relativeLocationInDistribution, filename), updateDescriptor)
		} else {
			modifiedFile := strings.TrimPrefix(filepath.Join(relativeLocationInDistribution, filename), constant.PATH_SEPARATOR)
			updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, modifiedFile)
		}
		err = copyFile(filename, isDir, locationInUpdate, relativeLocationInDistribution)
		util.HandleError(err)
	}
	return nil
}

//This will generate the location table and the index map which will be used to get user preference
func generateLocationTable(filename string, locationsInDistribution map[string]bool) (*tablewriter.Table, map[string]string) {
	locationTable := tablewriter.NewWriter(os.Stdout)
	locationTable.SetAlignment(tablewriter.ALIGN_LEFT)
	locationTable.SetHeader([]string{"Index", "Matching Location"})
	distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
	index := 1
	indexMap := make(map[string]string)
	for distributionFilepath, isDir := range locationsInDistribution {
		logger.Debug("[TABLE] filepath:", distributionFilepath, "; isDir:", isDir)
		indexMap[strconv.Itoa(index)] = distributionFilepath
		relativePath := "CARBON_HOME" + strings.TrimPrefix(distributionFilepath, distributionRoot)
		locationTable.Append([]string{strconv.Itoa(index), path.Join(relativePath, filename)})
		index++
	}
	return locationTable, indexMap
}

//This function will handle the copy process if the user wants to add a file as a new file
func handleNewFile(filename string, locationData *LocationData, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug("[HANDLE NEW] Update:", filename)
	locationInUpdate, isDir, err := getSingleLocationFromMap(locationData.locationsInUpdate)
	util.HandleError(err)
	logger.Debug("[HANDLE NEW] Update:", locationInUpdate, ";", isDir)
	readDestinationLoop:
	for {
		util.PrintInBold("Enter destination directory relative to CARBON_HOME: ")
		relativeLocationInDistribution, err := util.GetUserInput()
		util.HandleError(err, "Error occurred while getting input from the user.")
		logger.Debug("relativePath:", relativeLocationInDistribution)

		fullPath := filepath.Join(viper.GetString(constant.DISTRIBUTION_ROOT), relativeLocationInDistribution)
		logger.Debug("fullPath:", fullPath)
		//Ignore error because we are only checking whether the given path exists or not
		exists, _ := util.IsDirectoryExists(fullPath)
		if exists {
			err = copyFile(filename, isDir, locationInUpdate, relativeLocationInDistribution)
			util.HandleError(err)
			break
		} else {
			util.PrintInBold("Entered relative path does not exist in the distribution. ")
			for {
				util.PrintInBold("Copy anyway? [(Y)es/(N)o/(R)e-enter]: ")
				preference, err := util.GetUserInput()
				util.HandleError(err, "Error occurred while getting input from the user.")

				if util.IsYes(preference) {
					//todo: save the selected location to generate the final summary map
					if isDir {
						distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
						updateUpdateDescriptor(filepath.Join(locationInUpdate, filename), filepath.Join(distributionRoot, relativeLocationInDistribution, filename), updateDescriptor)
					} else {
						newFile := strings.TrimPrefix(filepath.Join(relativeLocationInDistribution, filename), constant.PATH_SEPARATOR)
						updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, newFile)
					}
					err = copyFile(filename, isDir, locationInUpdate, relativeLocationInDistribution)
					util.HandleError(err)
					break readDestinationLoop
				} else if util.IsNo(preference) {
					util.PrintWarning("Skipping copying", filename)
					return nil
				} else if util.IsReenter(preference) {
					break
				} else {
					util.PrintError("Invalid preference. Enter Y for Yes or N for No or R for Re-enter.")
				}
			}
		}
	}
	return nil
}

func updateUpdateDescriptor(source, destination string, updateDescriptor *util.UpdateDescriptor) {
	sourceInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}
	logger.Debug("\nsource:", source)
	err := readDirectoryStructure(source, &sourceInfo, nil, true)
	util.HandleError(err, "")
	logger.Trace("\nsourceInfo:", sourceInfo)

	destinationInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}

	logger.Debug("\ndestination:", destination)
	err = readDirectoryStructure(destination, &destinationInfo, nil, true)
	util.HandleError(err, "")
	logger.Trace("\ndestinationInfo:", destinationInfo)

	diff, err := getDiff(&sourceInfo, &destinationInfo, true)
	util.HandleError(err, "Error occurred while getting the diff.")

	logger.Debug("\ndiff:", diff)
	for filename, locationData := range diff.files {
		switch len(locationData.locationsInDistribution) {
		case 0:
			// new file
			distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
			relativePathInDistribution := strings.TrimPrefix(filepath.Join(destination, filename), distributionRoot)
			relativePathInDistribution = strings.TrimPrefix(relativePathInDistribution, string(os.PathSeparator))
			fmt.Println("XX NEW: ", relativePathInDistribution)
			updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, relativePathInDistribution)
		default:
			//modified file
			locationInDistribution, isDir := getLocationFromMap(locationData.locationsInDistribution)
			if !isDir {
				distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
				relativePathInDistribution := strings.TrimPrefix(filepath.Join(locationInDistribution, filename), distributionRoot)
				relativePathInDistribution = strings.TrimPrefix(relativePathInDistribution, string(os.PathSeparator))
				fmt.Println("XX Modified: ", relativePathInDistribution)
				updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, relativePathInDistribution)
			} else {
				newLocationData := diff.files[filename]
				newSource, sourceIsDir, err := getSingleLocationFromMap(newLocationData.locationsInUpdate)
				util.HandleError(err)
				newDestination, destinationIsDir, err := getSingleLocationFromMap(newLocationData.locationsInDistribution)
				util.HandleError(err)
				if ( sourceIsDir && destinationIsDir) {
					updateUpdateDescriptor(filepath.Join(newSource, filename), filepath.Join(newDestination, filename), updateDescriptor)
				}
			}
		}
	}
}

//This function will copy the file/directory from update to temp location
func copyFile(filename string, isDir bool, locationInUpdate, relativeLocationInTemp string) error {
	//todo list
	//check type
	//if file, check path matches /plugin path
	//else dir, check files in the directory for matches

	//get the relative path in the distribution and join to the temp directory to get the destination directory
	logger.Debug("[FINAL][COPY] Name:", filename, "; IsDir:", isDir, "; From:", locationInUpdate, "; To:", relativeLocationInTemp)
	updateName := viper.GetString(constant.UPDATE_NAME)
	source := filepath.Join(locationInUpdate, filename)
	destination := filepath.Join(constant.TEMP_DIR, updateName, constant.CARBON_HOME, relativeLocationInTemp)
	util.CreateDirectory(destination)
	if !isDir {
		isABundle := IsABundle(filename, relativeLocationInTemp)

		if isABundle {
			newName := ConstructBundleName(filename)
			logger.Debug("[FINAL][COPY][TEMP] Name:", filename, "; From:", source, "; To:", filepath.Join(destination, newName))
			err := util.CopyFile(source, filepath.Join(destination, newName))
			util.HandleError(err)
		} else {
			logger.Debug("[FINAL][COPY][TEMP2] Name:", filename, "; From:", source, "; To:", filepath.Join(destination, filename))
			err := util.CopyFile(source, filepath.Join(destination, filename))
			util.HandleError(err)
		}
	} else {
		logger.Debug("[FINAL][COPY][TEMP3] Name:", filename, "; From:", source, "; To:", filepath.Join(destination, filename))
		err := util.CopyDir(source, filepath.Join(destination, filename))
		util.HandleError(err)
	}
	return nil
}

//This function detects whether a jar should be a bundle or not according to its path
func IsABundle(filename, location string) bool {
	//todo
	if location == constant.PLUGINS_DIRECTORY {
		return true
	}
	return false
}

//This constructs the bundle name of a normal jar
func ConstructBundleName(filename string) string {
	//todo
	return filename
}

//This will return the values of the map only if the map contain single key, value pair
func getSingleLocationFromMap(locationMap map[string]bool) (string, bool, error) {
	//Can only have single entry
	if len(locationMap) > 1 {
		return "", false, &util.CustomError{What: "Multiple matches found."}
	}
	var location string
	var isDir bool
	for location, isDir = range locationMap {
		//do nothing
	}
	return location, isDir, nil
}

func getLocationFromMap(locationMap map[string]bool) (string, bool) {
	var location string
	var isDir bool
	for location, isDir = range locationMap {
		//do nothing
	}
	return location, isDir
}

//This function will set the log level
func setLogLevel() {
	//Setting default time format. This will be used in loggers. Otherwise complete date and time will be printed
	layout.DefaultTimeLayout = "15:04:05"
	//Setting new STDOUT layout to logger
	logger.Appender().SetLayout(layout.Pattern("[%d] [%p] %m"))
	//Set the log level. If the log level is not given, set the log level to default level
	if isDebugLogsEnabled {
		logger.SetLevel(levels.DEBUG)
		logger.Debug("Debug logs enabled")
	} else if isTraceLogsEnabled {
		logger.SetLevel(levels.TRACE)
		logger.Trace("Trace logs enabled")
	} else {
		logger.SetLevel(constant.DEFAULT_LOG_LEVEL)
	}
	logger.Debug("[LOG LEVEL]", logger.Level())
}

//This function will return the directory which would be the root directory of the distribution after extracting the zip file
func GetDistributionRootDirectory(distributionZipPath string) string {
	lastIndex := strings.LastIndex(distributionZipPath, ".")
	return distributionZipPath[:lastIndex]
}

//This function will iterate and read the file/folder structure of a given root directory
func readDirectoryStructure(root string, locationMap *FileLocationInfo, ignoredFiles map[string]bool, rootLevelOnly bool) error {
	if rootLevelOnly {
		files, _ := ioutil.ReadDir(root)
		for _, file := range files {
			if ignoredFiles != nil {

				_, found := ignoredFiles[file.Name()]
				if found {
					continue
				}
			}
			logger.Debug("filename: ", file.Name())
			absolutePath := filepath.Join(root, file.Name())
			if !file.IsDir() {
				logger.Debug("[MD5] Calculating MD5")
				//If it is a file, calculate md5 sum
				md5, err := util.GetMD5(absolutePath)
				if err != nil {
					return err
				}
				logger.Debug(absolutePath + " : " + file.Name() + ": " + md5)
				locationMap.Add(file.Name(), root, file.IsDir(), md5)
			} else {
				logger.Debug(absolutePath + " : " + file.Name())
				locationMap.Add(file.Name(), root, file.IsDir(), "")
			}
		}
		return nil
	} else {
		//Remove the / or \ at the end of the path if it exists/ Otherwise the root directory wont be ignored
		//root = strings.TrimSuffix(root, string(os.PathSeparator))
		return filepath.Walk(root, func(absolutePath string, fileInfo os.FileInfo, err error) error {
			logger.Debug("[WALK] %s", absolutePath, ";", fileInfo.IsDir())
			if err != nil {
				return err
			}
			//Ignore root directory
			if root == absolutePath {
				return nil
			}
			//check current file in ignored files map. This is useful to ignore update-descriptor.yaml, etc in update directory
			if ignoredFiles != nil {
				_, found := ignoredFiles[fileInfo.Name()]
				if found {
					return nil
				}
			}
			//get the parent directory path
			parentDirectory := strings.TrimSuffix(absolutePath, fileInfo.Name())
			//Check for file / directory
			if !fileInfo.IsDir() {
				logger.Debug("[MD5] Calculating MD5")
				//If it is a file, calculate md5 sum
				md5, err := util.GetMD5(absolutePath)
				if err != nil {
					return err
				}
				logger.Debug(absolutePath + " : " + fileInfo.Name() + ": " + md5)
				locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), md5)
				//
				//fullPath := filepath.Join(root, constant.PLUGINS_DIRECTORY, fileInfo.Name())
				//logger.Trace("[COMPARE] " + fullPath + " ; " + absolutePath)
				//if (fullPath == absolutePath) && util.HasJarExtension(absolutePath) {
				//	logger.Debug("[PLUGIN] FilePath:", absolutePath)
				//	newFileName := strings.Replace(fileInfo.Name(), "_", "-", 1)
				//	logger.Debug("[PLUGIN] New Name:", newFileName)
				//	if index := strings.Index(newFileName, "_"); index != -1 {
				//		return &util.CustomError{What: fileInfo.Name() + " is in " + constant.PLUGINS_DIRECTORY + ". But it has multiple _ in its name. Only one _ is expected." }
				//	}
				//	locationMap.Add(newFileName, parentDirectory, fileInfo.IsDir(), md5)
				//}
			} else {
				logger.Debug(absolutePath + " : " + fileInfo.Name())
				locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), "")
			}
			return nil
		})
	}
}
