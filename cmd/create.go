package cmd

import (
	"os"
	"archive/zip"
	"path/filepath"
	"io"
	"strings"
	"log"
	"io/ioutil"
	"fmt"
	"github.com/fatih/color"
	"github.com/apcera/termtables"
	"time"
	"github.com/gosuri/uilive"
	"bufio"
	"sort"
	"strconv"
	"gopkg.in/yaml.v2"
	"text/template"
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

	//Resourse directory which contains README,LICENCE and NOT_A_CONTRIBUTION files
	_RESOURCE_DIR = "res"
	//Temporary dirctory to copy files before creating the new zip
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
	//This contains the resource files that needs to be copied to the update zip
	_RESOURCE_FILES = []string{_LICENSE_FILE_NAME, _NOT_A_CONTRIBUTION_FILE_NAME}

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
)

//Main entry point to create the new patch
func Create(patchLocation, distributionLocation string, logsEnabled bool) {
	//Check whether logs are enabled or not. If logs are not enabled, discard the logs
	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("create command called")

	//Check whether the patch location has a / or \ character at the end and remove if it has a / or \ character
	if strings.HasSuffix(patchLocation, string(os.PathSeparator)) {
		patchLocation = strings.TrimSuffix(patchLocation, string(os.PathSeparator))
	}
	log.Println("Patch Loc: " + patchLocation)

	//Check whether the given patch location exists. It should be a directory.
	patchLocationExists := checkDir(patchLocation)
	if patchLocationExists {
		log.Println("Patch location exists.")
	} else {
		color.Set(color.FgRed)
		fmt.Println("Patch location does not exist. Enter a valid directory.")
		color.Unset()
		os.Exit(1)
	}
	//Check whether the given distribution is a zip file.
	if (isAZipFile(distributionLocation)) {
		//Check whether the given distribution zip exists.
		zipFileExists := checkFile(distributionLocation)
		if zipFileExists {
			log.Println("Distribution location exists.")
		} else {
			color.Set(color.FgRed)
			fmt.Println("Distribution zip does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	} else {
		//Check whether the given distribution directory exists.
		distributionLocationExists := checkDir(distributionLocation)
		if distributionLocationExists {
			log.Println("Distribution location exists.")
		} else {
			color.Set(color.FgRed)
			fmt.Println("Distribution location does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	}

	log.Println("Distribution Loc: " + distributionLocation)

	//This will have the update-descriptor.yaml file location
	descriptorLocation := patchLocation + string(os.PathSeparator) + _UPDATE_DESCRIPTOR_FILE_NAME
	log.Println("Descriptor Location: ", descriptorLocation)

	//Check whether the update-descriptor.yaml file exists
	descriptorExists := checkFile(descriptorLocation);
	log.Println("Descriptor Exists: ", descriptorExists)
	if descriptorExists {
		readDescriptor(descriptorLocation)
	} else {
		//readPatchInfo()
		color.Set(color.FgRed)
		fmt.Println(_UPDATE_DESCRIPTOR_FILE_NAME + " not found at " + descriptorLocation)
		color.Unset()
		os.Exit(1)
	}

	patchEntries = make(map[string]entry)
	distEntries = make(map[string]entry)

	//var unzipLocation string
	//If the distribution is a zip file, we need to unzip it
	if isAZipFile(distributionLocation) {
		log.Println("Distribution location is a zip file. Reading zip file...")

		readDistributionZip(distributionLocation, logsEnabled)

		distributionName := strings.TrimSuffix(distributionLocation, ".zip")
		log.Println("Unzip Location: " + distributionName)
		//unzipSuccessful, err := unzip(distributionLocation)
		//
		//if err != nil {
		//	color.Set(color.FgRed)
		//	fmt.Println("Error occurred while extracting zip file")
		//	color.Unset()
		//}
		//if unzipSuccessful {
		//	log.Println("File successfully unzipped...")
		log.Println("Traversing patch location")
		traverse(patchLocation, patchEntries, false)
		log.Println("Traversing patch location finished")
		//
		//	distLocationExists := checkDir(unzipLocation)
		//	if distLocationExists {
		//		log.Println("Distribution location(unzipped locations) exists. Reading files: ",
		//			unzipLocation)
		//	} else {
		//		color.Set(color.FgRed)
		//		fmt.Println("Distribution location(unzipped location) does not exist: ", unzipLocation)
		//		color.Unset()
		//		os.Exit(1)
		//	}
		//
		//	log.Println("Traversing unzip location")
		//	traverse(unzipLocation, distEntries, true)
		//	log.Println("Traversing unzip location finished")
		log.Println("Finding matches")
		findMatches(patchLocation, distributionName)
		log.Println("Finding matches finished")
		log.Println("Copying resource files")
		copyResourceFiles(patchLocation)
		log.Println("Copying resource files finished")
		log.Println("Creating zip file")
		createPatchZip()
		log.Println("Creating zip file finished")
		//} else {
		//	color.Set(color.FgRed)
		//	fmt.Println("Error occurred while unzipping")
		//	color.Unset()
		//}
	} else {
		log.Println("Distribution location is not a zip file")
		log.Println("Traversing patch location")
		traverse(patchLocation, patchEntries, false)
		log.Println("Traversing patch location finished")
		log.Println("Patch Entries: ", patchEntries)
		log.Println("Traversing distribution location")
		traverse(distributionLocation, distEntries, true)
		log.Println("Traversing distribution location finished")
		log.Println("Finding matches")
		findMatches(patchLocation, distributionLocation)
		log.Println("Finding matches finished")
		log.Println("Copying resource files")
		copyResourceFiles(patchLocation)
		log.Println("Copying resource files finished")
		log.Println("Creating zip file")
		createPatchZip()
		log.Println("Creating zip file finished")
	}
}

func readDescriptor(path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while reading the descriptor: ", err)
		color.Unset()
	}
	descriptor = update_descriptor{}

	err = yaml.Unmarshal(yamlFile, &descriptor)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while unmarshalling the yaml:", err)
		color.Unset()
	}

	log.Println("----------------------------------------------------------------")
	log.Println("update_number:", descriptor.Update_number)
	log.Println("kernel_version:", descriptor.Kernel_version)
	log.Println("platform_version:", descriptor.Platform_version)
	log.Println("applies_to: ", descriptor.Applies_to)
	log.Println("bug_fixes: ", descriptor.Bug_fixes)
	log.Println("file_changes: ", descriptor.File_changes)
	log.Println("----------------------------------------------------------------")

	//for key, value := range (descriptor.Bug_fixes) {
	//	fmt.Println("\t", key, ":", value)
	//}
	log.Println("description: " + descriptor.Description)

	if len(descriptor.Update_number) == 0 {
		color.Set(color.FgRed)
		fmt.Println("update_number", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Kernel_version) == 0 {
		color.Set(color.FgRed)
		fmt.Println("kernel_version", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Platform_version) == 0 {
		color.Set(color.FgRed)
		fmt.Println("platform_version", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Applies_to) == 0 {
		color.Set(color.FgRed)
		fmt.Println("applies_to", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Bug_fixes) == 0 {
		color.Set(color.FgRed)
		fmt.Println("bug_fixes", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}
	if len(descriptor.Description) == 0 {
		color.Set(color.FgRed)
		fmt.Println("description", " field not found in ", _UPDATE_DESCRIPTOR_FILE_NAME)
		color.Unset()
		os.Exit(1)
	}

	_KERNEL_VERSION = descriptor.Kernel_version
	log.Println("kernel version set to: ", _KERNEL_VERSION)

	_UPDATE_NUMBER = descriptor.Update_number
	log.Println("patch number set to: ", _UPDATE_NUMBER)

	_UPDATE_NAME = _UPDATE_NAME_PREFIX + "-" + _KERNEL_VERSION + "-" + _UPDATE_NUMBER
	log.Println("Patch Name: " + _UPDATE_NAME)
}

//func readPatchInfo() {
//	reader := bufio.NewReader(os.Stdin)
//	fmt.Print("Enter kernel version: ")
//	_KERNEL_VERSION, _ = reader.ReadString('\n')
//	_KERNEL_VERSION = strings.TrimSuffix(_KERNEL_VERSION, "\n")
//	log.Println("Entered kernel version: ", _KERNEL_VERSION)
//
//	fmt.Print("Enter patch number: ")
//	_PATCH_NUMBER, _ = reader.ReadString('\n')
//	_PATCH_NUMBER = strings.TrimSuffix(_PATCH_NUMBER, "\n")
//	log.Println("Entered patch number: ", _PATCH_NUMBER)
//
//	_PATCH_NAME = _PATCH_NAME_PREFIX + "-" + _KERNEL_VERSION + "-" + _PATCH_NUMBER
//	log.Println("Patch Name: " + _PATCH_NAME)
//}

//This method copies resource files to the
func copyResourceFiles(patchLocation string) {
	for _, resourceFile := range _RESOURCE_FILES {
		filePath := _RESOURCE_DIR + string(os.PathSeparator) + resourceFile
		ok := checkFile(filePath)
		if !ok {
			color.Set(color.FgRed)
			fmt.Println("Resource: ", filePath, " not found")
			color.Unset()
		} else {
			log.Println("Copying resource: ", filePath, " to: " + _TEMP_DIR_NAME)
			err := CopyFile(filePath, _TEMP_DIR_NAME + string(os.PathSeparator) + resourceFile)
			if (err != nil) {
				color.Set(color.FgRed)
				fmt.Println("Error occurred while copying the resource file: ", filePath, err)
				color.Unset()
			}
		}
	}
	filePath := patchLocation + string(os.PathSeparator) + _UPDATE_DESCRIPTOR_FILE_NAME
	ok := checkFile(filePath)
	if !ok {
		color.Set(color.FgRed)
		fmt.Println("Resource: ", filePath, " not found")
		color.Unset()
	} else {
		log.Println("Copying resource: ", filePath, " to: " + _TEMP_DIR_NAME)
		err := CopyFile(filePath, _TEMP_DIR_NAME + string(os.PathSeparator) + _UPDATE_DESCRIPTOR_FILE_NAME)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}

	filePath = patchLocation + string(os.PathSeparator) + _README_FILE_NAME
	ok = checkFile(filePath)
	if !ok {
		color.Set(color.FgRed)
		fmt.Println("Readme: ", filePath, " not found")
		color.Unset()
	} else {
		log.Println("Copying readme: ", filePath, " to: " + _TEMP_DIR_NAME)
		err := CopyFile(filePath, _TEMP_DIR_NAME + string(os.PathSeparator) + _README_FILE_NAME)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}

	filePath = patchLocation + string(os.PathSeparator) + _INSTRUCTIONS_FILE_NAME
	ok = checkFile(filePath)
	if !ok {
		color.Set(color.FgRed)
		fmt.Print(_INSTRUCTIONS_FILE_NAME, " file not found. Do you want to add a instructions.txt file?[Y/N]: ")
		color.Unset()

		for {
			reader := bufio.NewReader(os.Stdin)
			preference, _ := reader.ReadString('\n')
			if preference[0] == 'y' || preference[0] == 'Y' {
				fmt.Println("Please create an '" + _INSTRUCTIONS_FILE_NAME + "' file in the patch " +
				"directory and run the tool again.")
				os.Exit(0)
			} else if preference[0] == 'n' || preference[0] == 'N' {
				color.Set(color.FgGreen)
				fmt.Println("Skipping copying/creating '" + _INSTRUCTIONS_FILE_NAME + "' file")
				color.Unset()
				break;
			} else {
				fmt.Println("Invalid preference. Enter Y for Yes or N for No\n")
				fmt.Print("Do you want to add an instructions.txt file?[Y/N]: ")
			}
		}
	} else {
		log.Println("Copying instructions: ", filePath, " to: " + _TEMP_DIR_NAME)
		err := CopyFile(filePath, _TEMP_DIR_NAME + string(os.PathSeparator) + _INSTRUCTIONS_FILE_NAME)
		if (err != nil) {
			color.Set(color.FgRed)
			fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			color.Unset()
		}
	}
	//generateReadMe()
}

func generateReadMe() {

	type readMeData struct {
		Patch_id        string
		Applies_to      string
		Associated_jira string
		Description     string
		Instructions    string
	}

	t, err := template.ParseFiles(_RESOURCE_DIR + string(os.PathSeparator) + _README_FILE_NAME)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	f, err := os.Create(_TEMP_DIR_NAME + string(os.PathSeparator) + _README_FILE_NAME)
	if err != nil {
		log.Println("create file: ", err)
		os.Exit(1)
	}

	associatedJIRAs := ""
	for key, _ := range descriptor.Bug_fixes {
		associatedJIRAs += (_JIRA_URL_PREFIX + key + ", ")
	}
	associatedJIRAs = strings.TrimSuffix(associatedJIRAs, ", ")
	log.Println("Associated JIRAs: ", associatedJIRAs)

	data := readMeData{
		Patch_id:_UPDATE_NAME,
		Applies_to:descriptor.Applies_to,
		Associated_jira:associatedJIRAs,
		Description:descriptor.Description,
	}

	err = t.Execute(f, data) //merge template ‘t’ with content of ‘p’
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println(err)
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

func findMatches(patchLocation, distributionLocation string) {

	//fmt.Println("Matching files started ------------------------------------------------------------------------")
	termtables.EnableUTF8()
	overallViewTable := termtables.CreateTable()
	//overallViewTable.AddTitle("Summary")
	overallViewTable.AddHeaders("File/Folder", "Copied To")

	err := os.RemoveAll(_TEMP_DIR_NAME)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	err = os.MkdirAll(_TEMP_DIR_LOCATION, 0777)

	if err != nil {
		log.Fatal(err)
	}

	rowCount := 0
	for patchEntryString, patchEntry := range patchEntries {
		//todo check for underscore and dash when matching in plugins folder
		distEntry, ok := distEntries[patchEntryString]
		if ok {
			log.Println("Match found for ", patchEntryString)
			log.Println("Location(s) in Dist: ", distEntry)

			distPath := getDistPath(distributionLocation)
			log.Println("Dist Path used for trimming: ", distPath)

			if len(distEntry.locationMap) > 1 {
				color.Set(color.FgRed)
				fmt.Println("\"" + patchEntryString +
				"\" was found in multiple locations in the distribution")
				color.Unset()
				locationMap := make(map[string]string)

				tempTable := termtables.CreateTable()

				tempTable.AddHeaders("index", "Location(s) of similar" +
				" file(s)/folder(s) in the distribution")

				index := 1
				//isFirst := true
				for pathInDist, isDirInDist := range distEntry.locationMap {

					for _, isDirInPatch := range patchEntry.locationMap {

						if isDirInDist == isDirInPatch {

							locationMap[strconv.Itoa(index)] = pathInDist

							//if isFirst {
							//	overallViewTable.AddRow(patchEntryString, path)
							//	isFirst = false
							//} else {
							//	overallViewTable.AddRow("", path)
							//}
							log.Println("Trimming: ", pathInDist, "; using: ", distPath)
							tempTable.AddRow(index, strings.TrimPrefix(pathInDist, distPath))
							index++
						}
					}
				}

				log.Println("Location Map for Dist: ", locationMap)

				tempTable.SetAlign(termtables.AlignCenter, 1)
				fmt.Println(tempTable.Render())

				//loop until user enter valid indices or decide to exit
				for {
					fmt.Println("Enter preferred locations separated by commas[Enter 0 to " +
					"cancel]: ")
					//fmt.Println(locationList)
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Preferred Locations: ")
					enteredPreferences, _ := reader.ReadString('\n')

					enteredPreferences = strings.TrimSuffix(enteredPreferences, "\n")
					selectedIndices := strings.Split(enteredPreferences, ",");

					sort.Strings(selectedIndices)

					if selectedIndices[0] == "0" {
						fmt.Println("0 entered. Cancelling.....")
						os.Exit(0)
					} else {

						selectedPathsList := make([]string, 0)
						//todo check for valid indices
						log.Println("Sorted indices: ", selectedIndices)
						isOK := true
						for _, selectedIndex := range selectedIndices {
							selectedPath, ok := locationMap[selectedIndex]
							if ok {
								log.Println("Selected index ", selectedIndex, " was " +
								"found in map")
								log.Println("selected path: " + selectedPath)


								//delete(distEntry.locationMap,selectedPath)

								tempFilePath := strings.TrimPrefix(selectedPath, distributionLocation)
								selectedPathsList = append(selectedPathsList, selectedPath)

								src := patchLocation + string(os.PathSeparator) +
								patchEntryString
								destPath := _TEMP_DIR_LOCATION + tempFilePath + string(os.PathSeparator)
								dest := destPath + patchEntryString

								log.Println("src : ", src)
								log.Println("dest: ", dest)

								CopyDir(src, dest) //todo multiple files with same name

								//if checkFile(src) {
								//	log.Println("Copying file: ", src, " ; To:", dest)
								//	copyErr := CopyFile(src, dest)
								//	if copyErr != nil {
								//		fmt.Println("Error occurred while copying file:",
								//			copyErr)
								//		os.Exit(1)
								//	}
								//} else if checkDir(src) {
								//	log.Println("Copying directory: ", src, " ; To:", dest)
								//	copyErr := CopyDir(src, dest)
								//	if copyErr != nil {
								//		fmt.Println("Error occurred while copying " +
								//		"directory:", copyErr)
								//		os.Exit(1)
								//	}
								//}

							} else {
								color.Set(color.FgRed)
								fmt.Println("One or more entered indices are invalid. " +
								"Please enter again")
								color.Unset()
								isOK = false
								break
							}
						}
						if isOK {
							isFirst := true
							for path, isDirInDist := range distEntry.locationMap {

								for _, isDirInPatch := range patchEntry.locationMap {

									if isDirInDist == isDirInPatch {

										locationMap[strconv.Itoa(index)] = path

										if isFirst {
											found := stringIsInSlice(path, selectedPathsList)
											if found {
												overallViewTable.AddRow(patchEntryString, strings.TrimPrefix(path, distPath))
												isFirst = false
											}
										} else {
											found := stringIsInSlice(path, selectedPathsList)
											if found {
												overallViewTable.AddRow("", strings.TrimPrefix(path, distPath))
											}
										}
									}
								}
							}
							break
						}
					}
				}
			} else {
				for path, isDirInDist := range distEntry.locationMap {

					for pathInPatch, isDirInPatch := range patchEntry.locationMap {

						if isDirInDist == isDirInPatch {
							log.Println("Both locations contain same type")
							overallViewTable.AddRow(patchEntryString, strings.TrimPrefix(path, distPath))

							tempFilePath := strings.TrimPrefix(path, distributionLocation)

							src := pathInPatch + string(os.PathSeparator) + patchEntryString
							destPath := _TEMP_DIR_LOCATION + tempFilePath + string(os.PathSeparator)
							dest := destPath + patchEntryString

							log.Println("src : ", src)
							log.Println("dest: ", dest)

							err := os.MkdirAll(destPath, 0777)

							//newFile, err := os.Create(dest)
							if err != nil {
								color.Set(color.FgRed)
								fmt.Println("Error occurred while creating " +
								"directory", err)
								os.Exit(1)
							}
							//newFile.Close()

							if checkFile(src) {
								log.Println("Copying file: ", src, " ; To:", dest)
								copyErr := CopyFile(src, dest)
								if copyErr != nil {
									color.Set(color.FgRed)
									fmt.Println("Error occurred while copying file:",
										copyErr)
									os.Exit(1)
								}
							} else if checkDir(src) {
								log.Println("Copying directory: ", src, " ; To:", dest)
								copyErr := CopyDir(src, dest)
								if copyErr != nil {
									color.Set(color.FgRed)
									fmt.Println("Error occurred while copying " +
									"directory:", copyErr)
									os.Exit(1)
								}
							}

						} else {

							fmt.Println(color.RedString("\nFollowing locations contain"),
								patchEntryString,
								color.RedString("but types are different"))
							color.Set(color.FgRed)
							fmt.Println(" - ", path)
							fmt.Println(" - ", pathInPatch)
							fmt.Println()
							color.Unset()

							typePostfix := " (file)"
							if isDirInPatch {
								typePostfix = " (dir)"
							}
							overallViewTable.AddRow(patchEntryString + typePostfix, " - ")
						}
					}

				}
			}
		} else {
			color.Set(color.FgRed)
			fmt.Println("\nNo match found for ", patchEntryString, "\n")
			color.Unset()
			log.Println("Location(s) in Patch: ", patchEntry)
			overallViewTable.AddRow(patchEntryString, " - ")
		}
		log.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
		rowCount++
		if rowCount < len(patchEntries) {
			overallViewTable.AddSeparator()
		}
	}
	log.Println("Matching files ended ------------------------------------------------------------------------")
	color.Set(color.FgGreen)
	fmt.Println("# Summary\n")
	fmt.Println(overallViewTable.Render())
	defer color.Unset()
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
		}

	}
	return
}

// Recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
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
				log.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sfp, dfp)
			if err != nil {
				log.Println(err)
			}
		}
	}
	return
}

// A struct for returning custom error messages
type CustomError struct {
	What string
}

// Returns the error message defined in What as a string
func (e *CustomError) Error() string {
	return e.What
}

//Check whether the given path points to a directory
func checkDir(location string) bool {
	log.Println("Checking Location: " + location)
	locationInfo, err := os.Stat(location)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if !locationInfo.IsDir() {
		return false
	}
	return true
}

//Check whether the given path points to a file
func checkFile(path string) bool {
	log.Println("Checking path: " + path)
	locationInfo, err := os.Stat(path)
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
	//log.Println("Root: " + path)
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {
		if file.Name() != _UPDATE_DESCRIPTOR_FILE_NAME && file.Name() != _README_FILE_NAME && file.Name() != _INSTRUCTIONS_FILE_NAME {
			log.Println("Checking entry:", file.Name(), " ; path:", path)

			_, ok := entryMap[file.Name()]
			if (ok) {
				entry := entryMap[file.Name()]
				//log.Println("ENTRY: ", &entry.locations[0])
				entry.add(path)
				//entryMap[f.Name()] = entry
			} else {
				isDir := false
				if file.IsDir() {
					isDir = true
				}
				entryMap[file.Name()] = entry{
					map[string]bool{
						path: isDir,
					},
				}
			}
			if file.IsDir() && isDist {
				//log.Println("Is a dir: " + path + string(os.PathSeparator) + f.Name())
				traverse(path + string(os.PathSeparator) + file.Name(), entryMap, isDist)
			}
		}
	}
}

//This function creates the patch zip file
func createPatchZip() {
	log.Println("Creating patch zip file: ", _UPDATE_NAME + ".zip")
	outFile, err := os.Create(_UPDATE_NAME + ".zip")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	// Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)

	err = filepath.Walk(_TEMP_DIR_NAME, func(path string, fileInfo os.FileInfo, err error) error {

		log.Println("Walking: ", path)

		if !fileInfo.IsDir() {
			// Create and write files to the archive, which in turn
			// are getting written to the underlying writer to the
			// .zip file we created at the beginning

			//log.Println("Time: ", time.Now().UnixNano())

			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("Error occurred while creating the zip file: ", err)
				return err
				color.Unset()
			}

			header.Name = _UPDATE_NAME + string(os.PathSeparator) + strings.TrimPrefix(path, _TEMP_DIR_NAME + string(os.PathSeparator))
			fileWriter, err := zipWriter.CreateHeader(header)

			//header := &zip.FileHeader{
			//	Name:         _PATCH_NAME + string(os.PathSeparator) + strings.TrimPrefix(path, _TEMP_DIR_NAME + string(os.PathSeparator)),
			//	//a.filename,
			//	//Method:       zip.Store,
			//	//crea
			//	//ModifiedTime: uint16(time.Now().UnixNano()),
			//	//ModifiedDate: uint16(time.Now().UnixNano()),
			//}

			//fileWriter, err := zipWriter.Create(_PATCH_NAME + string(os.PathSeparator) +
			//strings.TrimPrefix(path, _TEMP_DIR_NAME + string(os.PathSeparator)))

			//fileWriter, err := zipWriter.CreateHeader(header)


			if err != nil {
				color.Set(color.FgRed)
				fmt.Println("Error occurred while creating the zip file: ", err)
				os.Exit(1)
				color.Unset()
			}

			file, err := os.Open(path)
			if err != nil {
				log.Fatal("Y: ", err)
			}
			data, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatal("Z: ", err)
			}

			_, err = fileWriter.Write(data)
			if err != nil {
				log.Fatal("W: ", err)
			}
		}
		log.Printf("Visited: %s\n", path)
		return nil
	})
	if err != nil {
		log.Println("Directory Walk failed: ", err)
		color.Set(color.FgRed)
		fmt.Println("Patch file " + _UPDATE_NAME + ".zip creation failed")
		color.Unset()
		os.Exit(1)
	} else {
		log.Println("Directory Walk completed successfully")
		color.Set(color.FgGreen)
		fmt.Println("Patch file " + _UPDATE_NAME + ".zip successfully created")
		color.Unset()
	}
	// Clean up
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
}

//This function reads the files of the given distribution zip
func readDistributionZip(zipLocation string, logsEnabled bool) (bool, error) {
	log.Println("Zip file reading started: ", zipLocation)

	distName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(distName, string(os.PathSeparator)); lastIndex > -1 {
		distName = distName[lastIndex + 1:]
	}
	log.Println("distName used for trimming: ", distName)

	index := strings.LastIndex(zipLocation, string(os.PathSeparator))
	distLocation := zipLocation[:index]
	log.Println("distLocation: ", distLocation)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)

	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while reading zip:", err)
		color.Unset()
		log.Fatal(err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	log.Println("File count in zip: ", totalFiles)

	filesRead := 0

	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		filesRead++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from distribution zip: (%d/%d)\n", filesRead, totalFiles)
			time.Sleep(time.Millisecond * 2)
		}

		//log.Println("Checking file: ", file.Name)

		//var relativeLocation string
		relativeLocation := file.Name

		if file.FileInfo().IsDir() {
			//relativeLocation = strings.TrimPrefix(file.Name, distName)
			relativeLocation = strings.TrimSuffix(relativeLocation, string(os.PathSeparator) + file.FileInfo().Name() + string(os.PathSeparator))
			//log.Println("Entry: ", relativeLocation)

			//log.Println("FileName: ", file.FileInfo().Name())

		} else {
			//relativeLocation = strings.TrimPrefix(file.Name, distName)
			relativeLocation = strings.TrimSuffix(relativeLocation, string(os.PathSeparator) + file.FileInfo().Name())
			//log.Println("Entry: ", relativeLocation)

			//log.Println("FileName: ", file.FileInfo().Name())
		}

		relativeLocation = distLocation + string(os.PathSeparator) + relativeLocation
		log.Println("FileName:", file.FileInfo().Name(), "; relativeLocation:", relativeLocation)

		_, ok := distEntries[ file.FileInfo().Name()]
		if (ok) {
			entry := distEntries[file.FileInfo().Name()]
			entry.add(relativeLocation)
		} else {
			isDir := false
			if file.FileInfo().IsDir() {
				isDir = true
			}
			distEntries[file.FileInfo().Name()] = entry{
				map[string]bool{
					relativeLocation: isDir,
				},
			}
		}
	}

	writer.Stop()
	log.Println("Zip file reading finished")
	log.Println("Total files read: ", filesRead)
	if totalFiles == filesRead {
		log.Println("All files read")
		return true, nil
	} else {
		color.Set(color.FgRed)
		fmt.Println("All files not read from zip file")
		color.Unset()
		os.Exit(1)
		return false, nil
	}
}

////This function unzips a zip file at given location
//func unzip(zipLocation string) (bool, error) {
//	log.Println("Unzipping started")
//
//	// Create a reader out of the zip archive
//	zipReader, err := zip.OpenReader(zipLocation)
//
//	if err != nil {
//		color.Set(color.FgRed)
//		fmt.Println(err)
//		color.Unset()
//		log.Fatal(err)
//	}
//	defer zipReader.Close()
//
//	totalFiles := len(zipReader.Reader.File)
//	log.Println("File count in zip: ", totalFiles)
//
//	extractedFiles := 0
//
//	writer := uilive.New()
//	//start listening for updates and render
//	writer.Start()
//
//	targetDir := "./"
//	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
//		targetDir = zipLocation[:lastIndex]
//	}
//	// Iterate through each file/dir found in
//
//	for _, file := range zipReader.Reader.File {
//		extractedFiles++
//		fmt.Fprintf(writer, "Extracting files .. (%d/%d)\n", extractedFiles, totalFiles)
//
//		//bar.Set(extractedFiles)
//		time.Sleep(time.Millisecond * 5)
//
//		// Open the file inside the zip archive
//		// like a normal file
//		zippedFile, err := file.Open()
//		if err != nil {
//			log.Println(err)
//			return false, err
//		}
//
//		// Specify what the extracted file name should be.
//		// You can specify a full path or a prefix
//		// to move it to a different directory.
//		// In this case, we will extract the file from
//		// the zip to a file of the same name.
//		extractionPath := filepath.Join(
//			targetDir,
//			file.Name,
//		)
//		// Extract the item (or create directory)
//		if file.FileInfo().IsDir() {
//			// Create directories to recreate directory
//			// structure inside the zip archive. Also
//			// preserves permissions
//			//log.Println("Creating directory:", extractionPath)
//			os.MkdirAll(extractionPath, file.Mode())
//		} else {
//			// Extract regular file since not a directory
//			//log.Println("Extracting file:", file.Name)
//
//			// Open an output file for writing
//			outputFile, err := os.OpenFile(
//				extractionPath,
//				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
//				file.Mode(),
//			)
//			if err != nil {
//				log.Println(err)
//				return false, err
//			}
//			if outputFile != nil {
//				// "Extract" the file by copying zipped file
//				// contents to the output file
//				_, err = io.Copy(outputFile, zippedFile)
//				outputFile.Close()
//
//				if err != nil {
//					log.Println(err)
//					return false, err
//				}
//			}
//		}
//		zippedFile.Close()
//	}
//	writer.Stop()
//	log.Println("Unzipping finished")
//	log.Println("Extracted file count: ", extractedFiles)
//	if totalFiles == extractedFiles {
//		log.Println("All files extracted")
//		return true, nil
//	} else {
//		log.Println("All files not extracted")
//		return false, nil
//	}
//}
