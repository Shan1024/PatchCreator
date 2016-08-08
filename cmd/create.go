package cmd

import (
	"os"
	"archive/zip"
	"path/filepath"
	"io"
	"strings"
	"io/ioutil"
	"fmt"
	"github.com/fatih/color"
	"time"
	"github.com/gosuri/uilive"
	"bufio"
	"sort"
	"strconv"
	"gopkg.in/yaml.v2"
	"github.com/olekukonko/tablewriter"
	"path"
	"github.com/ian-kent/go-log/log"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/layout"
)

//struct to store matching location(s) in the distribution for a given file/directory
type entry struct {
	locationMap map[string]bool
}

//function used to add locations in distribution of a given file/directory
func (entry *entry) add(path string) {
	entry.locationMap[path] = true
}

//struct that is used to read update-descriptor.yaml
type update_descriptor struct {
	Update_number    string            /*`yaml:"update_number"`*/
	Platform_version string            /*`yaml:"kernel_version"`*/
	Platform_name    string            /*`yaml:"platform_version"`*/
	Applies_to       string            /*`yaml:"applies_to"`*/
	Bug_fixes        map[string]string /*`yaml:"bug_fixes"`*/
	Description      string            /*`yaml:"description"`*/
	File_changes     struct {
				 Added_files    []string /*`yaml:"added_files,flow"`*/
				 Removed_files  []string /*`yaml:"removed_files,flow"`*/
				 Modified_files []string /*`yaml:"removed_files,flow"`*/
			 }
}

var (
	//This contains the mandatory resource files that needs to be copied to the update zip
	//_RESOURCE_FILES = []string{_LICENSE_FILE}

	//These are used to store file/directory locations to later find matches. Keys of the map are file/directory
	// names and the value will be a entry which contain a slice which has locations of that file
	updateEntriesMap map[string]entry
	distEntriesMap map[string]entry

	//This is used to read the update-descriptor.yaml file
	updateDescriptor update_descriptor

	//This holds the complete name of the update zip file/root folder of the zip. This will be a combination of
	// few other variables
	_UPDATE_NAME string

	//Create the logger
	logger = log.Logger()
)

//Main entry point to create the new update
func Create(updateLocation, distributionLocation string, debugLogsEnabled, traceLogsEnabled bool) {
	//Setting default time format. This will be used in loggers. Otherwise complete data, time will be printed
	layout.DefaultTimeLayout = "15:04:05"
	//Setting new layout for STDOUT
	logger.Appender().SetLayout(layout.Pattern("[%d] [%p] %m"))
	//Set the log level. If the log level is not given, set the log level to WARN
	if debugLogsEnabled {
		logger.SetLevel(levels.DEBUG)
		logger.Debug("Debug logs enabled")
	} else if traceLogsEnabled {
		logger.SetLevel(levels.TRACE)
		logger.Trace("Trace logs enabled")
	} else {
		logger.SetLevel(levels.WARN)
	}
	logger.Debug("create command called")

	//Initialize maps
	updateEntriesMap = make(map[string]entry)
	distEntriesMap = make(map[string]entry)

	updateDescriptor = update_descriptor{}

	//Check whether the given update location is a directory.
	updateLocationExists := directoryExists(updateLocation)
	if !updateLocationExists {
		printFailureAndExit("Update location does not exist. Enter a valid directory.")
	}
	logger.Debug("Update location exists.")

	logger.Debug("Distribution Loc: %s", distributionLocation)
	//Check whether the distribution is a zip file.
	if isAZipFile(distributionLocation) {
		//Check whether the distribution zip exists.
		distributionZipExists := fileExists(distributionLocation)
		if !distributionZipExists {
			printFailureAndExit("Distribution zip does not exist. Enter a valid location.")
		}
		logger.Debug("Distribution location exists.")
		unzipAndReadDistribution(distributionLocation, &distEntriesMap, debugLogsEnabled)
		//Delete the extracted distribution directory after method finishes
		defer os.RemoveAll(strings.TrimSuffix(distributionLocation, ".zip"))
	} else {
		//If the distribution is not a zip file, we need to read the files from the distribution directory. So
		//check whether the given distribution directory exists.
		distributionLocationExists := directoryExists(distributionLocation)
		if !distributionLocationExists {
			printFailureAndExit("Distribution location does not exist. Enter a valid location.")
		}
		logger.Debug("Distribution location exists.")
		//Traverse and read the distribution
		logger.Debug("Traversing distribution location")
		traverse(distributionLocation, &distEntriesMap, true)
		logger.Debug("Traversing distribution location finished")
	}

	//This will have the update-descriptor.yaml file location
	updateDescriptorLocation := path.Join(updateLocation, _UPDATE_DESCRIPTOR_FILE)
	logger.Debug("Descriptor Location: %s", updateDescriptorLocation)

	//Check whether the update-descriptor.yaml file exists
	updateDescriptorExists := fileExists(updateDescriptorLocation);
	logger.Debug("Descriptor Exists: %s", updateDescriptorExists)

	if !updateDescriptorExists {
		printFailureAndExit(_UPDATE_DESCRIPTOR_FILE, " not found at ", updateDescriptorLocation)
	}
	//Read the update-descriptor
	readDescriptor(&updateDescriptor, updateDescriptorLocation)

	//Traverse and read the update
	logger.Debug("Traversing update location")
	traverse(updateLocation, &updateEntriesMap, false)
	logger.Debug("Traversing update location finished")
	logger.Debug("Update Entries: ", updateEntriesMap)

	//Find matches
	if isAZipFile(distributionLocation) {
		logger.Debug("Finding matches")
		findMatches(&updateEntriesMap, &distEntriesMap, &updateDescriptor, updateLocation, strings.TrimSuffix(distributionLocation, ".zip"))
		logger.Debug("Finding matches finished")
	} else {
		logger.Debug("Finding matches")
		findMatches(&updateEntriesMap, &distEntriesMap, &updateDescriptor, updateLocation, distributionLocation)
		logger.Debug("Finding matches finished")
	}

	//Copy resource files to the temp location
	logger.Debug("Copying resource files")
	copyResourceFiles(updateLocation)
	logger.Debug("Copying resource files finished")

	//Update the update-descriptor with the newly added files
	updateNewFilesInUpdateDescriptor(&updateDescriptor)

	//Create the update zip file
	logger.Debug("Creating zip file")
	createUpdateZip(_UPDATE_NAME)
	logger.Debug("Creating zip file finished")
}

//This is used to read update-descriptor.yaml
func readDescriptor(updateDescriptor *update_descriptor, path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		printFailureAndExit("Error occurred while reading the descriptor: ", err)
	}

	//Unmarshal the update-descriptor file to updateDescriptor struct
	err = yaml.Unmarshal(yamlFile, &updateDescriptor)
	if err != nil {
		printFailureAndExit("Error occurred while unmarshalling the yaml:", err)
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
		printFailureAndExit("'update_number' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}
	if len(updateDescriptor.Platform_version) == 0 {
		printFailureAndExit("'kernel_version' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}
	if len(updateDescriptor.Platform_name) == 0 {
		printFailureAndExit("'platform_version' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}
	if len(updateDescriptor.Applies_to) == 0 {
		printFailureAndExit("'applies_to' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}
	if len(updateDescriptor.Bug_fixes) == 0 {
		printFailureAndExit("'bug_fixes' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}
	if len(updateDescriptor.Description) == 0 {
		printFailureAndExit("'description' field not found in ", _UPDATE_DESCRIPTOR_FILE)
	}

	updateData(updateDescriptor)
}

func updateData(updateDescriptor *update_descriptor) {
	//Read the corresponding details
	platformVersion := updateDescriptor.Platform_version
	logger.Debug("Platform version set to: %s", platformVersion)

	updateNumber := updateDescriptor.Update_number
	logger.Debug("Update number set to: %s", updateNumber)

	_UPDATE_NAME = _UPDATE_NAME_PREFIX + "-" + platformVersion + "-" + updateNumber
	logger.Debug("Update name: %s", _UPDATE_NAME)
}

func updateNewFilesInUpdateDescriptor(updateDescriptor *update_descriptor) {
	data, err := yaml.Marshal(&updateDescriptor)
	if err != nil {
		printFailureAndExit("Error occurred while matshalling the descriptor:", err)
	}
	logger.Debug("update-descriptor:\n%s\n\n", string(data))

	color.Set(color.FgGreen)
	//We need to
	updatedData := strings.Replace(string(data), "\"", "", 2)
	fmt.Println("-------------------------------------------------------------------------------------------------")
	fmt.Println(_UPDATE_DESCRIPTOR_FILE, "-\n")
	fmt.Println(strings.TrimSpace(updatedData))
	fmt.Println("-------------------------------------------------------------------------------------------------")
	color.Unset()

	destPath := path.Join(_TEMP_DIR, _UPDATE_DESCRIPTOR_FILE)
	logger.Debug("destPath: %s", destPath)
	// Open a new file for writing only
	file, err := os.OpenFile(
		destPath,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0666,
	)
	if err != nil {
		printFailureAndExit("Error occurred while opening", _UPDATE_DESCRIPTOR_FILE, ":", err)
	}
	defer file.Close()

	// Write bytes to file
	byteSlice := []byte(updatedData)
	bytesWritten, err := file.Write(byteSlice)
	if err != nil {
		printFailureAndExit("Error occurred while updating", _UPDATE_DESCRIPTOR_FILE, ":", err)
	}
	logger.Trace("Wrote %d bytes.\n", bytesWritten)
}

//This method copies resource files to the
func copyResourceFiles(patchLocation string) {
	////Copy all mandatory resource files
	//for _, resourceFile := range _RESOURCE_FILES {
	//	filePath := path.Join(_RESOURCE_DIR, resourceFile)
	//	ok := fileExists(filePath)
	//	if !ok {
	//		printFailureAndExit("Resource: ", filePath, " not found")
	//	}
	//	logger.Debug("Copying resource: %s to %s", filePath, _TEMP_DIR)
	//	tempPath := path.Join(_TEMP_DIR, resourceFile)
	//	err := CopyFile(filePath, tempPath)
	//	if (err != nil) {
	//		printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
	//	}
	//}

	//Mandatory file
	filePath := path.Join(patchLocation, _UPDATE_DESCRIPTOR_FILE)
	ok := fileExists(filePath)
	if !ok {
		printFailureAndExit(_UPDATE_DESCRIPTOR_FILE, ":", filePath, " not found")
	} else {
		logger.Debug("Copying: %s to %s", filePath, _TEMP_DIR)
		tempPath := path.Join(_TEMP_DIR, _UPDATE_DESCRIPTOR_FILE)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
		}
	}

	//Mandatory file
	filePath = path.Join(patchLocation, _LICENSE_FILE)
	ok = fileExists(filePath)
	if !ok {
		printFailureAndExit(_LICENSE_FILE, ":", filePath, " not found")
	} else {
		logger.Debug("Copying: %s to %s", filePath, _TEMP_DIR)
		tempPath := path.Join(_TEMP_DIR, _LICENSE_FILE)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
		}
	}

	//This file is optional
	filePath = path.Join(patchLocation, _NOT_A_CONTRIBUTION_FILE)
	ok = fileExists(filePath)
	if !ok {
		printWarning("'" + _NOT_A_CONTRIBUTION_FILE + "'", "not found in the patch directory. Make sure that the Apache License is in '" + _LICENSE_FILE + "' file.")
	} else {
		logger.Debug("Copying NOT_A_CONTRIBUTION: %s to %s", filePath, _TEMP_DIR)
		tempPath := path.Join(_TEMP_DIR, _NOT_A_CONTRIBUTION_FILE)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
		}
	}

	//This file is optional
	filePath = path.Join(patchLocation, _INSTRUCTIONS_FILE)
	ok = fileExists(filePath)
	if !ok {
		printWarning("'" + _INSTRUCTIONS_FILE + "'", "file not found.")
		printInYellow("Do you want to add an 'instructions.txt' file?[Y/N]: ")
		for {
			reader := bufio.NewReader(os.Stdin)
			preference, _ := reader.ReadString('\n')

			if preference[0] == 'y' || preference[0] == 'Y' {
				printInfo("\nPlease create an '" + _INSTRUCTIONS_FILE + "' file in the patch directory and run the tool again.")
				os.Exit(0)
			} else if preference[0] == 'n' || preference[0] == 'N' {
				printWarning("\nSkipping creating '" + _INSTRUCTIONS_FILE + "' file")
				break
			} else {
				printFailure("Invalid preference. Enter Y for Yes or N for No.")
				printInYellow("Do you want to add an instructions.txt file?[Y/N]: ")
			}
		}
	} else {
		logger.Debug("Copying instructions: %s to %s", filePath, _TEMP_DIR)
		tempPath := path.Join(_TEMP_DIR, _INSTRUCTIONS_FILE)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
		}
	}

	//Copy README.txt. This might be removed in the future. That is why this is copied separately
	filePath = path.Join(patchLocation, _README_FILE)
	ok = fileExists(filePath)
	if !ok {
		printFailureAndExit("Resource: ", filePath, " not found")
	} else {
		logger.Debug("Copying readme: %s to %s", filePath, _TEMP_DIR)
		tempPath := path.Join(_TEMP_DIR, _README_FILE)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			printFailureAndExit("Error occurred while copying the resource file: ", filePath, err)
		}
	}
}

//This is used to find matches of files/directories in the patch from distribution
func findMatches(updateEntriesMap, distEntriesMap *map[string]entry, updateDescriptor *update_descriptor, patchLocation, distributionLocation string) {
	//Create a new table to display summary
	overallViewTable := tablewriter.NewWriter(os.Stdout)
	overallViewTable.SetAlignment(tablewriter.ALIGN_LEFT)
	overallViewTable.SetHeader([]string{"File/Folder", "Copied To"})
	//Delete temp directory
	err := os.RemoveAll(_TEMP_DIR)
	if err != nil {
		if !os.IsNotExist(err) {
			printFailureAndExit("Error occurred while deleting temp directory:", err)
		}
	}
	//Create the temp directory
	err = os.MkdirAll(_UPDATE_DIR_ROOT, 0777)
	if err != nil {
		printFailureAndExit("Error occurred while creating temp directory:", err)
	}
	rowCount := 0
	//Find matches for each entry in the updateEntriesMap map
	for fileName, locationInUpdate := range *updateEntriesMap {
		//Find a match in distEntriesMap
		distributionEntry, foundInDistribution := (*distEntriesMap)[fileName]
		//If there is a match
		if foundInDistribution {
			logger.Trace("Match found for: %s", fileName)
			logger.Trace("Location(s) in Dist: %s", distributionEntry)
			//Get the distribution path. This is later used for trimming
			absoluteDistributionPath := getDistributionPath(distributionLocation)
			logger.Trace("Dist Path used for trimming: %s", absoluteDistributionPath)
			//If there are more than 1 location, we need to ask preferred location(s) from the user
			if len(distributionEntry.locationMap) > 1 {
				printWarning("'" + fileName + "' was found in multiple locations in the distribution.")
				//This is used to temporary store all the locations because we need to access them
				// later. Since the data is stored in a map, there is no direct way to get the location
				// using the user preference
				locationMap := make(map[string]string)
				//Create the temporary table to show the available locations
				tempTable := tablewriter.NewWriter(os.Stdout)
				tempTable.SetHeader([]string{"index", "Location"})
				//Add locations to the table. Index will be used to get the user preference
				index := 1
				for pathInDist, isDirInDist := range distributionEntry.locationMap {
					for _, isDirInPatch := range locationInUpdate.locationMap {
						//We only need to show the files with the same type. (Folder/Folder or
						// File/File)
						if isDirInDist == isDirInPatch {
							//Add the location to the map. Use the index as the key. This
							// will allow us to get the user selected locations easier.
							// Otherwise there is no direct way to get the location from the preference
							locationMap[strconv.Itoa(index)] = pathInDist
							logger.Trace("Trimming: %s ; using: %s", pathInDist, absoluteDistributionPath)
							relativePathInDistribution := strings.TrimPrefix(pathInDist, absoluteDistributionPath) + string(os.PathSeparator)
							tempTable.Append([]string{strconv.Itoa(index), strings.Replace(relativePathInDistribution, "\\", "/", -1)})
							index++
						}
					}
				}
				logger.Trace("Location Map for Dist: %s", locationMap)
				//Print the temporary table
				tempTable.Render()
				//loop until user enter valid indices or decide to exit
				for {
					fmt.Print("Enter preferred locations separated by commas[Enter 0 to cancel and exit]: ")
					//Get the user input
					reader := bufio.NewReader(os.Stdin)
					enteredPreferences, _ := reader.ReadString('\n')
					logger.Trace("enteredPreferences: %s", enteredPreferences)
					//Remove the new line at the end
					enteredPreferences = strings.TrimSpace(enteredPreferences)
					logger.Trace("enteredPreferences2: %s", enteredPreferences)
					//Split the locations
					selectedIndices := strings.Split(enteredPreferences, ",");
					logger.Trace("selectedIndices: %s", selectedIndices)
					//Sort the locations
					sort.Strings(selectedIndices)
					logger.Trace("Sorted indices: %s", selectedIndices)

					if selectedIndices[0] == "0" {
						printWarning("0 entered. Cancelling the operation and exiting.....")
						os.Exit(0)
					} else {
						//This is used for ???
						//selectedPathsList := make([]string, 0)

						//This is used to identify whether the all indices are valid
						isOK := true
						//Iterate through all the selected indices to check whether all indices
						// are valid. Later we add entries to the summary table only if all
						// indices are valid
						for _, selectedIndex := range selectedIndices {
							//Check whether the selected index is in the location map. If it
							// is not in the map, that means an invalid index is entered
							selectedPath, ok := locationMap[selectedIndex]
							//If the index is found
							if ok {
								logger.Trace("Selected index %s was found in map.", selectedIndex)
								logger.Trace("selected path: %s", selectedPath)
								logger.Trace("distPath: %s", absoluteDistributionPath)
								logger.Trace("distributionLocation: %s", distributionLocation)
								logger.Trace("Trimming: %s ; using: %s", selectedPath, distributionLocation)
								tempFilePath := strings.TrimPrefix(selectedPath, distributionLocation)
								logger.Trace("tempFilePath: %s", tempFilePath)

								//selectedPathsList = append(selectedPathsList, selectedPath)

								src := path.Join(patchLocation, fileName)
								destPath := path.Join(_UPDATE_DIR_ROOT + tempFilePath)
								logger.Trace("destPath: %s", destPath)
								dest := path.Join(destPath, fileName)

								logger.Trace("src 1: %s", src)
								logger.Trace("dest1: %s", dest)
								//If source is a file
								if fileExists(src) {
									logger.Debug("Copying file: %s ; To: %s", src, dest)
									//copy source file to destination
									copyErr := CopyFile(src, dest)
									if copyErr != nil {
										printFailureAndExit("Error occurred while copying file:", copyErr)
									}
									logger.Debug("Adding modified file X: ", tempFilePath)
									//add to modified file
									updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, path.Join(tempFilePath, fileName))
								} else if directoryExists(src) {
									//Compare the directories to identify new files
									tempPath := path.Join(selectedPath, fileName)
									compareDir(updateDescriptor, src, tempPath, patchLocation, distributionLocation)

									//If source is a directory
									logger.Debug("Copying directory: %s ; To: %s", src, dest)
									//copy source directory to destination
									copyErr := CopyDir(src, dest)
									if copyErr != nil {
										printFailureAndExit("Error occurred while copying directory:", copyErr)
									}
								} else {
									printFailureAndExit("src:", src, "is not a file or a folder")
								}
							} else {
								//If index is invalid
								printFailure("One or more entered indices are invalid. Please enter again")
								isOK = false
								break
							}
						}
						//If all the indices are valid, add the details to the table
						if isOK {
							isFirstEntry := true
							//Iterate through each selected index
							for _, selectedIndex := range selectedIndices {
								//Get the corresponding path
								selectedPath, _ := locationMap[selectedIndex]
								//Get the relative path
								relativePathInTempDir := filepath.Join(_CARBON_HOME, strings.TrimPrefix(selectedPath, distributionLocation)) + string(os.PathSeparator)
								logger.Trace("temp: %s", relativePathInTempDir)
								//Add the entry to the summary table. If this is the first entry, we want to add fileName and Relative path.
								//If it is not the first entry, we only need to add relative path.
								if isFirstEntry {
									overallViewTable.Append([]string{fileName, strings.Replace(relativePathInTempDir, "\\", "/", -1)})
									isFirstEntry = false
								} else {
									overallViewTable.Append([]string{"", strings.Replace(relativePathInTempDir, "\\", "/", -1)})
								}
							}
							//Break the infinite for loop
							break
						}
					}
				}
			} else {
				//If there is only one match in the distribution

				//Get the location in the distribution (we can use distEntry.locationMap[0] after a
				// nil check as well)
				for pathInDistribution, isDirInDist := range distributionEntry.locationMap {
					//Get the location in the patch file
					for pathInPatch, isDirInPatch := range locationInUpdate.locationMap {
						//Check whether both locations contain same type (files or directories)
						if isDirInDist == isDirInPatch {
							//Add an entry to the table
							logger.Trace("Both locations contain same type.")
							logger.Trace("pathInDist: %s", pathInDistribution)
							logger.Trace("distPath: %s", absoluteDistributionPath)
							logger.Trace("distributionLocation: %s", distributionLocation)
							relativeLocation := filepath.Join(_CARBON_HOME, strings.TrimPrefix(pathInDistribution, distributionLocation)) + string(os.PathSeparator)
							overallViewTable.Append([]string{fileName, strings.Replace(relativeLocation, "\\", "/", -1)})
							//Get the path relative to the distribution
							tempFilePath := strings.TrimPrefix(pathInDistribution, distributionLocation)
							logger.Trace("tempFilePath: %s", tempFilePath)
							//Construct the source location
							src := path.Join(pathInPatch, fileName)
							destPath := path.Join(_UPDATE_DIR_ROOT + tempFilePath)
							logger.Trace("destPath: %s", destPath)
							//Construct the destination location
							dest := path.Join(destPath, fileName)
							logger.Trace("src 2: %s", src)
							logger.Trace("dest2: %s", dest)
							//Create all directories. Otherwise copy will return an error.
							// We cannot copy directories in GO. We have to copy file
							// by file
							err := os.MkdirAll(destPath, 0777)
							if err != nil {
								printFailureAndExit("Error occurred while creating directory", err)
							}
							//If source is a file
							if fileExists(src) {
								logger.Debug("Copying file: %s ; To: %s", src, dest)
								//copy source file to destination
								copyErr := CopyFile(src, dest)
								if copyErr != nil {
									printFailureAndExit("Error occurred while copying file:", copyErr)
								}
								logger.Debug("Adding modified file Y: ", tempFilePath)
								//add to modified file
								updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, path.Join(tempFilePath, fileName))
							} else if directoryExists(src) {
								tempPath := path.Join(pathInDistribution, fileName)
								//Compare the directories to identify new files
								compareDir(updateDescriptor, src, tempPath, patchLocation, distributionLocation)
								//If source is a directory
								logger.Debug("Copying directory: %s ; To: %s", src, dest)
								//copy source directory to destination
								copyErr := CopyDir(src, dest)
								if copyErr != nil {
									printFailureAndExit("Error occurred while copying directory:", copyErr)
								}
							} else {
								printFailureAndExit("src:", src, "is not a file or a folder")
							}
						} else {
							//If file types are different(if one is a file and one is a
							// directory), show a warning message
							printWarning("Following locations contain", fileName, "but types are different")
							color.Set(color.FgYellow, color.Bold)
							fmt.Println(" - ", pathInDistribution)
							fmt.Println(" - ", pathInPatch)
							fmt.Println()
							color.Unset()
							typePostfix := " (file)"
							if isDirInPatch {
								typePostfix = " (dir)"
							}
							overallViewTable.Append([]string{fileName + typePostfix, " - "})
						}
					}
				}
			}
		} else {
			//If there is no match
			printWarning("No match found for '" + fileName + "'")
			color.Set(color.FgYellow, color.Bold)
			for {
				fmt.Print("Do you want to add this as a new file/folder?[Y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				enteredPreferences, _ := reader.ReadString('\n')
				logger.Trace("enteredPreferences: %s", enteredPreferences)
				//Remove the new line at the end
				enteredPreferences = strings.TrimSpace(enteredPreferences)
				logger.Debug("enteredPreferences: %s", enteredPreferences)
				if enteredPreferences[0] == 'Y' || enteredPreferences[0] == 'y' {
					skipCopy:
					for {
						fmt.Print("Enter relative path in the distribution: ")
						relativePath, _ := reader.ReadString('\n')
						logger.Trace("copyPath: %s", relativePath)
						//Remove the new line at the end
						relativePath = path.Join(distributionLocation, strings.TrimSpace(relativePath))
						logger.Trace("copyPath2: %s", relativePath)
						if !directoryExists(relativePath) {
							for {
								fmt.Print("Entered relative location does not exist in the " +
									"distribution. Do you want to copy anyway?[Y/N/R(Re-enter)]: ")
								enteredPreferences, _ := reader.ReadString('\n')
								logger.Debug("enteredPreferences: %s", enteredPreferences)
								//Remove the new line at the end
								enteredPreferences = strings.TrimSpace(enteredPreferences)
								logger.Debug("enteredPreferences2: %s", enteredPreferences)

								if enteredPreferences[0] == 'Y' || enteredPreferences[0] == 'y' {
									//do nothing
									logger.Debug("Creating the new relative location and copying the file.")
									break
								} else if enteredPreferences[0] == 'N' || enteredPreferences[0] == 'n' {
									logger.Debug("Not creating the new relative location and copying the file.")
									overallViewTable.Append([]string{fileName, " - "})
									break skipCopy
								} else if enteredPreferences[0] == 'R' || enteredPreferences[0] == 'r' {
									logger.Debug("Re-enter selected.")
									continue skipCopy
								} else {
									fmt.Println("Invalid preference. Try again.\n")
									continue
								}
							}
						}
						//Construct the destination location
						tempFilePath := strings.TrimPrefix(relativePath, distributionLocation)

						relativeLocation := path.Join(tempFilePath, fileName)

						//Add the new path to added_file section in update-descriptor.yaml
						updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, relativeLocation)

						printInfo("'" + relativeLocation + "' path was added to 'added_files' " + "section in '" + _UPDATE_DESCRIPTOR_FILE + "'")

						destPath := path.Join(_UPDATE_DIR_ROOT, tempFilePath)
						logger.Debug("destPath: %s", destPath)
						dest := path.Join(destPath, fileName)
						//Create all directories in the path
						err := os.MkdirAll(destPath, 0777)
						if err != nil {
							printFailureAndExit("Error occurred while creating directory", err)
						}
						logger.Debug("Entered location is a directory. Copying ...")
						fileLocation := filepath.Join(patchLocation, fileName)
						tempDistPath := path.Join(_CARBON_HOME, tempFilePath) + string(os.PathSeparator)
						//Check for file/folder and copy
						if fileExists(fileLocation) {
							logger.Trace("File found: %s", fileLocation)
							//copyPath = path.Join(copyPath, patchEntryName)
							logger.Debug("Copying file: %s ; To: %s", fileLocation, dest)
							CopyFile(fileLocation, dest)
							overallViewTable.Append([]string{fileName, tempDistPath})
						} else if directoryExists(fileLocation) {
							logger.Trace("dir found: %s", fileLocation)
							//copyPath = path.Join(copyPath, patchEntryName)
							logger.Debug("Copying file: %s", fileLocation, "; To:", dest)
							CopyDir(fileLocation, dest)
							overallViewTable.Append([]string{fileName, tempDistPath})
						} else {
							logger.Debug("Location not valid:", fileLocation)
							printFailureAndExit("File not found:", fileLocation)
						}
						break
					}
					break
				} else if enteredPreferences[0] == 'N' || enteredPreferences[0] == 'n' {
					logger.Debug("Not copying file.")
					logger.Trace("Location(s) in Patch: %s", locationInUpdate)
					overallViewTable.Append([]string{fileName, " - "})
					break
				} else {
					fmt.Println("Invalid preference. Try again.\n")
				}
			}
			color.Unset()
		}
		rowCount++
		if rowCount < len(*updateEntriesMap) {
			//add separator
			overallViewTable.Append([]string{" ", " "})
		}
	}
	//Print summary
	fmt.Println("\n# Summary\n")
	overallViewTable.Render()
	fmt.Println()
}

//This will compare and print warnings for new files when copying directories from patch to temp directory
func compareDir(updateDescriptor *update_descriptor, pathInPatch, pathInDist, patchLoc, distLoc string) {
	logger.Debug("patchLoc: %s", patchLoc)
	logger.Debug("distLoc: %s", distLoc)
	//Create maps to store the file details
	filesInPatch := make(map[string]bool)
	filesInDist := make(map[string]bool)
	logger.Debug("Comparing: %s ; %s", pathInPatch, pathInDist)
	//Walk the directory in the patch
	err := filepath.Walk(pathInPatch, func(path string, fileInfo os.FileInfo, err error) error {
		logger.Trace("Walking: %s", path)
		if err != nil {
			printFailureAndExit("Error occurred while traversing pathInPatch: ", err)
		}
		//We only want to check files
		if !fileInfo.IsDir() {
			logger.Trace("File in patch: %s", path)
			//construct the relative path in the distribution
			tempPatchFilePath := strings.TrimPrefix(path, pathInPatch)
			logger.Trace("tempPath: %s", tempPatchFilePath)
			//Add the entry
			filesInPatch[tempPatchFilePath] = true
		}
		return nil
	})
	if err != nil {
		printFailureAndExit("Error occurred while traversing pathInPatch:", err)
	}
	//Walk the directory in the distribution
	err = filepath.Walk(pathInDist, func(path string, fileInfo os.FileInfo, err error) error {
		logger.Trace("Walking: %s", path)
		if err != nil {
			printFailureAndExit("Error occurred while traversing pathInDist: ", err)
		}
		//We only want to check files
		if !fileInfo.IsDir() {
			logger.Trace("File in dist: ", path)
			tempDistFilePath := strings.TrimPrefix(path, pathInDist)
			logger.Trace("tempPath: ", tempDistFilePath)
			filesInDist[tempDistFilePath] = true
		}
		return nil
	})
	if err != nil {
		printFailureAndExit("Error occurred while traversing pathInDist:", err)
	}
	//Look for match for each file in patch location
	for path := range filesInPatch {
		//Check whether distribution has a match
		_, found := filesInDist[path]
		logger.Trace("path: ", path)
		logger.Trace("pathInDist: ", pathInDist)
		logger.Trace("distLoc: ", distLoc)
		tempDistFilePath := strings.TrimPrefix(pathInDist, distLoc)
		logger.Trace("tempDistFilePath: ", tempDistFilePath)
		tempPath := strings.Replace(tempDistFilePath + path, "\\", "/", -1)
		logger.Trace("tempPath: ", tempPath)
		if found {
			logger.Debug("Adding modified file: ", tempPath)
			//add to modified file
			updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, tempPath)
			//If a match found, log it
			logger.Debug("'%s' found in the distribution.", path)
		} else {
			//If no match is found, show warning message and add file to added_file section in update-descriptor.yaml
			printWarning("'" + strings.Replace(strings.TrimPrefix(path, string(os.PathSeparator)), "\\", "/", -1) + "' not found in '" +
				strings.TrimPrefix(pathInDist + string(os.PathSeparator), distLoc) + "'")

			updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, tempPath)

			printInfo("'" + tempPath + "' path was added to 'added_files' " + "section in '" + _UPDATE_DESCRIPTOR_FILE + "'")
		}
	}
}

//Get the path of the distribution location. This is used to trim the prefixes
func getDistributionPath(distributionLoc string) string {
	index := strings.LastIndex(distributionLoc, string(os.PathSeparator))
	if index != -1 {
		return distributionLoc[:index]
	} else {
		return distributionLoc
	}
}

//Traverse the given path and add entries to the given map
func traverse(path string, entryMap *map[string]entry, isDist bool) {
	//Get all the files/directories
	files, _ := ioutil.ReadDir(path)
	//Iterate through all files
	for _, file := range files {
		//update-descriptor, README, instructions files might be in the patch location. We don't need to find
		// matches for them
		if file.Name() != _UPDATE_DESCRIPTOR_FILE && file.Name() != _README_FILE && file.Name() != _INSTRUCTIONS_FILE && file.Name() != _LICENSE_FILE && file.Name() != _NOT_A_CONTRIBUTION_FILE {
			logger.Trace("Checking entry: %s ; path: %s", file.Name(), path)
			//Check whether the filename is already in the map
			_, ok := (*entryMap)[file.Name()]
			if (ok) {
				//If the file is already in the map, we only need to add a new entry
				entry := (*entryMap)[file.Name()]
				entry.add(path)
			} else {
				//This is to identify whether the location contain a file or a directory
				isDir := false
				if file.IsDir() {
					isDir = true
				}
				//Add a new entry
				(*entryMap)[file.Name()] = entry{
					locationMap: map[string]bool{
						path: isDir,
					},
				}
			}
			// This function is used to read both patch location and distribution location. We only want
			// to get the 1st level files/directories in the patch. So we don't recursively traverse in
			// the patch location.isDist is used to identify whether this is used to read patch or
			// distribution. If this is the distribution, recursively iterate
			if file.IsDir() && isDist {
				traverse(path + string(os.PathSeparator) + file.Name(), entryMap, isDist)
			}
		}
	}
}

//This function creates the patch zip file
func createUpdateZip(updateName string) {
	logger.Debug("Creating patch zip file: %s.zip", updateName)
	//Create the new zip file
	outFile, err := os.Create(updateName + ".zip")
	if err != nil {
		printFailureAndExit("Error occurred while creating the zip file: %s", err)
	}
	defer outFile.Close()
	//Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)
	//Start traversing
	err = filepath.Walk(_TEMP_DIR, func(path string, fileInfo os.FileInfo, err error) error {
		logger.Trace("Walking: %s", path)
		if err != nil {
			printFailureAndExit("Error occurred while traversing the temp files: ", err)
		}
		//We only want to add the files to the zip. Corresponding directories will be auto created
		if !fileInfo.IsDir() {
			//We need to create a header from fileInfo. Otherwise, the file creation time will be set as
			// the start time in go (1979)
			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				printFailureAndExit("Error occurred while creating the zip file: ", err)
			}
			//Construct the file path
			tempHeaderName := filepath.Join(updateName, strings.TrimPrefix(path, _TEMP_DIR))

			// CRITICAL ----------------------------------------------------------------------------------
			// If the paths in zip file have \ separators, they will not shown correctly on Ubuntu. But if
			// we have / path separators, the file paths will be correctly shown in both Windows and
			// Ubuntu. So we need to replace all \ with / before creating the zip.
			//--------------------------------------------------------------------------------------------
			header.Name = strings.Replace(tempHeaderName, "\\", "/", -1)
			logger.Trace("header.Name: %s", header.Name)
			//Create a Writer using the header
			fileWriter, err := zipWriter.CreateHeader(header)
			if err != nil {
				printFailureAndExit("Error occurred while creating the zip file: ", err)
			}
			//Open the file for reading
			file, err := os.Open(path)
			if err != nil {
				printFailureAndExit("Error occurred when file was open to write to zip:", err)
			}
			//Convert the file to byte array
			data, err := ioutil.ReadAll(file)
			if err != nil {
				printFailureAndExit("Error occurred when getting the byte array from the file", err)
			}
			//Write the bytes to zip file
			_, err = fileWriter.Write(data)
			if err != nil {
				printFailureAndExit("Error occurred when writing the byte array to the zip file", err)
			}
		}
		return nil
	})
	if err != nil {
		printFailureAndExit("Error occurred while traversing the temp location:", err)
	}
	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		printFailureAndExit("Error occurred when closing the zip writer", err)
	}
	logger.Trace("Directory Walk completed successfully.")
	color.Set(color.FgGreen)
	printSuccess("Update file '" + updateName + ".zip' successfully created.")
	color.Unset()
}

//This function unzips a zip file at given location
func unzipAndReadDistribution(zipLocation string, distEntriesMap *map[string]entry, logsEnabled bool) {
	logger.Debug("Unzipping started.")
	//Get the location of the zip file. This is later used to create the full path of a file
	index := strings.LastIndex(zipLocation, string(os.PathSeparator))
	distLocation := zipLocation[:index]
	logger.Debug("distLocation: %s", distLocation)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)
	if err != nil {
		printFailureAndExit("Error occurred while reading zip:", err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	logger.Debug("File count in zip: %s", totalFiles)

	filesRead := 0
	// writer to show the progress
	writer := uilive.New()
	// start listening for updates and render
	writer.Start()
	targetDir := "./"
	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
		targetDir = zipLocation[:lastIndex]
	}
	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		filesRead++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Extracting and reading files from distribution zip: (%d/%d)\n", filesRead, totalFiles)
			time.Sleep(time.Millisecond * 2)
		}
		logger.Trace("Checking file: %s", file.Name)
		//Start constructing the full path
		fullPath := file.Name
		logger.Trace("fullPath: %s", file.Name)
		// Open the file inside the zip archive
		// like a normal file
		zippedFile, err := file.Open()
		if err != nil {
			printFailureAndExit("Error occuured while opening the file:", file, "; Error:", err)
		}
		// Specify what the extracted file name should be. You can specify a full path or a prefix to move it
		// to a different directory. In this case, we will extract the file from the zip to a file of the same
		// name.
		extractionPath := filepath.Join(
			targetDir,
			file.Name,
		)
		// Extract the item (or create directory)
		if file.FileInfo().IsDir() {
			logger.Trace("Is a directory")
			// Create directories to recreate directory
			// structure inside the zip archive. Also
			// preserves permissions
			//logger.Debug("Creating directory:", extractionPath)
			os.MkdirAll(extractionPath, file.Mode())

			// We only need the location of the directory. fullPath contains the directory name too. We
			// need to trim and remove the directory name.
			//string(os.PathSeparator) removed because it does not work properly in windows
			dir := "/" + file.FileInfo().Name() + "/"

			logger.Trace("Trimming: %s ; using: %s", fullPath, dir)
			fullPath = strings.TrimSuffix(fullPath, dir)
			logger.Trace("fullPath: %s", fullPath)
		} else {
			logger.Trace("Is a file")
			// Extract regular file since not a directory
			//logger.Debug("Extracting file:", file.Name)

			// Open an output file for writing
			outputFile, err := os.OpenFile(
				extractionPath,
				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				file.Mode(),
			)
			if err != nil {
				printFailureAndExit("Error occuured while opening the file:", outputFile, "; Error:", err)
			}
			if outputFile != nil {
				// "Extract" the file by copying zipped file
				// contents to the output file
				_, err = io.Copy(outputFile, zippedFile)
				outputFile.Close()
				if err != nil {
					printFailureAndExit("Error occuured while opening the file:", file, "; Error:", err)
				}
			}
			// We only need the location of the file. fullPath contains the file name too. We
			// need to trim and remove the file name.

			//string(os.PathSeparator) removed because it does not work properly in windows
			logger.Trace("Trimming: %s ; using: /%s", fullPath, file.FileInfo().Name())
			fullPath = strings.TrimSuffix(fullPath, "/" + file.FileInfo().Name())
			logger.Trace("fullPath: %s", fullPath)
		}
		// Add the distribution location so that the full path will look like it points to locations of the
		// extracted zip
		fullPath = path.Join(distLocation, fullPath)
		logger.Trace("FileName: %s ; fullPath: %s", file.FileInfo().Name(), fullPath)

		//Add the entries to the distEntries map
		addToDistEntryMap(distEntriesMap, file, fullPath)

		//We need to close the file. Otherwise an error will occur because large number of files are open
		zippedFile.Close()
	}
	writer.Stop()
	logger.Debug("Unzipping finished")
	logger.Debug("Extracted file count: ", filesRead)
	if totalFiles == filesRead {
		logger.Debug("All files extracted")
	} else {
		logger.Debug("All files not extracted")
		printFailureAndExit("All files were not extracted. Total files: %s ; Extracted: %s", totalFiles, filesRead)
	}
}

func addToDistEntryMap(distEntriesMap *map[string]entry, file *zip.File, fullPath string) {
	//Check whether the filename is already in the map
	_, ok := (*distEntriesMap)[(*file).FileInfo().Name()]
	if (ok) {
		//If the file is already in the map, we only need to add a new entry
		entry := (*distEntriesMap)[file.FileInfo().Name()]
		entry.add(fullPath)
	} else {
		//This is to identify whether the location contain a file or a directory
		isDir := false
		if (*file).FileInfo().IsDir() {
			isDir = true
		}
		//Add a new entry
		(*distEntriesMap)[(*file).FileInfo().Name()] = entry{
			locationMap: map[string]bool{
				fullPath: isDir,
			},
		}
	}
}
