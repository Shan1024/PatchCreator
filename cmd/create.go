package cmd

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ian-kent/go-log/log"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/layout"
	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/shan1024/wum-uc/constant"
	"github.com/shan1024/wum-uc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <update_dir> <dist_loc>",
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
		createUpdate(args[0], args[1])
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
	createCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", false, "Enable debug logs")
	createCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", false, "Enable trace logs")
}

func createUpdate(updateDirectoryPath, distributionPath string) {

	//set debug level
	SetLogLevel()
	logger.Debug("create command called")

	//Flow - First check whether the given locations exists and required files exists. Then start processing.
	//If one step fails, print error message and exit.

	//1) Check whether the given update directory exists
	exists, err := util.IsDirectoryExists(updateDirectoryPath)
	util.HandleError(err, "Update directory does not exist.")
	if !exists {
		util.PrintErrorAndExit("Update Directory does not exist at %s.", updateDirectoryPath)
	}
	updateRoot := strings.TrimSuffix(updateDirectoryPath, "/")
	updateRoot = strings.TrimSuffix(updateRoot, "\\")
	log.Debug("updateRoot: %s", updateRoot)
	viper.Set(constant.UPDATE_ROOT, updateRoot)

	//todo: break into two parts. Reading and verifying the structure of yaml file
	//2) Check whether the update-descriptor.yaml file exists
	//Construct the update-descriptor.yaml file location
	updateDescriptorPath := path.Join(updateDirectoryPath, constant.UPDATE_DESCRIPTOR_FILE)
	exists, err = util.IsFileExists(updateDescriptorPath)
	util.HandleError(err, "")
	if !exists {
		util.PrintErrorAndExit("%s not found at %s.", constant.UPDATE_DESCRIPTOR_FILE, updateDescriptorPath)
	}
	logger.Debug("Descriptor Exists. Location %s", updateDescriptorPath)

	//3) Check whether the given distribution exists
	exists, err = isDistributionExists(distributionPath)
	util.HandleError(err, "")
	if !exists {
		util.PrintErrorAndExit("Distribution does not exist at %s.", updateDirectoryPath)
	}

	//4) Read update-descriptor.yaml and set the update name which will be used when creating the update zip file.
	//This is used to read the update-descriptor.yaml file
	updateDescriptor := update_descriptor{}
	err = readDescriptor(&updateDescriptor, updateDirectoryPath)
	util.HandleError(err, "")
	//set the update name
	setUpdateName(&updateDescriptor)

	//5) Traverse and read the update
	ignoredFiles := getIgnoredFilesInUpdate()
	updateLocationInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}
	err = readDirectoryStructure(updateDirectoryPath, &updateLocationInfo, ignoredFiles)
	util.HandleError(err, "")

	logger.Trace("updateLocationInfo:", updateLocationInfo)

	//6) Traverse and read distribution
	distributionLocationInfo := FileLocationInfo{
		nameLocationInfoMap: make(map[string]LocationInfo),
	}

	if util.HasZipExtension(distributionPath) {

		unzipDirectory := util.GetParentDirectory(distributionPath)
		logger.Debug("unzipDirectory: %s", unzipDirectory)

		util.PrintInfo("Extracting zip file. Please wait...")
		err := archiver.Unzip(distributionPath, unzipDirectory)
		util.HandleError(err, "")

		util.PrintInfo("Extracting zip file finished successfully.")

		distributionRoot := getDistributionRootDirectory(distributionPath)
		log.Debug("distributionRoot: %s", distributionRoot)
		viper.Set(constant.DISTRIBUTION_ROOT, distributionRoot)

		err = readDirectoryStructure(distributionRoot, &distributionLocationInfo, nil)
		util.HandleError(err, "")

		//Delete the extracted distribution directory after function is finished
		//defer os.RemoveAll(strings.TrimSuffix(distributionPath, ".zip")) //todo: add
	} else {
		distributionRoot := strings.TrimSuffix(distributionPath, "/")
		distributionRoot = strings.TrimSuffix(distributionRoot, "\\")
		log.Debug("distributionRoot: %s", distributionRoot)
		viper.Set(constant.DISTRIBUTION_ROOT, distributionRoot)

		err = readDirectoryStructure(distributionPath, &distributionLocationInfo, nil)
		util.HandleError(err, "")
	}

	logger.Trace("distributionLocationInfo:", distributionLocationInfo)

	diff, err := getDiff(&updateLocationInfo, &distributionLocationInfo)
	util.HandleError(err, "Error occurred while getting the diff.")

	logger.Trace("diff: ", diff)

	err = createFolderStructure(diff)
	util.HandleError(err, "Error occurred while creating the folder structure.")



	////7) Find matches
	//if hasZipExtension(distributionPath) {
	//	logger.Debug("Finding matches")
	//	findMatches(&updateEntriesMap, &distEntriesMap, &updateDescriptor, updateDirectory, strings.TrimSuffix(distributionPath, ".zip"))
	//	logger.Debug("Finding matches finished")
	//} else {
	//	logger.Debug("Finding matches")
	//	findMatches(&updateEntriesMap, &distEntriesMap, &updateDescriptor, updateDirectory, distributionPath)
	//	logger.Debug("Finding matches finished")
	//}
	//
	////Copy resource files to the temp location
	//logger.Debug("Copying resource files")
	//copyResourceFiles(updateDirectory)
	//logger.Debug("Copying resource files finished")
	//
	////Update the update-descriptor with the newly added files
	//prepareUpdateDescriptorForSaving(&updateDescriptor)
	//
	////Create the update zip file
	//logger.Debug("Creating zip file")
	//createUpdateZip(_UPDATE_NAME)
	//logger.Debug("Creating zip file finished")

	//Remove the temp directory
	err = util.DeleteDirectory(constant.TEMP_DIR)
	util.HandleError(err, "")
}

func getIgnoredFilesInUpdate() map[string]bool {
	return map[string]bool{
		constant.UPDATE_DESCRIPTOR_FILE: true,
		constant.LICENSE_FILE: true,
		constant.README_FILE: true,
		constant.NOT_A_CONTRIBUTION_FILE: true,
		constant.INSTRUCTIONS_FILE: true,
	}
}

func getDiff(updateLocationMap, distributionLocationMap *FileLocationInfo) (*Diff, error) {
	diff := Diff{
		files: make(map[string]LocationData),
	}

	distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)

	for filename, updateFileLocationInfo := range updateLocationMap.nameLocationInfoMap {
		logger.Debug("[UPDATE FILE INFO]:", filename, ":", updateFileLocationInfo)

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
			for filepath, info := range distributionLocationInfo.filepathInfoMap {
				//append(locationData.locationsInUpdate, info)
				logger.Trace("[DIST FILE INFO] filepath:", filepath, ",Info:", info)

				if updateFileInfo.md5 == info.md5 {
					message := filename + " found in both update, distribution locations. But have the same md5 hash(" + info.md5 + ")" +
						"\n\tLocation in update: " + updateFilePath + filename +
						"\n\tLocation in dist  : CARBON_HOME" + strings.TrimPrefix(filepath, distributionRoot) + filename +
						"\nIt is possible that the old file was copied to the update location instead of the new file."
					return nil, &util.CustomError{What: message }
				} else if updateFileInfo.isDir != info.isDir {
					//Has same type, but different types. Ignore these paths
					continue
				} else {
					//Add
					locationData.locationsInDistribution[filepath] = info.isDir
				}
			}
			log.Trace("[LOCATION DATA] locationData:", locationData)
		} else {
			logger.Debug("[404] No match found in distribution.")

			//Add only the location in the update directory. This will be considered as a new file. Later we can check the locations in the distribution to identify new files.
			locationData.locationsInUpdate[updateFilePath] = updateFileInfo.isDir
		}
		diff.Add(filename, locationData)
	}
	//for filename, locationInfo := range distributionLocationMap.nameLocationMap {
	//	fmt.Println(filename, ":", locationInfo)
	//}
	return &diff, nil
}

//ok
func getUpdateFilePathAndInfo(updateFileLocationInfo *LocationInfo) (string, *Info, error) {
	//Check for duplicate filename. A File and A Directory in the root level of the update directory might have same name(it is highly unlikely). But this is not possible in Ubuntu. Need to check on other OSs
	if len(updateFileLocationInfo.filepathInfoMap) > 1 {
		return "", nil, &util.CustomError{What: "Duplicate files found in the update directory.Possible reason for this error is that there are a file and a directory with the same name."}
	}
	var filepath string
	var locationInfo Info
	for filepath, locationInfo = range updateFileLocationInfo.filepathInfoMap {
		//do nothing
	}
	return filepath, &locationInfo, nil
}

func createFolderStructure(diff *Diff) error {
	for filename, locationData := range diff.files {
		logger.Debug("[CREATE STRUCTURE] filename:", filename)
		logger.Debug("[CREATE STRUCTURE] locationData:", locationData)
		switch len(locationData.locationsInDistribution) {
		case 0:
			err := handleNoMatch(filename, &locationData, false, "")
			util.HandleError(err)
		case 1:
			err := handleSingleMatch(filename, &locationData)
			util.HandleError(err)
		default:
			err := handleMultipleMatches(filename, &locationData)
			util.HandleError(err)
		}
	}
	return nil
}

//ok
func handleNoMatch(filename string, locationData *LocationData, skipUserInput bool, defaultCopyLocation string) error {
	logger.Debug("[NO MATCH]", filename)
	//if skipUserInput {
	//	if len(defaultCopyLocation) > 0 {
	//
	//	} else {
	//
	//	}
	//} else {
	fmt.Print(filename + " not found in distribution. ")
	for {
		fmt.Print("Do you want to add it as a new file? [(Y)es/(N)o]: ")
		preference, err := GetUserInput()
		util.HandleError(err, "Error occurred while getting input from the user.")

		if util.IsYes(preference) {
			err = handleNewFile(filename, locationData)
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
	//}
	return nil
}

//ok
func handleSingleMatch(filename string, locationData *LocationData) error {
	logger.Debug("[SINGLE MATCH]", filename)
	locationInUpdate, isDir, err := getLocationFromMap(locationData.locationsInUpdate)
	util.HandleError(err)
	logger.Debug("[SINGLE MATCH] Location in Update:", locationInUpdate)

	locationInDistribution, _, err := getLocationFromMap(locationData.locationsInDistribution)
	util.HandleError(err)
	logger.Debug("[SINGLE MATCH] Matching location in the Distribution:", locationInDistribution)

	err = copyFile(filename, isDir, locationInUpdate, locationInDistribution)
	util.HandleError(err, "Error occurred while copying the '" + filename + "' ; From " + locationInUpdate + " ; To: " + locationInDistribution)
	return nil
}

func handleMultipleMatches(filename string, locationData *LocationData) error {
	fmt.Println("[MULTIPLE MATCHES]", filename)

	//for filepath, isDir := range locationData.locationsInDistribution {
	//	fmt.Println("filepath:", filepath, "; isDir:", isDir)
	//}

	locationTable, indexMap := generateTable(filename, locationData.locationsInDistribution)
	locationTable.Render()

	fmt.Println("indexMap:", indexMap)
	//todo
	return nil
}

func generateTable(filename string, locationsInDistribution map[string]bool) (*tablewriter.Table, map[string]string) {
	locationTable := tablewriter.NewWriter(os.Stdout)
	locationTable.SetAlignment(tablewriter.ALIGN_LEFT)
	locationTable.SetHeader([]string{"Index", "Location"})

	index := 1
	indexMap := make(map[string]string)
	for filepath, isDir := range locationsInDistribution {
		fmt.Println("filepath:", filepath, "; isDir:", isDir)
		indexMap[strconv.Itoa(index)] = filepath
		locationTable.Append([]string{strconv.Itoa(index), path.Join(filepath, filename)})
		index++
	}

	return locationTable, indexMap
}

//ok
func handleNewFile(filename string, locationData *LocationData) error {
	logger.Debug("[HANDLE NEW] Update:", filename)
	locationInUpdate, isDir, err := getLocationFromMap(locationData.locationsInUpdate)
	util.HandleError(err)
	logger.Debug("[HANDLE NEW] Update:", locationInUpdate, ";", isDir)

	readDestinationLoop:
	for {
		fmt.Print("Enter destination directory relative to CARBON_HOME: ")
		relativePath, err := GetUserInput()
		util.HandleError(err, "Error occurred while getting input from the user.")
		logger.Debug("relativePath:", relativePath)

		fullPath := filepath.Join(viper.GetString(constant.DISTRIBUTION_ROOT), relativePath)
		logger.Debug("fullPath:", fullPath)

		//Ignore error because we are only checking whether the given path exists or not
		exists, _ := util.IsDirectoryExists(fullPath)
		if exists {
			err = copyFile(filename, isDir, locationInUpdate, relativePath)
			util.HandleError(err)
			break
		} else {
			fmt.Print("Entered relative path does not exist in the distribution. ")
			for {
				fmt.Print("Copy anyway? [(Y)es/(N)o/(R)e-enter]: ")
				preference, err := GetUserInput()
				util.HandleError(err, "Error occurred while getting input from the user.")

				if util.IsYes(preference) {
					err = copyFile(filename, isDir, locationInUpdate, relativePath)
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

func copyFile(filename string, isDir bool, locationInUpdate, relativeLocationInTemp string) error {
	fmt.Println("Copying: Name:", filename, "; IsDir:", isDir, "; From:", locationInUpdate, "; To:", relativeLocationInTemp)
	//todo
	//check type
	//if file, check path matches /plugin path
	//else dir, check files in the directory for matches
	return nil
}

func IsABundle(filename, location string) (bool, error) {
	//todo
	return false, nil
}

func ConstructBundleName(filename string) string {
	//todo
	return ""
}

//ok
func getLocationFromMap(locationMap map[string]bool) (string, bool, error) {
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

//ok
func GetUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	preference, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	logger.Debug("[INPUT] User input:", preference)
	return strings.TrimSpace(preference), nil
}

//ok
func isDistributionExists(distributionPath string) (bool, error) {
	if util.HasZipExtension(distributionPath) {
		exists, err := util.IsFileExists(distributionPath)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		} else {
			return false, nil
		}
	} else {
		exists, err := util.IsDirectoryExists(distributionPath)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		} else {
			return false, nil
		}
	}
	return false, nil
}

//This function will set log level
func SetLogLevel() {
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

//ok
func getDistributionRootDirectory(distributionZipPath string) string {
	lastIndex := strings.LastIndex(distributionZipPath, ".")
	return distributionZipPath[:lastIndex]
}

//ok
func readDirectoryStructure(root string, locationMap *FileLocationInfo, ignoredFiles map[string]bool) error {
	//Remove the / or \ at the end of the path if it exists/ Otherwise the root directory wont be ignored
	//root = strings.TrimSuffix(root, string(os.PathSeparator))
	return filepath.Walk(root, func(absolutePath string, fileInfo os.FileInfo, err error) error {
		logger.Trace("[WALK] %s", absolutePath)
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
			//If it is a file, calculate md5 sum
			md5, err := getMD5(absolutePath)
			if err != nil {
				return err
			}
			logger.Debug(absolutePath + " : " + fileInfo.Name() + ": " + md5)
			locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), md5)

			fullPath := filepath.Join(root, constant.PLUGINS_DIRECTORY, fileInfo.Name())
			logger.Trace("[COMPARE] " + fullPath + " ; " + absolutePath)
			if (fullPath == absolutePath) && util.HasJarExtension(absolutePath) {
				logger.Debug("[PLUGIN] FilePath:", absolutePath)
				newFileName := strings.Replace(fileInfo.Name(), "_", "-", 1)
				logger.Debug("[PLUGIN] New Name:", newFileName)
				if index := strings.Index(newFileName, "_"); index != -1 {
					return &util.CustomError{What: fileInfo.Name() + " is in " + constant.PLUGINS_DIRECTORY + ". But it has multiple _ in its name. Only one _ is expected." }
				}
				locationMap.Add(newFileName, parentDirectory, fileInfo.IsDir(), md5)
			}
		} else {
			logger.Debug(absolutePath + " : " + fileInfo.Name())
			locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), "")
		}
		return nil
	})
}

//ok
func getMD5(filepath string) (string, error) {
	var result []byte
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(result)), nil
}

//This function will read update-descriptor.yaml
func readDescriptor(updateDescriptor *update_descriptor, updateDirectory string) error {

	//Construct the file path
	updateDescriptorPath := filepath.Join(updateDirectory, constant.UPDATE_DESCRIPTOR_FILE)
	//Read the file
	yamlFile, err := ioutil.ReadFile(updateDescriptorPath)
	if err != nil {
		return &util.CustomError{What: "Error occurred while reading the descriptor: " + err.Error()}
	}
	//Un-marshal the update-descriptor file to updateDescriptor struct
	err = yaml.Unmarshal(yamlFile, &updateDescriptor)
	if err != nil {
		return &util.CustomError{What: "Error occurred while unmarshalling the yaml: " + err.Error()}
	}
	logger.Debug("----------------------------------------------------------------")
	logger.Debug("update_number: %s", updateDescriptor.Update_number)
	logger.Debug("kernel_version: %s", updateDescriptor.Platform_version)
	logger.Debug("platform_version: %s", updateDescriptor.Platform_name)
	logger.Debug("applies_to: %s", updateDescriptor.Applies_to)
	logger.Debug("bug_fixes: %s", updateDescriptor.Bug_fixes)
	logger.Debug("file_changes: %s", updateDescriptor.File_changes)
	logger.Debug("description: %s", updateDescriptor.Description)
	logger.Debug("----------------------------------------------------------------")

	if len(updateDescriptor.Update_number) == 0 {
		return &util.CustomError{What: "'update_number' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	if len(updateDescriptor.Platform_version) == 0 {
		return &util.CustomError{What: "'platform_version' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	if len(updateDescriptor.Platform_name) == 0 {
		return &util.CustomError{What: "'platform_name' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	if len(updateDescriptor.Applies_to) == 0 {
		return &util.CustomError{What: "'applies_to' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	if len(updateDescriptor.Bug_fixes) == 0 {
		return &util.CustomError{What: "'bug_fixes' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	if len(updateDescriptor.Description) == 0 {
		return &util.CustomError{What: "'description' field not found in " + constant.UPDATE_DESCRIPTOR_FILE}
	}
	return nil
}

//This function will set the update name which will be used when creating the update zip
func setUpdateName(updateDescriptor *update_descriptor) {
	//Read the corresponding details
	platformVersion := updateDescriptor.Platform_version
	logger.Debug("Platform version set to: %s", platformVersion)

	updateNumber := updateDescriptor.Update_number
	logger.Debug("Update number set to: %s", updateNumber)

	updateName := constant.UPDATE_NAME_PREFIX + "-" + platformVersion + "-" + updateNumber
	logger.Debug("Update name: %s", updateName)

	viper.Set(constant.UPDATE_NAME, updateName)
}

//
//func prepareUpdateDescriptorForSaving(updateDescriptor *update_descriptor) {
//	data, err := yaml.Marshal(&updateDescriptor)
//	if err != nil {
//		printFailureAndExit("Error occurred while matshalling the descriptor:", err)
//	}
//	logger.Debug("update-descriptor:\n%s\n\n", string(data))
//
//	color.Set(color.FgGreen)
//	//We need to replace the "" (which will be added when marshalling) enclosing the update number
//	updatedData := strings.Replace(string(data), "\"", "", 2)
//	log.Debug("-------------------------------------------------------------------------------------------------")
//	log.Debug(_UPDATE_DESCRIPTOR_FILE, "-\n")
//	log.Debug(strings.TrimSpace(updatedData))
//	log.Debug("-------------------------------------------------------------------------------------------------")
//	color.Unset()
//
//	destPath := path.Join(_TEMP_DIR, _UPDATE_DESCRIPTOR_FILE)
//	logger.Debug("destPath: %s", destPath)
//	saveUpdateDescriptor(updatedData, destPath)
//}
//
//func saveUpdateDescriptor(data, destPath string) {
//	// Open a new file for writing only
//	file, err := os.OpenFile(
//		destPath,
//		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
//		0777,
//	)
//	if err != nil {
//		printFailureAndExit("Error occurred while opening", _UPDATE_DESCRIPTOR_FILE, ":", err)
//	}
//	defer file.Close()
//
//	// Write bytes to file
//	byteSlice := []byte(data)
//	bytesWritten, err := file.Write(byteSlice)
//	if err != nil {
//		printFailureAndExit("Error occurred while updating", _UPDATE_DESCRIPTOR_FILE, ":", err)
//	}
//	logger.Trace("Wrote %d bytes.\n", bytesWritten)
//}
//
////This method copies resource files to the
//func copyResourceFiles(updateLocation string) {
//	//Copy all mandatory resource files
//	for _, resourceFile := range _MANDATORY_RESOURCE_FILES {
//		filePath := path.Join(updateLocation, resourceFile)
//		exists, err := isFileExists(filePath)
//		if err != nil {
//			//todo: handle error
//		}
//		if !exists {
//			printFailureAndExit("Resource: ", filePath, " not found")
//		}
//		logger.Debug("Copying resource: %s to %s", filePath, _TEMP_DIR)
//		tempPath := path.Join(_TEMP_DIR, resourceFile)
//		err = CopyFile(filePath, tempPath)
//		if (err != nil) {
//			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
//		}
//	}
//
//	//This file is optional
//	filePath := path.Join(updateLocation, _NOT_A_CONTRIBUTION_FILE)
//	exists, err := isFileExists(filePath)
//	if err != nil {
//		//todo: handle error
//	}
//	if !exists {
//		printWarning("'" + _NOT_A_CONTRIBUTION_FILE + "'", "not found in the update directory. Make sure that the Apache License is in '" + _LICENSE_FILE + "' file.")
//	} else {
//		logger.Debug("Copying NOT_A_CONTRIBUTION: %s to %s", filePath, _TEMP_DIR)
//		tempPath := path.Join(_TEMP_DIR, _NOT_A_CONTRIBUTION_FILE)
//		err := CopyFile(filePath, tempPath)
//		if (err != nil) {
//			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
//		}
//	}
//
//	//This file is optional
//	filePath = path.Join(updateLocation, _INSTRUCTIONS_FILE)
//	exists, err = isFileExists(filePath)
//	if err != nil {
//		//todo: handle error
//	}
//	if !exists {
//		printWarning("'" + _INSTRUCTIONS_FILE + "'", "file not found.")
//		printInYellow("Do you want to add an 'instructions.txt' file?[Y/N]: ")
//		for {
//			reader := bufio.NewReader(os.Stdin)
//			preference, _ := reader.ReadString('\n')
//
//			if preference[0] == 'y' || preference[0] == 'Y' {
//				printInfo("Please create an '" + _INSTRUCTIONS_FILE + "' file in the update directory and run the tool again.")
//				//delete temp directory and exit
//				deleteDir(_TEMP_DIR)
//				os.Exit(0)
//			} else if preference[0] == 'n' || preference[0] == 'N' {
//				printWarning("Skipping creating '" + _INSTRUCTIONS_FILE + "' file")
//				break
//			} else {
//				printFailure("Invalid preference. Enter Y for Yes or N for No.")
//				printInYellow("Do you want to add an instructions.txt file?[Y/N]: ")
//			}
//		}
//	} else {
//		logger.Debug("Copying instructions: %s to %s", filePath, _TEMP_DIR)
//		tempPath := path.Join(_TEMP_DIR, _INSTRUCTIONS_FILE)
//		err := CopyFile(filePath, tempPath)
//		if (err != nil) {
//			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
//		}
//	}
//
//	//Copy README.txt. This might be removed in the future. That is why this is copied separately
//	filePath = path.Join(updateLocation, _README_FILE)
//	exists, err = isFileExists(filePath)
//	if err != nil {
//		//todo: handle error
//	}
//	if !exists {
//		printFailureAndExit("Resource: ", filePath, " not found")
//	} else {
//		logger.Debug("Copying readme: %s to %s", filePath, _TEMP_DIR)
//		tempPath := path.Join(_TEMP_DIR, _README_FILE)
//		err := CopyFile(filePath, tempPath)
//		if (err != nil) {
//			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
//		}
//	}
//}

////This is used to find matches of files/directories in the update from distribution
//func findMatches(updateEntriesMap, distEntriesMap *map[string]entry, updateDescriptor *update_descriptor, updateLocation, distributionLocation string) {
//	//Create a new table to display summary
//	overallViewTable := tablewriter.NewWriter(os.Stdout)
//	overallViewTable.SetAlignment(tablewriter.ALIGN_LEFT)
//	overallViewTable.SetHeader([]string{"File/Folder", "Copied To"})
//
//	//Delete temp directory if it exists before proceeding
//	err := deleteTempDir(_TEMP_DIR)
//	if err != nil {
//		if !os.IsNotExist(err) {
//			printFailureAndExit("Error occurred while deleting temp directory:", err)
//		}
//	}
//
//	//Create the temp directory
//	err = createTempDir(_TEMP_DIR)
//	if err != nil {
//		printFailureAndExit("Error occurred while creating temp directory:", err)
//	}
//
//	rowCount := 0
//	//Find matches for each entry in the updateEntriesMap map
//	for fileName, locationInUpdate := range *updateEntriesMap {
//		//Find a match in distEntriesMap
//		distributionEntry, foundInDistribution := (*distEntriesMap)[fileName]
//		//If there is a match
//		if foundInDistribution {
//			logger.Trace("Match found for: %s", fileName)
//			logger.Trace("Location(s) in Dist: %s", distributionEntry)
//			//Get the distribution path. This is later used for trimming
//			absoluteDistributionPath := getDistributionPath(distributionLocation)
//			logger.Trace("Dist Path used for trimming: %s", absoluteDistributionPath)
//			//If there are more than 1 location, we need to ask preferred location(s) from the user
//			if len(distributionEntry.locationMap) > 1 {
//				printWarning("'" + fileName + "' was found in multiple locations in the distribution.")
//				//This is used to temporary store all the locations because we need to access them
//				// later. Since the data is stored in a map, there is no direct way to get the location
//				// using the user preference
//				locationMap := make(map[string]string)
//				//Create the temporary table to show the available locations
//				tempTable := tablewriter.NewWriter(os.Stdout)
//				tempTable.SetHeader([]string{"index", "Location"})
//				//Add locations to the table. Index will be used to get the user preference
//				index := 1
//				for pathInDist, isDirInDist := range distributionEntry.locationMap {
//					for _, isDirInUpdate := range locationInUpdate.locationMap {
//						//We only need to show the files with the same type. (Folder/Folder or
//						// File/File)
//						if isDirInDist == isDirInUpdate {
//							//Add the location to the map. Use the index as the key. This
//							// will allow us to get the user selected locations easier.
//							// Otherwise there is no direct way to get the location from the preference
//							locationMap[strconv.Itoa(index)] = pathInDist
//							logger.Trace("Trimming: %s ; using: %s", pathInDist, absoluteDistributionPath)
//							relativePathInDistribution := strings.TrimPrefix(pathInDist, absoluteDistributionPath) + string(os.PathSeparator)
//							tempTable.Append([]string{strconv.Itoa(index), strings.Replace(relativePathInDistribution, "\\", "/", -1)})
//							index++
//						}
//					}
//				}
//				logger.Trace("Location Map for Dist: %s", locationMap)
//				//Print the temporary table
//				tempTable.Render()
//				//loop until user enter valid indices or decide to exit
//				for {
//					fmt.Print("Enter preferred locations separated by commas[Enter 0 to cancel and exit]: ")
//					//Get the user input
//					reader := bufio.NewReader(os.Stdin)
//					enteredPreferences, _ := reader.ReadString('\n')
//					logger.Trace("enteredPreferences: %s", enteredPreferences)
//					//Remove the new line at the end
//					enteredPreferences = strings.TrimSpace(enteredPreferences)
//					logger.Trace("enteredPreferences2: %s", enteredPreferences)
//					//Split the locations
//					selectedIndices := strings.Split(enteredPreferences, ",");
//					logger.Trace("selectedIndices: %s", selectedIndices)
//					//Sort the locations
//					sort.Strings(selectedIndices)
//					logger.Trace("Sorted indices: %s", selectedIndices)
//
//					if selectedIndices[0] == "0" {
//						printWarning("0 entered. Cancelling the operation and exiting.....")
//						os.Exit(0)
//					} else {
//						//This is used for ???
//						//selectedPathsList := make([]string, 0)
//
//						//This is used to identify whether the all indices are valid
//						isOK := true
//						//Iterate through all the selected indices to check whether all indices
//						// are valid. Later we add entries to the summary table only if all
//						// indices are valid
//						for _, selectedIndex := range selectedIndices {
//							//Check whether the selected index is in the location map. If it
//							// is not in the map, that means an invalid index is entered
//							selectedPath, ok := locationMap[selectedIndex]
//							//If the index is found
//							if ok {
//								logger.Trace("Selected index %s was found in map.", selectedIndex)
//								logger.Trace("selected path: %s", selectedPath)
//								logger.Trace("distPath: %s", absoluteDistributionPath)
//								logger.Trace("distributionLocation: %s", distributionLocation)
//								logger.Trace("Trimming: %s ; using: %s", selectedPath, distributionLocation)
//								tempFilePath := strings.TrimPrefix(selectedPath, distributionLocation)
//								logger.Trace("tempFilePath: %s", tempFilePath)
//
//								//selectedPathsList = append(selectedPathsList, selectedPath)
//
//								src := path.Join(updateLocation, fileName)
//								destPath := path.Join(_UPDATE_DIR_ROOT + tempFilePath)
//								logger.Trace("destPath: %s", destPath)
//								dest := path.Join(destPath, fileName)
//
//								logger.Trace("src 1: %s", src)
//								logger.Trace("dest1: %s", dest)
//								//If source is a file
//								if isFileExists(src) {
//									logger.Debug("Copying file: %s ; To: %s", src, dest)
//									//copy source file to destination
//									copyErr := CopyFile(src, dest)
//									if copyErr != nil {
//										printFailureAndExit("Error occurred while copying file:", copyErr)
//									}
//									logger.Debug("Adding modified file X: ", tempFilePath)
//									//add to modified file
//									updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, path.Join(tempFilePath, fileName))
//								} else if isDirectoryExists(src) {
//									//Compare the directories to identify new files
//									tempPath := path.Join(selectedPath, fileName)
//									compareDir(updateDescriptor, src, tempPath, updateLocation, distributionLocation)
//
//									//If source is a directory
//									logger.Debug("Copying directory: %s ; To: %s", src, dest)
//									//copy source directory to destination
//									copyErr := CopyDir(src, dest)
//									if copyErr != nil {
//										printFailureAndExit("Error occurred while copying directory:", copyErr)
//									}
//								} else {
//									printFailureAndExit("src:", src, "is not a file or a folder")
//								}
//							} else {
//								//If index is invalid
//								printFailure("One or more entered indices are invalid. Please enter again")
//								isOK = false
//								break
//							}
//						}
//						//If all the indices are valid, add the details to the table
//						if isOK {
//							isFirstEntry := true
//							//Iterate through each selected index
//							for _, selectedIndex := range selectedIndices {
//								//Get the corresponding path
//								selectedPath, _ := locationMap[selectedIndex]
//								//Get the relative path
//								relativePathInTempDir := filepath.Join(_CARBON_HOME, strings.TrimPrefix(selectedPath, distributionLocation)) + string(os.PathSeparator)
//								logger.Trace("temp: %s", relativePathInTempDir)
//								//Add the entry to the summary table. If this is the first entry, we want to add fileName and Relative path.
//								//If it is not the first entry, we only need to add relative path.
//								if isFirstEntry {
//									overallViewTable.Append([]string{fileName, strings.Replace(relativePathInTempDir, "\\", "/", -1)})
//									isFirstEntry = false
//								} else {
//									overallViewTable.Append([]string{"", strings.Replace(relativePathInTempDir, "\\", "/", -1)})
//								}
//							}
//							//Break the infinite for loop
//							break
//						}
//					}
//				}
//			} else {
//				//If there is only one match in the distribution
//
//				//Get the location in the distribution (we can use distEntry.locationMap[0] after a
//				// nil check as well)
//				for pathInDistribution, isDirInDist := range distributionEntry.locationMap {
//					//Get the location in the update file
//					for pathInUpdate, isDirInUpdate := range locationInUpdate.locationMap {
//						//Check whether both locations contain same type (files or directories)
//						if isDirInDist == isDirInUpdate {
//							//Add an entry to the table
//							logger.Trace("Both locations contain same type.")
//							logger.Trace("pathInDist: %s", pathInDistribution)
//							logger.Trace("distPath: %s", absoluteDistributionPath)
//							logger.Trace("distributionLocation: %s", distributionLocation)
//							relativeLocation := filepath.Join(_CARBON_HOME, strings.TrimPrefix(pathInDistribution, distributionLocation)) + string(os.PathSeparator)
//							overallViewTable.Append([]string{fileName, strings.Replace(relativeLocation, "\\", "/", -1)})
//							//Get the path relative to the distribution
//							tempFilePath := strings.TrimPrefix(pathInDistribution, distributionLocation)
//							logger.Trace("tempFilePath: %s", tempFilePath)
//							//Construct the source location
//							src := path.Join(pathInUpdate, fileName)
//							destPath := path.Join(_UPDATE_DIR_ROOT + tempFilePath)
//							logger.Trace("destPath: %s", destPath)
//							//Construct the destination location
//							dest := path.Join(destPath, fileName)
//							logger.Trace("src 2: %s", src)
//							logger.Trace("dest2: %s", dest)
//							//Create all directories. Otherwise copy will return an error.
//							// We cannot copy directories in GO. We have to copy file
//							// by file
//							err := os.MkdirAll(destPath, 0777)
//							if err != nil {
//								printFailureAndExit("Error occurred while creating directory", err)
//							}
//							//If source is a file
//							if isFileExists(src) {
//								logger.Debug("Copying file: %s ; To: %s", src, dest)
//								//copy source file to destination
//								copyErr := CopyFile(src, dest)
//								if copyErr != nil {
//									printFailureAndExit("Error occurred while copying file:", copyErr)
//								}
//								logger.Debug("Adding modified file Y: ", tempFilePath)
//								//add to modified file
//								updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, path.Join(tempFilePath, fileName))
//							} else if isDirectoryExists(src) {
//								tempPath := path.Join(pathInDistribution, fileName)
//								//Compare the directories to identify new files
//								compareDir(updateDescriptor, src, tempPath, updateLocation, distributionLocation)
//								//If source is a directory
//								logger.Debug("Copying directory: %s ; To: %s", src, dest)
//								//copy source directory to destination
//								copyErr := CopyDir(src, dest)
//								if copyErr != nil {
//									printFailureAndExit("Error occurred while copying directory:", copyErr)
//								}
//							} else {
//								printFailureAndExit("src:", src, "is not a file or a folder")
//							}
//						} else {
//							//If file types are different(if one is a file and one is a
//							// directory), show a warning message
//							printWarning("Following locations contain", fileName, "but types are different")
//							color.Set(color.FgYellow, color.Bold)
//							fmt.Println(" - ", pathInDistribution)
//							fmt.Println(" - ", pathInUpdate)
//							fmt.Println()
//							color.Unset()
//							typePostfix := " (file)"
//							if isDirInUpdate {
//								typePostfix = " (dir)"
//							}
//							overallViewTable.Append([]string{fileName + typePostfix, " - "})
//						}
//					}
//				}
//			}
//		} else {
//			//If there is no match
//			printWarning("No match found for '" + fileName + "'")
//			color.Set(color.FgYellow, color.Bold)
//			for {
//				fmt.Print("Do you want to add this as a new file/folder?[Y/N]: ")
//				reader := bufio.NewReader(os.Stdin)
//				enteredPreferences, _ := reader.ReadString('\n')
//				logger.Trace("enteredPreferences: %s", enteredPreferences)
//				//Remove the new line at the end
//				enteredPreferences = strings.TrimSpace(enteredPreferences)
//				logger.Debug("enteredPreferences: %s", enteredPreferences)
//				if enteredPreferences[0] == 'Y' || enteredPreferences[0] == 'y' {
//					skipCopy:
//					for {
//						fmt.Print("Enter relative path in the distribution: ")
//						relativePath, _ := reader.ReadString('\n')
//						logger.Trace("copyPath: %s", relativePath)
//						//Remove the new line at the end
//						relativePath = path.Join(distributionLocation, strings.TrimSpace(relativePath))
//						logger.Trace("copyPath2: %s", relativePath)
//						if !isDirectoryExists(relativePath) {
//							for {
//								fmt.Print("Entered relative location does not exist in the " +
//									"distribution. Do you want to copy anyway?[Y/N/R(Re-enter)]: ")
//								enteredPreferences, _ := reader.ReadString('\n')
//								logger.Debug("enteredPreferences: %s", enteredPreferences)
//								//Remove the new line at the end
//								enteredPreferences = strings.TrimSpace(enteredPreferences)
//								logger.Debug("enteredPreferences2: %s", enteredPreferences)
//
//								if enteredPreferences[0] == 'Y' || enteredPreferences[0] == 'y' {
//									//do nothing
//									logger.Debug("Creating the new relative location and copying the file.")
//									break
//								} else if enteredPreferences[0] == 'N' || enteredPreferences[0] == 'n' {
//									logger.Debug("Not creating the new relative location and copying the file.")
//									overallViewTable.Append([]string{fileName, " - "})
//									break skipCopy
//								} else if enteredPreferences[0] == 'R' || enteredPreferences[0] == 'r' {
//									logger.Debug("Re-enter selected.")
//									continue skipCopy
//								} else {
//									printFailure("Invalid preference. Enter Y for Yes, No for No and R to Re-enter a new relative path.")
//									continue
//								}
//							}
//						}
//						//Construct the destination location
//						tempFilePath := strings.TrimPrefix(relativePath, distributionLocation)
//
//						relativeLocation := path.Join(tempFilePath, fileName)
//
//						//Add the new path to added_file section in update-descriptor.yaml
//						updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, relativeLocation)
//
//						printInfo("'" + relativeLocation + "' path was added to 'added_files' " + "section in '" + _UPDATE_DESCRIPTOR_FILE + "'")
//
//						destPath := path.Join(_UPDATE_DIR_ROOT, tempFilePath)
//						logger.Debug("destPath: %s", destPath)
//						dest := path.Join(destPath, fileName)
//						//Create all directories in the path
//						err := os.MkdirAll(destPath, 0777)
//						if err != nil {
//							printFailureAndExit("Error occurred while creating directory", err)
//						}
//						logger.Debug("Entered location is a directory. Copying ...")
//						fileLocation := filepath.Join(updateLocation, fileName)
//						tempDistPath := path.Join(_CARBON_HOME, tempFilePath) + string(os.PathSeparator)
//						//Check for file/folder and copy
//						if isFileExists(fileLocation) {
//							logger.Trace("File found: %s", fileLocation)
//							logger.Debug("Copying file: %s ; To: %s", fileLocation, dest)
//							CopyFile(fileLocation, dest)
//							overallViewTable.Append([]string{fileName, tempDistPath})
//						} else if isDirectoryExists(fileLocation) {
//							logger.Trace("dir found: %s", fileLocation)
//							logger.Debug("Copying file: %s", fileLocation, "; To:", dest)
//							copyErr := CopyDir(fileLocation, dest)
//							if copyErr != nil {
//								printFailureAndExit("Error occurred while copying directory:", copyErr)
//							}
//							overallViewTable.Append([]string{fileName, tempDistPath})
//						} else {
//							logger.Debug("Location not valid:", fileLocation)
//							printFailureAndExit("File not found:", fileLocation)
//						}
//						break
//					}
//					break
//				} else if enteredPreferences[0] == 'N' || enteredPreferences[0] == 'n' {
//					logger.Debug("Not copying file.")
//					logger.Trace("Location(s) in update: %s", locationInUpdate)
//					overallViewTable.Append([]string{fileName, " - "})
//					break
//				} else {
//					printInRed("Invalid preference. Try again.\n")
//				}
//			}
//			color.Unset()
//		}
//		rowCount++
//		if rowCount < len(*updateEntriesMap) {
//			//add separator
//			overallViewTable.Append([]string{" ", " "})
//		}
//	}
//	//Print summary
//	printInYellow("\n# Summary\n")
//	overallViewTable.Render()
//	fmt.Println()
//}
//
////This will compare and print warnings for new files when copying directories from update to temp directory
//func compareDir(updateDescriptor *update_descriptor, pathInUpdate, pathInDist, updateLoc, distLoc string) {
//	logger.Debug("updateLoc: %s", updateLoc)
//	logger.Debug("distLoc: %s", distLoc)
//	//Create maps to store the file details
//	filesInUpdate := make(map[string]bool)
//	filesInDist := make(map[string]bool)
//	logger.Debug("Comparing: %s ; %s", pathInUpdate, pathInDist)
//	//Walk the directory in the update
//	err := filepath.Walk(pathInUpdate, func(path string, fileInfo os.FileInfo, err error) error {
//		logger.Trace("Walking: %s", path)
//		if err != nil {
//			printFailureAndExit("Error occurred while traversing pathInUpdate: ", err)
//		}
//		//We only want to check files
//		if !fileInfo.IsDir() {
//			logger.Trace("File in update: %s", path)
//			//construct the relative path in the distribution
//			tempUpdateFilePath := strings.TrimPrefix(path, pathInUpdate)
//			logger.Trace("tempPath: %s", tempUpdateFilePath)
//			//Add the entry
//			filesInUpdate[tempUpdateFilePath] = true
//		}
//		return nil
//	})
//	if err != nil {
//		printFailureAndExit("Error occurred while traversing pathInUpdate:", err)
//	}
//	//Walk the directory in the distribution
//	err = filepath.Walk(pathInDist, func(path string, fileInfo os.FileInfo, err error) error {
//		logger.Trace("Walking: %s", path)
//		if err != nil {
//			printFailureAndExit("Error occurred while traversing pathInDist: ", err)
//		}
//		//We only want to check files
//		if !fileInfo.IsDir() {
//			logger.Trace("File in dist: ", path)
//			tempDistFilePath := strings.TrimPrefix(path, pathInDist)
//			logger.Trace("tempPath: ", tempDistFilePath)
//			filesInDist[tempDistFilePath] = true
//		}
//		return nil
//	})
//	if err != nil {
//		printFailureAndExit("Error occurred while traversing pathInDist:", err)
//	}
//	//Look for match for each file in update location
//	for path := range filesInUpdate {
//		//Check whether distribution has a match
//		_, found := filesInDist[path]
//		logger.Trace("path: ", path)
//		logger.Trace("pathInDist: ", pathInDist)
//		logger.Trace("distLoc: ", distLoc)
//		tempDistFilePath := strings.TrimPrefix(pathInDist, distLoc)
//		logger.Trace("tempDistFilePath: ", tempDistFilePath)
//		tempPath := strings.Replace(tempDistFilePath + path, "\\", "/", -1)
//		logger.Trace("tempPath: ", tempPath)
//		if found {
//			logger.Debug("Adding modified file: ", tempPath)
//			//add to modified file
//			updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, tempPath)
//			//If a match found, log it
//			logger.Debug("'%s' found in the distribution.", path)
//		} else {
//			//If no match is found, show warning message and add file to added_file section in update-descriptor.yaml
//			printWarning("'" + strings.Replace(strings.TrimPrefix(path, string(os.PathSeparator)), "\\", "/", -1) + "' not found in '" +
//				strings.TrimPrefix(pathInDist + string(os.PathSeparator), distLoc) + "'")
//
//			updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, tempPath)
//
//			printInfo("'" + tempPath + "' path was added to 'added_files' " + "section in '" + _UPDATE_DESCRIPTOR_FILE + "'")
//		}
//	}
//}
//
////Get the path of the distribution location. This is used to trim the prefixes
//func getDistributionPath(distributionLoc string) string {
//	index := strings.LastIndex(distributionLoc, string(os.PathSeparator))
//	if index != -1 {
//		return distributionLoc[:index]
//	} else {
//		return distributionLoc
//	}
//}
//
////Traverse the given path and add entries to the given map
//func traverseAndRead(path string, entryMap *map[string]entry, isDist bool) error {
//	//Get all the files/directories
//	files, err := ioutil.ReadDir(path)
//	if err != nil {
//		return err
//	}
//
//	//update-descriptor.yaml, README, instructions files might be in the update directory. We don't need
//	//to find matches for them. We store them in a map and later check using file names.
//	ignoredFilesMap := make(map[string]bool)
//	ignoredFilesMap[_UPDATE_DESCRIPTOR_FILE] = true
//	ignoredFilesMap[_README_FILE] = true
//	ignoredFilesMap[_INSTRUCTIONS_FILE] = true
//	ignoredFilesMap[_LICENSE_FILE] = true
//	ignoredFilesMap[_NOT_A_CONTRIBUTION_FILE] = true
//
//	//Iterate through all files
//	for _, file := range files {
//		//Check whether the current file is a ignored file
//		_, isInIgnoredMap := ignoredFilesMap[file.Name()]
//		//If not an ignored file, process the file
//		if !isInIgnoredMap {
//			logger.Trace("Checking entry: %s ; path: %s", file.Name(), path)
//			//Check whether the filename is already in the map
//			_, isAlreadyInTheMap := (*entryMap)[file.Name()]
//			if isAlreadyInTheMap {
//				//If the file is already in the map, we only need to add a new entry
//				entry := (*entryMap)[file.Name()]
//				entry.add(path, file.IsDir())
//			} else {
//				//Add a new entry
//				(*entryMap)[file.Name()] = entry{
//					locationMap: map[string]bool{
//						path: file.IsDir(),
//					},
//				}
//			}
//			// This function is used to read both update location and distribution location. We only want
//			// to get the 1st level files/directories in the update. So we don't recursively traverse in
//			// the update location.isDist is used to identify whether this is used to read update or
//			// distribution. If this is the distribution, recursively iterate through all directories
//			if file.IsDir() && isDist {
//				err = traverseAndRead(filepath.Join(path, file.Name()), entryMap, isDist)
//				if err != nil {
//					return err
//				}
//			}
//		}
//	}
//	return nil
//}
//
////This function creates the update zip file
//func createUpdateZip(updateName string) {
//	logger.Debug("Creating update zip file: %s.zip", updateName)
//	//Create the new zip file
//	outFile, err := os.Create(updateName + ".zip")
//	if err != nil {
//		printFailureAndExit("Error occurred while creating the zip file: %s", err)
//	}
//	defer outFile.Close()
//	//Create a zip writer on top of the file writer
//	zipWriter := zip.NewWriter(outFile)
//	//Start traversing
//	err = filepath.Walk(_TEMP_DIR, func(path string, fileInfo os.FileInfo, err error) error {
//		logger.Trace("Walking: %s", path)
//		if err != nil {
//			printFailureAndExit("Error occurred while traversing the temp files: ", err)
//		}
//		//We only want to add the files to the zip. Corresponding directories will be auto created
//		if !fileInfo.IsDir() {
//			//We need to create a header from fileInfo. Otherwise, the file creation time will be set as
//			// the start time in go (1979)
//			header, err := zip.FileInfoHeader(fileInfo)
//			if err != nil {
//				printFailureAndExit("Error occurred while creating the zip file: ", err)
//			}
//			//Construct the file path
//			tempHeaderName := filepath.Join(updateName, strings.TrimPrefix(path, _TEMP_DIR))
//
//			// CRITICAL ----------------------------------------------------------------------------------
//			// If the paths in zip file have \ separators, they will not shown correctly on Ubuntu. But if
//			// we have / path separators, the file paths will be correctly shown in both Windows and
//			// Ubuntu. So we need to replace all \ with / before creating the zip.
//			//--------------------------------------------------------------------------------------------
//			header.Name = strings.Replace(tempHeaderName, "\\", "/", -1)
//			logger.Trace("header.Name: %s", header.Name)
//			//Create a Writer using the header
//			fileWriter, err := zipWriter.CreateHeader(header)
//			if err != nil {
//				printFailureAndExit("Error occurred while creating the zip file: ", err)
//			}
//			//Open the file for reading
//			file, err := os.Open(path)
//			if err != nil {
//				printFailureAndExit("Error occurred when file was open to write to zip:", err)
//			}
//			//Convert the file to byte array
//			data, err := ioutil.ReadAll(file)
//			if err != nil {
//				printFailureAndExit("Error occurred when getting the byte array from the file", err)
//			}
//			//Write the bytes to zip file
//			_, err = fileWriter.Write(data)
//			if err != nil {
//				printFailureAndExit("Error occurred when writing the byte array to the zip file", err)
//			}
//		}
//		return nil
//	})
//	if err != nil {
//		printFailureAndExit("Error occurred while traversing the temp location:", err)
//	}
//	// Close the zip writer
//	err = zipWriter.Close()
//	if err != nil {
//		printFailureAndExit("Error occurred when closing the zip writer", err)
//	}
//	logger.Trace("Directory Walk completed successfully.")
//	color.Set(color.FgGreen)
//	printSuccess("Update file '" + updateName + ".zip' successfully created.")
//	color.Unset()
//}
//
////This function unzips a zip file at given location
//func unzipAndReadDistribution(zipLocation string, distEntriesMap *map[string]entry, logsEnabled bool) {
//	logger.Debug("Unzipping started.")
//	//Get the parent directory of the zip file. This is later used to create the absolute paths of files
//	parentDirectory := "./"
//	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
//		parentDirectory = zipLocation[:lastIndex]
//	}
//	//index := strings.LastIndex(zipLocation, string(os.PathSeparator))
//	//parentDirectory := zipLocation[:index]
//	logger.Debug("parentDirectory: %s", parentDirectory)
//
//	// Create a reader out of the zip archive
//	zipReader, err := zip.OpenReader(zipLocation)
//	if err != nil {
//		printFailureAndExit("Error occurred while reading distribution zip:", err)
//	}
//	//Close the zipReader after the function ends
//	defer zipReader.Close()
//
//	//Get the total number of files in the zip
//	fileCount := len(zipReader.Reader.File)
//	logger.Debug("File count in zip: %s", fileCount)
//
//	//We need to make sure all files are extracted. So we keep count of how many files are extracted
//	extractedFileCount := 0
//	//writer to show the progress
//	writer := uilive.New()
//	//start listening for updates and render
//	writer.Start()
//
//	////Set the target directory to extract
//	//targetDir := "./"
//	//if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
//	//	targetDir = zipLocation[:lastIndex]
//	//}
//
//	// Iterate through each file/dir found in
//	for _, file := range zipReader.Reader.File {
//		extractedFileCount++
//		if (!logsEnabled) {
//			fmt.Fprintf(writer, "Extracting and reading files from distribution zip: (%d/%d)\n", extractedFileCount, fileCount)
//			time.Sleep(time.Millisecond * 2)
//		}
//		logger.Trace("Checking file: %s", file.Name)
//		//Start constructing the full path
//		fullPath := file.Name
//		logger.Trace("fullPath: %s", file.Name)
//		// Open the file inside the zip archive
//		// like a normal file
//		zippedFile, err := file.Open()
//		if err != nil {
//			printFailureAndExit("Error occurred while opening the file:", file, "; Error:", err)
//		}
//		// Specify what the extracted file name should be. You can specify a full path or a prefix to move it
//		// to a different directory. In this case, we will extract the file from the zip to a file of the same
//		// name.
//		extractionPath := filepath.Join(parentDirectory, file.Name) //todo: changed from targetDir to parentDirectory
//		// Extract the item (or create directory)
//		if file.FileInfo().IsDir() {
//			logger.Trace("Is a directory")
//			// Create directories to recreate directory structure inside the zip archive. Also
//			// preserves permissions
//			logger.Trace("Creating directory: ", extractionPath)
//			os.MkdirAll(extractionPath, file.Mode())
//
//			// We only need the location of the directory. fullPath contains the directory name too. We
//			// need to trim and remove the directory name.
//			//string(os.PathSeparator) removed because it does not work properly in windows
//			dir := "/" + file.FileInfo().Name() + "/"
//
//			logger.Trace("Trimming: %s ; using: %s", fullPath, dir)
//			fullPath = strings.TrimSuffix(fullPath, dir)
//			logger.Trace("fullPath: %s", fullPath)
//		} else {
//			logger.Trace("Is a file")
//			// Extract regular file since not a directory
//			//logger.Debug("Extracting file:", file.Name)
//
//			// Open an output file for writing
//			outputFile, err := os.OpenFile(
//				extractionPath,
//				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
//				file.Mode(),
//			)
//			if err != nil {
//				printFailureAndExit("Error occuured while opening the file:", outputFile, "; Error:", err)
//			}
//			if outputFile != nil {
//				// "Extract" the file by copying zipped file
//				// contents to the output file
//				_, err = io.Copy(outputFile, zippedFile)
//				outputFile.Close()
//				if err != nil {
//					printFailureAndExit("Error occuured while opening the file:", file, "; Error:", err)
//				}
//			}
//			// We only need the location of the file. fullPath contains the file name too. We
//			// need to trim and remove the file name.
//
//			//string(os.PathSeparator) removed because it does not work properly in windows
//			logger.Trace("Trimming: %s ; using: /%s", fullPath, file.FileInfo().Name())
//			fullPath = strings.TrimSuffix(fullPath, "/" + file.FileInfo().Name())
//			logger.Trace("fullPath: %s", fullPath)
//		}
//		// Add the distribution location so that the full path will look like it points to locations of the
//		// extracted zip
//		fullPath = path.Join(parentDirectory, fullPath)
//		logger.Trace("FileName: %s ; fullPath: %s", file.FileInfo().Name(), fullPath)
//
//		//Add the entries to the distEntries map
//		addToDistEntryMap(distEntriesMap, file.FileInfo().Name(), file.FileInfo().IsDir(), fullPath)
//
//		//We need to close the file. Otherwise an error will occur because large number of files are open
//		zippedFile.Close()
//	}
//	writer.Stop()
//	logger.Debug("Unzipping finished")
//	logger.Debug("Extracted file count: ", extractedFileCount)
//	if fileCount == extractedFileCount {
//		logger.Debug("All files extracted")
//	} else {
//		logger.Debug("All files not extracted")
//		printFailureAndExit("All files were not extracted. Total files: %s ; Extracted: %s", fileCount, extractedFileCount)
//	}
//}
//
//func addToDistEntryMap(distEntriesMap *map[string]entry, fileName string, isADir bool, fullPath string) {
//	//Check whether the filename is already in the map
//	_, ok := (*distEntriesMap)[fileName]
//
//	//This is to identify whether the location contain a file or a directory
//	isDir := false
//	if isADir {
//		isDir = true
//	}
//	if (ok) {
//		//If the file is already in the map, we only need to add a new entry
//		entry := (*distEntriesMap)[fileName]
//		entry.add(fullPath, isDir)
//	} else {
//		//Add a new entry
//		(*distEntriesMap)[fileName] = entry{
//			locationMap: map[string]bool{
//				fullPath: isDir,
//			},
//		}
//	}
//}
