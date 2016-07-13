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
	"text/template"
	"github.com/olekukonko/tablewriter"
	"path"
	"github.com/ian-kent/go-log/log"
	"github.com/ian-kent/go-log/levels"
)

//struct to store location(s) in the distribution for a given file/directory
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
	Kernel_version   string            /*`yaml:"kernel_version"`*/
	Platform_version string            /*`yaml:"platform_version"`*/
	Applies_to       string            /*`yaml:"applies_to"`*/
	Bug_fixes        map[string]string /*`yaml:"bug_fixes"`*/
	Description      string            /*`yaml:"description"`*/
	File_changes     struct {
				 Added_files   []string /*`yaml:"added_files,flow"`*/
				 Removed_files []string /*`yaml:"removed_files,flow"`*/
			 }
}

const (
	//constants to store resource files
	_README_FILE_NAME = "README.txt"
	_LICENSE_FILE_NAME = "LICENSE.txt"
	_NOT_A_CONTRIBUTION_FILE_NAME = "NOT_A_CONTRIBUTION.txt"
	_INSTRUCTIONS_FILE_NAME = "instructions.txt"
	_UPDATE_DESCRIPTOR_FILE_NAME = "update-descriptor.yaml"

	//Resource directory which contains README,LICENCE and NOT_A_CONTRIBUTION files
	_RESOURCE_DIR = ".." + string(os.PathSeparator) + "res"
	//Temporary directory to copy files before creating the new zip
	_TEMP_DIR_NAME = "temp"
	//This is used to store carbon.home string
	_CARBON_HOME = "carbon.home"
	//Temporary directory location including carbon.home. All patched files will be copied to this location
	_TEMP_DIR_LOCATION = _TEMP_DIR_NAME + string(os.PathSeparator) + _CARBON_HOME
	//Prefix of the update file and the root folder of the update zip
	_UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"
	//Prefix of the JIRA urls. This is used to generate README
	_JIRA_URL_PREFIX = "https://wso2.org/jira/browse/"
)

var (
	//This contains the mandatory resource files that needs to be copied to the update zip
	_RESOURCE_FILES = []string{_LICENSE_FILE_NAME}

	//These are used to store file/directory locations to later find matches. Keys of the map are file/directory
	// names and the value will be a entry which contain a slice which has locations of that file
	patchEntries map[string]entry
	distEntries map[string]entry

	//This is used to read the update-descriptor.yaml file
	descriptor update_descriptor

	//This holds the kernel version. This will be read from the update-descriptor.yaml
	_KERNEL_VERSION string
	//This holds the update number. This will be read from the update-descriptor.yaml
	_UPDATE_NUMBER string
	//This holds the complete name of the update zip file/root folder of the zip. This will be a combination of
	// few other variables
	_UPDATE_NAME string

	//Create the logger
	logger = log.Logger()
)

//Main entry point to create the new patch
func Create(patchLocation, distributionLocation string, debugLogsEnabled, traceLogsEnabled bool) {
	//Set the log level. If the log level is not given, set the log level to WARN
	if (debugLogsEnabled) {
		logger.SetLevel(levels.DEBUG)
		logger.Debug("Logs enabled")
	} else if (traceLogsEnabled) {
		logger.SetLevel(levels.TRACE)
		logger.Debug("Logs enabled")
	} else {
		logger.SetLevel(levels.WARN)
	}
	logger.Debug("create command called")

	//Initialize maps
	patchEntries = make(map[string]entry)
	distEntries = make(map[string]entry)

	//Check whether the given patch location exists. It should be a directory.
	patchLocationExists := checkDir(patchLocation)
	if patchLocationExists {
		logger.Debug("Patch location exists.")
	} else {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Patch location does not exist. Enter a valid directory.")
		color.Unset()
		os.Exit(1)
	}
	//Check whether the given distribution is a zip file.
	if (isAZipFile(distributionLocation)) {
		//Check whether the given distribution zip exists.
		zipFileExists := checkFile(distributionLocation)
		if zipFileExists {
			logger.Debug("Distribution location exists.")
			unzipAndReadDistribution(distributionLocation, debugLogsEnabled)

			//Delete the extracted distribution directory
			defer os.RemoveAll(strings.TrimSuffix(distributionLocation, ".zip"))
		} else {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Distribution zip does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	} else {
		//If the distribution is not a zip file, we need to read the files from the distribution directory

		//Check whether the given distribution directory exists.
		distributionLocationExists := checkDir(distributionLocation)
		if distributionLocationExists {
			logger.Debug("Distribution location exists.")
			//Traverse and read the distribution
			logger.Debug("Traversing distribution location")
			traverse(distributionLocation, distEntries, true)
			logger.Debug("Traversing distribution location finished")
		} else {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Distribution location does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	}

	logger.Debug("Distribution Loc: %s", distributionLocation)

	//This will have the update-descriptor.yaml file location
	descriptorLocation := path.Join(patchLocation, _UPDATE_DESCRIPTOR_FILE_NAME)
	logger.Debug("Descriptor Location: %s", descriptorLocation)

	//Check whether the update-descriptor.yaml file exists
	descriptorExists := checkFile(descriptorLocation);
	logger.Debug("Descriptor Exists: %s", descriptorExists)
	if descriptorExists {
		readDescriptor(descriptorLocation)
	} else {
		//readPatchInfo()
		color.Set(color.FgRed)
		fmt.Println("[FAILURE]", _UPDATE_DESCRIPTOR_FILE_NAME + " not found at " + descriptorLocation)
		color.Unset()
		os.Exit(1)
	}

	//Traverse and read the patch
	logger.Debug("Traversing patch location")
	traverse(patchLocation, patchEntries, false)
	logger.Debug("Traversing patch location finished")
	logger.Debug("Patch Entries: ", patchEntries)

	//Find matches
	if (isAZipFile(distributionLocation)) {
		logger.Debug("Finding matches")
		findMatches(patchLocation, strings.TrimSuffix(distributionLocation, ".zip"))
		logger.Debug("Finding matches finished")
	} else {
		logger.Debug("Finding matches")
		findMatches(patchLocation, distributionLocation)
		logger.Debug("Finding matches finished")
	}

	//Copy resource files to the temp location
	logger.Debug("Copying resource files")
	copyResourceFiles(patchLocation)
	logger.Debug("Copying resource files finished")

	updateYAML()

	//Create the update zip file
	logger.Debug("Creating zip file")
	createUpdateZip()
	logger.Debug("Creating zip file finished")
}

//This is used to read update-descriptor.yaml
func readDescriptor(path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while reading the descriptor: ", err)
		color.Unset()
	}
	descriptor = update_descriptor{}

	err = yaml.Unmarshal(yamlFile, &descriptor)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while unmarshalling the yaml:", err)
		color.Unset()
	}

	logger.Debug("----------------------------------------------------------------")
	logger.Debug("update_number: %s", descriptor.Update_number)
	logger.Debug("kernel_version: %s", descriptor.Kernel_version)
	logger.Debug("platform_version: %s", descriptor.Platform_version)
	logger.Debug("applies_to: %s", descriptor.Applies_to)
	logger.Debug("bug_fixes: %s", descriptor.Bug_fixes)
	logger.Debug("file_changes: %s", descriptor.File_changes)
	logger.Debug("description: %s", descriptor.Description)
	logger.Debug("----------------------------------------------------------------")

	if len(descriptor.Update_number) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'update_number' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Kernel_version) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'kernel_version' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Platform_version) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'platform_version' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Applies_to) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'applies_to' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Bug_fixes) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'bug_fixes' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Description) == 0 {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] 'description' field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}

	_KERNEL_VERSION = descriptor.Kernel_version
	logger.Debug("kernel version set to: %s", _KERNEL_VERSION)

	_UPDATE_NUMBER = descriptor.Update_number
	logger.Debug("patch number set to: %s", _UPDATE_NUMBER)

	_UPDATE_NAME = _UPDATE_NAME_PREFIX + "-" + _KERNEL_VERSION + "-" + _UPDATE_NUMBER
	logger.Debug("Patch Name: %s", _UPDATE_NAME)
}

func updateYAML() {
	data, err := yaml.Marshal(&descriptor)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while matshalling the descriptor:", err)
		color.Unset()
		os.Exit(1)
	}
	logger.Debug("update-descriptor:\n%s\n\n", string(data))

	color.Set(color.FgGreen)
	updatedData := strings.Replace(string(data), "\"", "", 2)
	fmt.Println("-------------------------------------------------------------------------------------------------")
	fmt.Println(_UPDATE_DESCRIPTOR_FILE_NAME, "-\n")
	fmt.Println(strings.TrimSpace(updatedData))
	fmt.Println("-------------------------------------------------------------------------------------------------")
	color.Unset()

	destPath := path.Join(_TEMP_DIR_NAME, _UPDATE_DESCRIPTOR_FILE_NAME)
	logger.Debug("destPath: %s", destPath)
	// Open a new file for writing only
	file, err := os.OpenFile(
		destPath,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0666,
	)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while opening", _UPDATE_DESCRIPTOR_FILE_NAME, ":", err)
		color.Unset()
		os.Exit(1)
	}
	defer file.Close()

	// Write bytes to file
	byteSlice := []byte(updatedData)
	bytesWritten, err := file.Write(byteSlice)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while updating", _UPDATE_DESCRIPTOR_FILE_NAME, ":", err)
		color.Unset()
		os.Exit(1)
	}
	logger.Trace("Wrote %d bytes.\n", bytesWritten)
}

//This method copies resource files to the
func copyResourceFiles(patchLocation string) {
	for _, resourceFile := range _RESOURCE_FILES {
		filePath := path.Join(_RESOURCE_DIR, resourceFile)
		ok := checkFile(filePath)
		if !ok {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Resource: ", filePath, " not found")
			color.Unset()
			os.Exit(1)
		} else {
			logger.Debug("Copying resource: %s to %s", filePath, _TEMP_DIR_NAME)
			tempPath := path.Join(_TEMP_DIR_NAME, resourceFile)
			err := CopyFile(filePath, tempPath)
			if (err != nil) {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred while copying the resource file: ", filePath, err)
				color.Unset()
			}
		}
	}

	filePath := path.Join(patchLocation, _UPDATE_DESCRIPTOR_FILE_NAME)
	ok := checkFile(filePath)
	if !ok {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Resource: ", filePath, " not found")
		color.Unset()
		os.Exit(1)
	} else {
		logger.Debug("Copying resource: %s to %s", filePath, _TEMP_DIR_NAME)
		tempPath := path.Join(_TEMP_DIR_NAME, _UPDATE_DESCRIPTOR_FILE_NAME)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}

	filePath = path.Join(patchLocation, _README_FILE_NAME)
	ok = checkFile(filePath)
	if !ok {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Readme: ", filePath, " not found")
		color.Unset()
		os.Exit(1)
	} else {
		logger.Debug("Copying readme: %s to %s", filePath, _TEMP_DIR_NAME)
		tempPath := path.Join(_TEMP_DIR_NAME, _README_FILE_NAME)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}

	filePath = path.Join(patchLocation, _NOT_A_CONTRIBUTION_FILE_NAME)
	ok = checkFile(filePath)
	if !ok {
		color.Set(color.FgYellow)
		fmt.Println("\n[WARNING]", _NOT_A_CONTRIBUTION_FILE_NAME, "not found in the patch directory. Make " +
		"sure that the Apache License is in", _LICENSE_FILE_NAME, "file.\n")
		color.Unset()
	} else {
		logger.Debug("Copying NOT_A_CONTRIBUTION: %s to %s", filePath, _TEMP_DIR_NAME)
		tempPath := path.Join(_TEMP_DIR_NAME, _README_FILE_NAME)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}

	filePath = path.Join(patchLocation, _INSTRUCTIONS_FILE_NAME)
	ok = checkFile(filePath)
	if !ok {
		color.Set(color.FgYellow)
		fmt.Print("[WARNING] ", _INSTRUCTIONS_FILE_NAME, " file not found. Do you want to add an " +
		"'instructions.txt' file?[Y/N]: ")
		color.Unset()

		for {
			reader := bufio.NewReader(os.Stdin)
			preference, _ := reader.ReadString('\n')
			if preference[0] == 'y' || preference[0] == 'Y' {
				fmt.Println("Please create an '" + _INSTRUCTIONS_FILE_NAME + "' file in the patch " +
				"directory and run the tool again.")
				os.Exit(0)
			} else if preference[0] == 'n' || preference[0] == 'N' {
				color.Set(color.FgYellow)
				fmt.Println("[WARNING] Skipping creating '" + _INSTRUCTIONS_FILE_NAME + "' file")
				color.Unset()
				break;
			} else {
				fmt.Println("Invalid preference. Enter Y for Yes or N for No.\n")
				fmt.Print("Do you want to add an instructions.txt file?[Y/N]: ")
			}
		}
	} else {
		logger.Debug("Copying instructions: %s to %s", filePath, _TEMP_DIR_NAME)
		tempPath := path.Join(_TEMP_DIR_NAME, _INSTRUCTIONS_FILE_NAME)
		err := CopyFile(filePath, tempPath)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}
	//generateReadMe()
}

//This is used to generate README from the update-descriptor.yaml
func generateReadMe() {

	type readMeData struct {
		Patch_id        string
		Applies_to      string
		Associated_jira string
		Description     string
		Instructions    string
	}

	tempPath := path.Join(_RESOURCE_DIR, _README_FILE_NAME)
	t, err := template.ParseFiles(tempPath)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while reading the", _README_FILE_NAME, "template file:", err)
		color.Unset()
		os.Exit(1)
	}

	tempPath = path.Join(_TEMP_DIR_NAME, _README_FILE_NAME)
	f, err := os.Create(tempPath)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while creating", tempPath, "file:", err)
		color.Unset()
		os.Exit(1)
	}

	associatedJIRAs := ""
	for key, _ := range descriptor.Bug_fixes {
		associatedJIRAs += (_JIRA_URL_PREFIX + key + ", ")
	}
	associatedJIRAs = strings.TrimSuffix(associatedJIRAs, ", ")
	logger.Debug("Associated JIRAs: ", associatedJIRAs)

	data := readMeData{
		Patch_id:_UPDATE_NAME,
		Applies_to:descriptor.Applies_to,
		Associated_jira:associatedJIRAs,
		Description:descriptor.Description,
	}

	err = t.Execute(f, data) //merge template ‘t’ with content of ‘p’
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while processing the README template:", err)
		os.Exit(1)
	}
	//f.Close()
	//err = t.Execute(os.Stdout, data) //merge template ‘t’ with content of ‘p’
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
}

//Check whether the given path contain a zip file
func isAZipFile(path string) bool {
	return strings.HasSuffix(path, ".zip")
}

//This is used to find matches of files/directories in the patch from distribution
func findMatches(patchLocation, distributionLocation string) {

	//Create a new table
	overallViewTable := tablewriter.NewWriter(os.Stdout)
	overallViewTable.SetAlignment(tablewriter.ALIGN_LEFT)
	overallViewTable.SetHeader([]string{"File/Folder", "Copied To"})

	//Delete temp files
	err := os.RemoveAll(_TEMP_DIR_NAME)
	if err != nil {
		if !os.IsNotExist(err) {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Error occurred while deleting temp directory:", err)
			color.Unset()
			os.Exit(1)
		}
	}

	//Create temp directory
	err = os.MkdirAll(_TEMP_DIR_LOCATION, 0777)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while creating temp directory:", err)
		color.Unset()
		os.Exit(1)
	}

	rowCount := 0
	//Find matches for each entry in the patchEntries map
	for patchEntryName, patchEntry := range patchEntries {
		//Find a match in distEntries
		distEntry, ok := distEntries[patchEntryName]
		//If there is a match
		if ok {
			logger.Trace("Match found for: %s", patchEntryName)
			logger.Trace("Location(s) in Dist: %s", distEntry)
			//Get the distribution path. This is later used for trimming
			distPath := getDistPath(distributionLocation)
			logger.Trace("Dist Path used for trimming: %s", distPath)

			//If there are more than 1 match, we need to ask preferred locations from the user
			if len(distEntry.locationMap) > 1 {
				color.Set(color.FgRed)
				fmt.Println("\n'" + patchEntryName + "' was found in multiple locations in the distribution")
				color.Unset()
				//This is used to temporary store all the locations
				locationMap := make(map[string]string)
				//Create the temporary table
				tempTable := tablewriter.NewWriter(os.Stdout)
				tempTable.SetHeader([]string{"index", "Location"})

				//Add data to the table
				index := 1
				for pathInDist, isDirInDist := range distEntry.locationMap {
					for _, isDirInPatch := range patchEntry.locationMap {
						if isDirInDist == isDirInPatch {
							//Add the location to the map. Key is the index
							locationMap[strconv.Itoa(index)] = pathInDist
							logger.Trace("Trimming: %s ; using: %s", pathInDist, distPath)
							tempLoc := strings.TrimPrefix(pathInDist, distPath) + string(os.PathSeparator)
							tempTable.Append([]string{strconv.Itoa(index), strings.Replace(tempLoc, "\\", "/", -1)})
							index++
						}
					}
				}
				logger.Trace("Location Map for Dist: %s", locationMap)

				//Print the temporary table
				tempTable.Render()

				//loop until user enter valid indices or decide to exit
				for {
					fmt.Print("Enter preferred locations separated by commas[Enter 0 to " +
					"cancel and exit]: ")
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
						fmt.Println("0 entered. Cancelling.....")
						os.Exit(0)
					} else {
						//This is used
						selectedPathsList := make([]string, 0)

						//This is used to identify whether the all indices are valid
						isOK := true
						//Iterate through all the selected indices
						for _, selectedIndex := range selectedIndices {
							//Check whether the selected index is in the location map. If it
							// is not in the map, that means an invalid index is entered
							selectedPath, ok := locationMap[selectedIndex]
							//If the index is found
							if ok {
								logger.Trace("Selected index %s was found in map.", selectedIndex)
								logger.Trace("selected path: %s", selectedPath)
								logger.Trace("distPath: %s", distPath)
								logger.Trace("distributionLocation: %s", distributionLocation)
								logger.Trace("Trimming: %s ; using: %s", selectedPath, distributionLocation)
								tempFilePath := strings.TrimPrefix(selectedPath, distributionLocation)
								logger.Trace("tempFilePath: %s", tempFilePath)

								selectedPathsList = append(selectedPathsList, selectedPath)

								src := path.Join(patchLocation, patchEntryName)
								destPath := path.Join(_TEMP_DIR_LOCATION + tempFilePath)
								logger.Trace("destPath: %s", destPath)
								dest := path.Join(destPath, patchEntryName)

								logger.Trace("src 1: %s", src)
								logger.Trace("dest1: %s", dest)

								//If source is a file
								if checkFile(src) {
									logger.Debug("Copying file: %s ; To: %s", src, dest)
									//copy source file to destination
									copyErr := CopyFile(src, dest)
									if copyErr != nil {
										color.Set(color.FgRed)
										fmt.Println("[FAILURE] Error occurred while copying file:", copyErr)
										color.Unset()
										os.Exit(1)
									}
								} else if checkDir(src) {
									//Compare the directories to identify new files
									tempPath := path.Join(selectedPath, patchEntryName)
									compareDir(src, tempPath, patchLocation, distributionLocation)

									//If source is a directory
									logger.Debug("Copying directory: %s ; To: %s", src, dest)
									//copy source directory to destination
									copyErr := CopyDir(src, dest)
									if copyErr != nil {
										color.Set(color.FgRed)
										fmt.Println("[FAILURE] Error occurred while copying " +
										"directory:", copyErr)
										color.Unset()
										os.Exit(1)
									}
								}
							} else {
								//If index is invalid
								color.Set(color.FgRed)
								fmt.Println("[FAILURE] One or more entered indices are invalid. " +
								"Please enter again")
								color.Unset()
								isOK = false
								break
							}
						}
						//If all the indices are valid, generate the table
						if isOK {
							isFirst := true
							for path, isDirInDist := range distEntry.locationMap {
								for _, isDirInPatch := range patchEntry.locationMap {
									if isDirInDist == isDirInPatch {
										locationMap[strconv.Itoa(index)] = path
										temp := filepath.Join(_CARBON_HOME, strings.TrimPrefix(path, distributionLocation)) + string(os.PathSeparator)
										logger.Trace("temp: %s", temp)
										if isFirst {
											found := stringIsInSlice(path, selectedPathsList)
											if found {
												overallViewTable.Append([]string{patchEntryName, strings.Replace(temp, "\\", "/", -1)})
												isFirst = false
											}
										} else {
											found := stringIsInSlice(path, selectedPathsList)
											if found {
												overallViewTable.Append([]string{"", strings.Replace(temp, "\\", "/", -1)})
											}
										}
									}
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
				for pathInDist, isDirInDist := range distEntry.locationMap {
					//Get the location in the patch file
					for pathInPatch, isDirInPatch := range patchEntry.locationMap {
						//Check whether both locations contain same type (files or directories)
						if isDirInDist == isDirInPatch {
							//Add an entry to the table
							logger.Trace("Both locations contain same type.")
							logger.Trace("pathInDist: %s", pathInDist)
							logger.Trace("distPath: %s", distPath)
							logger.Trace("distributionLocation: %s", distributionLocation)

							tempLoc := filepath.Join(_CARBON_HOME, strings.TrimPrefix(pathInDist, distributionLocation)) + string(os.PathSeparator)
							overallViewTable.Append([]string{patchEntryName, strings.Replace(tempLoc, "\\", "/", -1)})

							//Get the path relative to the distribution
							tempFilePath := strings.TrimPrefix(pathInDist, distributionLocation)
							logger.Trace("tempFilePath: %s", tempFilePath)
							//Construct the source location
							src := path.Join(pathInPatch, patchEntryName)
							destPath := path.Join(_TEMP_DIR_LOCATION + tempFilePath)
							logger.Trace("destPath: %s", destPath)
							//Construct the destination location
							dest := path.Join(destPath, patchEntryName)

							logger.Trace("src 2: %s", src)
							logger.Trace("dest2: %s", dest)

							//Create all directories. Otherwise copy will return an error.
							// We cannot copy directories in GO. We have to copy file
							// by file
							err := os.MkdirAll(destPath, 0777)
							if err != nil {
								color.Set(color.FgRed)
								fmt.Println("[FAILURE] Error occurred while creating " +
								"directory", err)
								color.Unset()
								os.Exit(1)
							}

							//If source is a file
							if checkFile(src) {
								logger.Debug("Copying file: %s ; To: %s", src, dest)
								//copy source file to destination
								copyErr := CopyFile(src, dest)
								if copyErr != nil {
									color.Set(color.FgRed)
									fmt.Println("[FAILURE] Error occurred while copying file:", copyErr)
									color.Unset()
									os.Exit(1)
								}
							} else if checkDir(src) {

								tempPath := path.Join(pathInDist, patchEntryName)
								//Compare the directories to identify new files
								compareDir(src, tempPath, patchLocation, distributionLocation)

								//If source is a directory
								logger.Debug("Copying directory: %s ; To: %s", src, dest)
								//copy source directory to destination
								copyErr := CopyDir(src, dest)
								if copyErr != nil {
									color.Set(color.FgRed)
									fmt.Println("[FAILURE] Error occurred while copying " +
									"directory:", copyErr)
									color.Unset()
									os.Exit(1)
								}
							}
						} else {
							//If file types are different(if one is a file and one is a
							// directory), show a warning message
							color.Set(color.FgYellow)
							fmt.Println("\n[WARNING] Following locations contain", patchEntryName, "but types are different")
							fmt.Println(" - ", pathInDist)
							fmt.Println(" - ", pathInPatch)
							fmt.Println()
							color.Unset()
							typePostfix := " (file)"
							if isDirInPatch {
								typePostfix = " (dir)"
							}
							overallViewTable.Append([]string{patchEntryName + typePostfix, " - "})
						}
					}
				}
			}
		} else {
			//If there is no match
			color.Set(color.FgYellow)
			fmt.Println("\n[WARNING] No match found for '" + patchEntryName + "'\n")

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
					copyPath, _ := reader.ReadString('\n')
					logger.Trace("copyPath: %s", copyPath)
					//Remove the new line at the end
					copyPath = path.Join(distributionLocation, strings.TrimSpace(copyPath))
					logger.Trace("copyPath2: %s", copyPath)

					if !checkDir(copyPath) {

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
								overallViewTable.Append([]string{patchEntryName, " - "})
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

					tempFilePath := strings.TrimPrefix(copyPath, distributionLocation)
					destPath := path.Join(_TEMP_DIR_LOCATION, tempFilePath)
					logger.Debug("destPath: %s", destPath)
					//Construct the destination location
					dest := path.Join(destPath, patchEntryName)

					err := os.MkdirAll(destPath, 0777)
					if err != nil {
						color.Set(color.FgRed)
						fmt.Println("[FAILURE] Error occurred while creating " +
						"directory", err)
						color.Unset()
						os.Exit(1)
					}

					logger.Debug("Entered location is a directory. Copying ...")
					fileLocation := filepath.Join(patchLocation, patchEntryName)
					tempDistPath := path.Join(_CARBON_HOME, tempFilePath) + string(os.PathSeparator)
					if checkFile(fileLocation) {
						logger.Trace("File found: %s", fileLocation)
						//copyPath = path.Join(copyPath, patchEntryName)
						logger.Debug("Copying file: %s ; To: %s", fileLocation, dest)
						CopyFile(fileLocation, dest)
						overallViewTable.Append([]string{patchEntryName, tempDistPath})
					} else if checkDir(fileLocation) {
						logger.Trace("dir found: %s", fileLocation)
						//copyPath = path.Join(copyPath, patchEntryName)
						logger.Debug("Copying file: %s", fileLocation, "; To:", dest)
						CopyDir(fileLocation, dest)
						overallViewTable.Append([]string{patchEntryName, tempDistPath})
					} else {
						logger.Debug("Location not valid:", fileLocation)
					}

					break
				}

			} else if enteredPreferences[0] == 'N' || enteredPreferences[0] == 'n' {
				logger.Debug("Not copying file.")
				logger.Trace("Location(s) in Patch: %s", patchEntry)
				overallViewTable.Append([]string{patchEntryName, " - "})
			} else {
				fmt.Println("Invalid preference. Try again.\n")
			}
			color.Unset()
		}
		rowCount++
		if rowCount < len(patchEntries) {
			//todo: add separator
		}
	}
	//Print summary
	fmt.Println("\n# Summary\n")
	overallViewTable.Render()
}

//This will compare and print warnings for new files when copying directories from patch to temp directory
func compareDir(pathInPatch, pathInDist, patchLoc, distLoc string) {

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
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Error occurred while traversing pathInPatch: ", err)
			color.Unset()
			os.Exit(1)
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
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while traversing pathInPatch:", err)
		color.Unset()
		os.Exit(1)
	}

	//Walk the directory in the distribution
	err = filepath.Walk(pathInDist, func(path string, fileInfo os.FileInfo, err error) error {
		logger.Trace("Walking: %s", path)
		if err != nil {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Error occurred while traversing pathInDist: ", err)
			color.Unset()
			os.Exit(1)
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
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while traversing pathInDist:", err)
		color.Unset()
		os.Exit(1)
	}

	//Look for match for each file in patch location
	for path, _ := range filesInPatch {
		//Check whether distribution has a match
		_, found := filesInDist[path]
		if found {
			//If a match found, logger it
			logger.Debug("'%s' found in the distribution.", path)
		} else {
			//If no match is found, show warning message and how to add it to the update -descriptor.yaml
			color.Set(color.FgYellow)
			fmt.Println("[WARNING] '" + strings.Replace(strings.TrimPrefix(path, string(os.PathSeparator)), "\\", "/", -1) + "' not found in '" +
			strings.TrimPrefix(pathInDist + string(os.PathSeparator), distLoc) + "'")
			tempDistFilePath := strings.TrimPrefix(pathInDist, distLoc)

			tempPath := strings.Replace(tempDistFilePath + path, "\\", "/", -1)

			descriptor.File_changes.Added_files = append(descriptor.File_changes.Added_files, tempPath)

			fmt.Println("[INFO] '" + tempPath + "' path was added to 'added_files' " + "section in '" + _UPDATE_DESCRIPTOR_FILE_NAME + "'\n")

			color.Unset()
		}
	}
}

//Get the path of the distribution location. This is used to trim the prefixes
func getDistPath(distributionLoc string) string {
	index := strings.LastIndex(distributionLoc, string(os.PathSeparator))
	if index != -1 {
		return distributionLoc[:index]
	} else {
		return distributionLoc
	}
}

//Check whether the given string is in the given slice
func stringIsInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Copies file source to destination dest.
func CopyFile(source string, dest string) (err error) {
	sf, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err == nil {
		si, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, si.Mode())
			fmt.Println("Error occurred while copying:", err)
			os.Exit(1)
		}

	}
	return
}

//Recursively copies a directory tree, attempting to preserve permissions.
//Source directory must exist, destination directory must *not* exist.
func CopyDir(source string, dest string) (err error) {
	// get properties of source dir
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return &CustomError{"Source is not a directory"}
	}
	// ensure dest dir does not already exist
	_, err = os.Open(dest)
	if !os.IsNotExist(err) {
		return &CustomError{"Destination already exists"}
	}
	// create dest dir
	err = os.MkdirAll(dest, fi.Mode())
	if err != nil {
		return err
	}
	entries, err := ioutil.ReadDir(source)
	for _, entry := range entries {
		sfp := source + "/" + entry.Name()
		dfp := dest + "/" + entry.Name()
		if entry.IsDir() {
			err = CopyDir(sfp, dfp)
			if err != nil {
				fmt.Println("Error occurred while copying:", err)
				os.Exit(1)
			}
		} else {
			// perform copy
			err = CopyFile(sfp, dfp)
			if err != nil {
				fmt.Println("Error occurred while copying:", err)
				os.Exit(1)
			}
		}
	}
	return
}

//A struct for returning custom error messages
type CustomError struct {
	What string
}

//Returns the error message defined in What as a string
func (e *CustomError) Error() string {
	return e.What
}

//Check whether the given location points to a directory
func checkDir(location string) bool {
	logger.Trace("Checking Location: %s", location)
	locationInfo, err := os.Stat(location)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if locationInfo.IsDir() {
		return true
	}
	return false
}

//Check whether the given location points to a file
func checkFile(location string) bool {
	logger.Trace("Checking location: %s", location)
	locationInfo, err := os.Stat(location)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if locationInfo.IsDir() {
		return false
	}
	return true
}

//Traverse the given path and add entries to the given map
func traverse(path string, entryMap map[string]entry, isDist bool) {
	//Get all the files/directories
	files, _ := ioutil.ReadDir(path)
	//Iterate through all files
	for _, file := range files {
		//update-descriptor, README, instructions files might be in the patch location. We don't need to find
		// matches for them
		if file.Name() != _UPDATE_DESCRIPTOR_FILE_NAME && file.Name() != _README_FILE_NAME && file.Name() != _INSTRUCTIONS_FILE_NAME {
			logger.Trace("Checking entry: %s ; path: %s", file.Name(), path)
			//Check whether the filename is already in the map
			_, ok := entryMap[file.Name()]
			if (ok) {
				//If the file is already in the map, we only need to add a new entry
				entry := entryMap[file.Name()]
				entry.add(path)
			} else {
				//This is to identify whether the location contain a file or a directory
				isDir := false
				if file.IsDir() {
					isDir = true
				}
				//Add a new entry
				entryMap[file.Name()] = entry{
					map[string]bool{
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
func createUpdateZip() {
	logger.Debug("Creating patch zip file: %s.zip", _UPDATE_NAME)
	//Create the new zip file
	outFile, err := os.Create(_UPDATE_NAME + ".zip")
	if err != nil {
		color.Set(color.FgRed)
		logger.Debug("[FAILURE] Error occurred while creating the zip file: %s", err)
		color.Unset()
	}
	defer outFile.Close()

	//Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)

	//Start traversing
	err = filepath.Walk(_TEMP_DIR_NAME, func(path string, fileInfo os.FileInfo, err error) error {
		logger.Trace("Walking: %s", path)
		if err != nil {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Error occurred while traversing the temp files: ", err)
			color.Unset()
			os.Exit(1)
		}
		//We only want to add the files to the zip. Corresponding directories will be auto created
		if !fileInfo.IsDir() {
			//We need to create a header from fileInfo. Otherwise, the file creation time will be set as
			// the start time in go (1979)
			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred while creating the zip file: ", err)
				color.Unset()
				os.Exit(1)
			}

			//Construct the file path
			tempHeaderName := filepath.Join(_UPDATE_NAME, strings.TrimPrefix(path, _TEMP_DIR_NAME))

			//Critical -
			//If the paths in zip file have \ separators, they will not shown correctly on Ubuntu. But if
			// we have / path separators, the file paths will be correctly shown in both Windows and
			// Ubuntu. So we need to replace all \ with / before creating the zip.
			header.Name = strings.Replace(tempHeaderName, "\\", "/", -1)
			logger.Trace("header.Name: %s", header.Name)
			//Create a Writer using the header
			fileWriter, err := zipWriter.CreateHeader(header)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred while creating the zip file: ", err)
				color.Unset()
				os.Exit(1)
			}

			//Open the file for reading
			file, err := os.Open(path)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred when file was open to write to zip:", err)
				color.Unset()
				os.Exit(1)
			}
			//Convert the file to byte array
			data, err := ioutil.ReadAll(file)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred when getting the byte array from the file", err)
				color.Unset()
				os.Exit(1)
			}
			//Write the bytes to zip file
			_, err = fileWriter.Write(data)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE] Error occurred when writing the byte array to the zip file", err)
				color.Unset()
				os.Exit(1)
			}
		}
		return nil
	})
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while traversing the temp location:", err)
		color.Unset()
		os.Exit(1)
	}

	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred when closing the zip writer", err)
		color.Unset()
		os.Exit(1)
	}

	logger.Trace("Directory Walk completed successfully.")
	color.Set(color.FgGreen)
	fmt.Println("\n[INFO] Update file '" + _UPDATE_NAME + ".zip' successfully created\n\n")
	color.Unset()
}

//This function unzips a zip file at given location
func unzipAndReadDistribution(zipLocation string, logsEnabled bool) {
	logger.Debug("Unzipping started.")
	//Get the location of the zip file. This is later used to create the full path of a file
	index := strings.LastIndex(zipLocation, string(os.PathSeparator))
	distLocation := zipLocation[:index]
	logger.Debug("distLocation: %s", distLocation)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while reading zip:", err)
		color.Unset()
		os.Exit(1)
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
			fmt.Fprintf(writer, "Extracting and reading files from distribution zip: (%d/%d)\n", filesRead,
				totalFiles)
			time.Sleep(time.Millisecond * 2)
		}
		logger.Trace("Checking file: %s", file.Name)

		//Start constructing the full path
		fullPath := file.Name
		logger.Trace("fullPath1: %s", file.Name)

		// Open the file inside the zip archive
		// like a normal file
		zippedFile, err := file.Open()
		if err != nil {
			color.Set(color.FgRed)
			fmt.Println("Error occuured while opening the file:", file, "; Error:", err)
			color.Unset()
			os.Exit(1)
		}

		// Specify what the extracted file name should be.
		// You can specify a full path or a prefix
		// to move it to a different directory.
		// In this case, we will extract the file from
		// the zip to a file of the same name.
		extractionPath := filepath.Join(
			targetDir,
			file.Name,
		)
		// Extract the item (or create directory)
		if file.FileInfo().IsDir() {
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
			logger.Trace("fullPath2: %s", fullPath)
		} else {
			// Extract regular file since not a directory
			//logger.Debug("Extracting file:", file.Name)

			// Open an output file for writing
			outputFile, err := os.OpenFile(
				extractionPath,
				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				file.Mode(),
			)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("Error occuured while opening the file:", outputFile, "; Error:", err)
				color.Unset()
				os.Exit(1)
			}
			if outputFile != nil {
				// "Extract" the file by copying zipped file
				// contents to the output file
				_, err = io.Copy(outputFile, zippedFile)
				outputFile.Close()

				if err != nil {
					color.Set(color.FgRed)
					fmt.Println("Error occuured while opening the file:", file, "; Error:", err)
					color.Unset()
					os.Exit(1)
				}
			}

			// We only need the location of the file. fullPath contains the file name too. We
			// need to trim and remove the file name.

			//string(os.PathSeparator) removed because it does not work properly in windows
			logger.Trace("Trimming: %s ; using: /%s", fullPath, file.FileInfo().Name())
			fullPath = strings.TrimSuffix(fullPath, "/" + file.FileInfo().Name())
			logger.Trace("fullPath3: %s", fullPath)
		}

		// Add the distribution location so that the full path will look like it points to locations of the
		// extracted zip
		fullPath = path.Join(distLocation, fullPath)
		logger.Trace("FileName: %s ; fullPath: %s", file.FileInfo().Name(), fullPath)

		//Add the entries to the distEntries map

		//Check whether the filename is already in the map
		_, ok := distEntries[file.FileInfo().Name()]
		if (ok) {
			//If the file is already in the map, we only need to add a new entry
			entry := distEntries[file.FileInfo().Name()]
			entry.add(fullPath)
		} else {
			//This is to identify whether the location contain a file or a directory
			isDir := false
			if file.FileInfo().IsDir() {
				isDir = true
			}
			//Add a new entry
			distEntries[file.FileInfo().Name()] = entry{
				map[string]bool{
					fullPath: isDir,
				},
			}
		}

		zippedFile.Close()
	}
	writer.Stop()
	logger.Debug("Unzipping finished")
	logger.Debug("Extracted file count: ", filesRead)
	if totalFiles == filesRead {
		logger.Debug("All files extracted")
	} else {
		logger.Debug("All files not extracted")
		color.Set(color.FgRed)
		fmt.Println("All files were not extracted. Total files: %s ; Extracted: %s", totalFiles, filesRead)
		color.Unset()
		os.Exit(1)
	}
}
