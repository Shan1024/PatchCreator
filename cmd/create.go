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
)

type entry struct {
	locationMap map[string]bool
}

func (entry *entry) add(path string) {
	entry.locationMap[path] = true
}

type update_descriptor struct {
	Update_number    string `yaml:"update_number"`
	Kernel_version   string `yaml:"kernel_version"`
	Platform_version string `yaml:"platform_version"`
	Applies_to       string `yaml:"applies_to"`
	Bug_fixes        map[string]string `yaml:"bug_fixes"`
	Description      string `yaml:"description"`
}

const (
	_RESOURCE_DIR = "res"
	_TEMP_DIR_NAME = "temp"
	_CARBON_HOME = "carbon.home"
	_TEMP_DIR_LOCATION = _TEMP_DIR_NAME + string(os.PathSeparator) + _CARBON_HOME
	_DESCRIPTOR_YAML_NAME = "update-descriptor.yaml"
	_PATCH_NAME_PREFIX = "WSO2-CARBON-PATCH"
)

var (
	_RESOURCE_FILES = []string{"LICENSE.txt", "NOT_A_CONTRIBUTION.txt", "README.txt"}

	descriptor update_descriptor
	patchEntries map[string]entry
	distEntries map[string]entry

	_KERNEL_VERSION string
	_PATCH_NUMBER string
	_PATCH_NAME string
)

func Create(patchLocation, distributionLocation string, logsEnabled bool) {
	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("create command called")

	if strings.HasSuffix(patchLocation, string(os.PathSeparator)) {
		patchLocation = strings.TrimSuffix(patchLocation, string(os.PathSeparator))
	}
	log.Println("Patch Loc: " + patchLocation)

	patchLocationExists := checkDir(patchLocation)
	if patchLocationExists {
		log.Println("Patch location exists.")
	} else {
		fmt.Println("Patch location does not exist. Enter a valid directory.")
		os.Exit(1)
	}

	if (isAZipFile(distributionLocation)) {
		zipFileExists := checkFile(distributionLocation)
		if zipFileExists {
			log.Println("Distribution location exists.")
		} else {
			fmt.Println("Distribution zip does not exist. Enter a valid location.")
			os.Exit(1)
		}
	} else {
		distributionLocationExists := checkDir(distributionLocation)
		if distributionLocationExists {
			log.Println("Distribution location exists.")
		} else {
			fmt.Println("Distribution location does not exist. Enter a valid location.")
			os.Exit(1)
		}
	}

	log.Println("Product Loc: " + distributionLocation)

	descriptorLocation := patchLocation + string(os.PathSeparator) + _DESCRIPTOR_YAML_NAME
	log.Println("Descriptor Location: ", descriptorLocation)

	descriptorExists := checkFile(descriptorLocation);
	log.Println("Descriptor Exists: ", descriptorExists)

	if descriptorExists {
		readDescriptor(descriptorLocation)
	} else {
		//readPatchInfo()
		fmt.Println(_DESCRIPTOR_YAML_NAME + " not found at " + descriptorLocation)
		os.Exit(1)
	}

	patchEntries = make(map[string]entry)
	distEntries = make(map[string]entry)

	var unzipLocation string
	if isAZipFile(distributionLocation) {
		log.Println("Distribution location is a zip file. Extracting zip file...")
		unzipLocation = strings.TrimSuffix(distributionLocation, ".zip")
		log.Println("Unzip Location: " + unzipLocation)
		unzipSuccessful, err := unzip(distributionLocation)

		if err != nil {
			fmt.Println("Error occurred while extracting zip file")
		}
		if unzipSuccessful {
			log.Println("File successfully unzipped...")
			log.Println("Traversing patch location")
			traverse(patchLocation, patchEntries, false)
			log.Println("Traversing patch location finished")

			distLocationExists := checkDir(unzipLocation)
			if distLocationExists {
				log.Println("Distribution location(unzipped locations) exists. Reading files: ",
					unzipLocation)
			} else {
				fmt.Println("Distribution location(unzipped location) does not exist: ", unzipLocation)
				os.Exit(1)
			}

			log.Println("Traversing unzip location")
			traverse(unzipLocation, distEntries, true)
			log.Println("Traversing unzip location finished")
			log.Println("Finding matches")
			findMatches(patchLocation, unzipLocation)
			log.Println("Finding matches finished")
			log.Println("Copying resource files")
			copyResourceFiles(patchLocation)
			log.Println("Copying resource files finished")
			log.Println("Creating zip file")
			createPatchZip()
			log.Println("Creating zip file finished")
		} else {
			fmt.Println("Error occurred while unzipping")
		}
	} else {
		log.Println("Distribution location is not a zip file")
		log.Println("Traversing patch location")
		traverse(patchLocation, patchEntries, false)
		log.Println("Traversing patch location finished")
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
		log.Println("Error occurred while reading the descriptor: ", err)
	}
	descriptor = update_descriptor{}

	err = yaml.Unmarshal(yamlFile, &descriptor)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("update_number:", descriptor.Update_number)
	log.Println("kernel_version:", descriptor.Kernel_version)
	log.Println("platform_version:", descriptor.Platform_version)
	log.Println("applies_to: ", descriptor.Applies_to)
	log.Println("bug_fixes: ")
	for key, value := range (descriptor.Bug_fixes) {
		fmt.Println("\t", key, ":", value)
	}
	log.Println("description: \n" + descriptor.Description)

	if descriptor.Update_number == nil {
		fmt.Println("update_number", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}
	if descriptor.Kernel_version == nil {
		fmt.Println("kernel_version", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}
	if descriptor.Platform_version == nil {
		fmt.Println("platform_version", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}
	if descriptor.Applies_to == nil {
		fmt.Println("applies_to", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}
	if descriptor.Bug_fixes == nil {
		fmt.Println("bug_fixes", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}
	if descriptor.Description == nil {
		fmt.Println("description", " field not found in ", _DESCRIPTOR_YAML_NAME)
		os.Exit(1)
	}

	_KERNEL_VERSION, _ = descriptor.Kernel_version
	log.Println("kernel version set to: ", _KERNEL_VERSION)

	_PATCH_NUMBER, _ = descriptor.Update_number
	log.Println("patch number set to: ", _PATCH_NUMBER)

	_PATCH_NAME = _PATCH_NAME_PREFIX + "-" + _KERNEL_VERSION + "-" + _PATCH_NUMBER
	log.Println("Patch Name: " + _PATCH_NAME)
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
			fmt.Println("Resource: ", filePath, " not found")
		} else {
			log.Println("Copying resource: ", filePath, " to: " + _TEMP_DIR_NAME)
			err := CopyFile(filePath, _TEMP_DIR_NAME + string(os.PathSeparator) + resourceFile)
			if (err != nil) {
				fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			}
		}
	}
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

	err := os.RemoveAll(_TEMP_DIR_LOCATION)
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

		//todo check for underscore and dash when matching


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
				for path, isDirInDist := range distEntry.locationMap {

					for _, isDirInPatch := range patchEntry.locationMap {

						if isDirInDist == isDirInPatch {

							locationMap[strconv.Itoa(index)] = path

							//if isFirst {
							//	overallViewTable.AddRow(patchEntryString, path)
							//	isFirst = false
							//} else {
							//	overallViewTable.AddRow("", path)
							//}
							log.Println("Trimming: ", path, "; using: ", distPath)
							tempTable.AddRow(index, strings.TrimPrefix(path, distPath))
							index++
						}
					}
				}

				log.Println("Map: ", locationMap)

				tempTable.SetAlign(termtables.AlignCenter, 1)
				fmt.Println(tempTable.Render())

				//loop until user enter valid indices or decide to exit
				for {
					fmt.Println("Enter preferred locations separated by commas[Enter 0 to exit]: ")
					//fmt.Println(locationList)
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Preferred Locations: ")
					enteredPreferences, _ := reader.ReadString('\n')

					enteredPreferences = strings.TrimSuffix(enteredPreferences, "\n")
					selectedIndices := strings.Split(enteredPreferences, ",");

					sort.Strings(selectedIndices)

					if selectedIndices[0] == "0" {
						fmt.Println("0 entered. Exiting.....")
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

								CopyDir(src, dest)

							} else {
								fmt.Println("One or more entered indices are invalid. " +
								"Please enter again")
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

					for pathInDist, isDirInPatch := range patchEntry.locationMap {

						if isDirInDist == isDirInPatch {
							log.Println("Both locations contain same type")
							overallViewTable.AddRow(patchEntryString, strings.TrimPrefix(path, distPath))

							tempFilePath := strings.TrimPrefix(path, distributionLocation)

							src := path + string(os.PathSeparator) + patchEntryString
							destPath := _TEMP_DIR_LOCATION + tempFilePath + string(os.PathSeparator)
							dest := destPath + patchEntryString

							log.Println("src : ", src)
							log.Println("dest: ", dest)

							err := os.MkdirAll(destPath, 0777)

							//newFile, err := os.Create(dest)
							if err != nil {
								fmt.Println(err)
							}
							//newFile.Close()

							copyErr := CopyFile(src, dest)
							if copyErr != nil {
								fmt.Println(copyErr)
							}
						} else {

							fmt.Println(color.RedString("\nFollowing locations contain"),
								patchEntryString,
								color.RedString("but types are different"))
							color.Set(color.FgRed)
							fmt.Println(" - ", path)
							fmt.Println(" - ", pathInDist)
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
		log.Println("++++++++++++++++++++++++++++++++++++++++++++++++++++")
		rowCount++
		if rowCount < len(patchEntries) {
			overallViewTable.AddSeparator()
		}
	}
	log.Println("Matching files ended ------------------------------------------------------------------------")
	color.Set(color.FgYellow)
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
	for _, f := range files {
		_, ok := entryMap[f.Name()]
		if (ok) {
			entry := entryMap[f.Name()]
			//log.Println("ENTRY: ", &entry.locations[0])
			entry.add(path)
			//entryMap[f.Name()] = entry
		} else {
			isDir := false
			if f.IsDir() {
				isDir = true
			}
			entryMap[f.Name()] = entry{
				map[string]bool{
					path: isDir,
				},
			}
		}
		if f.IsDir() && isDist {
			//log.Println("Is a dir: " + path + string(os.PathSeparator) + f.Name())
			traverse(path + string(os.PathSeparator) + f.Name(), entryMap, isDist)
		}
	}
}

//This function creates the patch zip file
func createPatchZip() {
	log.Println("Creating patch zip file: ", _PATCH_NAME + ".zip")
	outFile, err := os.Create(_PATCH_NAME + ".zip")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	// Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)

	err = filepath.Walk(_TEMP_DIR_NAME, func(path string, f os.FileInfo, err error) error {

		log.Println("Walking: ", path)

		if !f.IsDir() {
			// Create and write files to the archive, which in turn
			// are getting written to the underlying writer to the
			// .zip file we created at the beginning
			fileWriter, err := zipWriter.Create(_PATCH_NAME + string(os.PathSeparator) +
			strings.TrimPrefix(path, _TEMP_DIR_NAME + string(os.PathSeparator)))
			if err != nil {
				log.Fatal("X: ", err)
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
		fmt.Println("Patch file " + _PATCH_NAME + ".zip creation failed")
		color.Unset()
	} else {
		log.Println("Directory Walk completed successfully")
		color.Set(color.FgGreen)
		fmt.Println("Patch file " + _PATCH_NAME + ".zip successfully created")
		color.Unset()
	}
	// Clean up
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
}

//This function unzips a zip file at given location
func unzip(zipLocation string) (bool, error) {
	log.Println("Unzipping started")

	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	log.Println("File count in zip: ", totalFiles)

	extractedFiles := 0

	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	targetDir := "./"
	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
		targetDir = zipLocation[:lastIndex]
	}
	// Iterate through each file/dir found in

	for _, file := range zipReader.Reader.File {
		extractedFiles++
		fmt.Fprintf(writer, "Extracting files .. (%d/%d)\n", extractedFiles, totalFiles)

		//bar.Set(extractedFiles)
		time.Sleep(time.Millisecond * 5)

		// Open the file inside the zip archive
		// like a normal file
		zippedFile, err := file.Open()
		if err != nil {
			log.Println(err)
			return false, err
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
			//log.Println("Creating directory:", extractionPath)
			os.MkdirAll(extractionPath, file.Mode())
		} else {
			// Extract regular file since not a directory
			//log.Println("Extracting file:", file.Name)

			// Open an output file for writing
			outputFile, err := os.OpenFile(
				extractionPath,
				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				file.Mode(),
			)
			if err != nil {
				log.Println(err)
				return false, err
			}
			if outputFile != nil {
				// "Extract" the file by copying zipped file
				// contents to the output file
				_, err = io.Copy(outputFile, zippedFile)
				outputFile.Close()

				if err != nil {
					log.Println(err)
					return false, err
				}
			}
		}
		zippedFile.Close()
	}
	writer.Stop()
	log.Println("Unzipping finished")
	log.Println("Extracted file count: ", extractedFiles)
	if totalFiles == extractedFiles {
		log.Println("All files extracted")
		return true, nil
	} else {
		log.Println("All files not extracted")
		return false, nil
	}
}
