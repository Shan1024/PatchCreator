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
)

type Entry struct {
	locationMap map[string]bool
}

func (entry *Entry) add(path string) {
	entry.locationMap[path] = true
}

var patchEntries map[string]Entry
var distEntries map[string]Entry

var RESOURCE_FILES = []string{"LICENSE.txt", "NOT_A_CONTRIBUTION.txt", "README.txt"}

const RESOURCE_DIR = "res"
const TEMP_DIR_NAME = "temp"
const CARBON_HOME = "carbon.home"
const TEMP_DIR_LOCATION = TEMP_DIR_NAME + string(os.PathSeparator) + CARBON_HOME

var PATCH_NAME_PREFIX = "WSO2-CARBON-PATCH"
var KERNEL_VERSION string
var PATCH_NUMBER string
var PATCH_NAME string

func Create(patchLocation, distributionLocation string, logsEnabled bool) {
	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("create command called")
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

	readPatchInfo()

	patchEntries = make(map[string]Entry)
	distEntries = make(map[string]Entry)

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
			copyResourceFiles(patchLocation, unzipLocation)
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
		copyResourceFiles(patchLocation, distributionLocation)
		log.Println("Copying resource files finished")
		log.Println("Creating zip file")
		createPatchZip()
		log.Println("Creating zip file finished")
	}
}

func readPatchInfo() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter kernel version: ")
	KERNEL_VERSION, _ = reader.ReadString('\n')
	KERNEL_VERSION = strings.TrimSuffix(KERNEL_VERSION, "\n")
	log.Println("Entered kernel version: ", KERNEL_VERSION)

	fmt.Print("Enter patch number: ")
	PATCH_NUMBER, _ = reader.ReadString('\n')
	PATCH_NUMBER = strings.TrimSuffix(PATCH_NUMBER, "\n")
	log.Println("Entered patch number: ", PATCH_NUMBER)

	PATCH_NAME = PATCH_NAME_PREFIX + "-" + KERNEL_VERSION + "-" + PATCH_NUMBER + ".zip"
	log.Println("Patch Name: " + PATCH_NAME)
}

func copyResourceFiles(patchLocation, distributionLocation string) {
	for _, resourceFile := range RESOURCE_FILES {
		filePath := RESOURCE_DIR + string(os.PathSeparator) + resourceFile
		ok := checkFile(filePath)
		if !ok {
			fmt.Println("Resource: ", filePath, " not found")
		} else {
			log.Println("Copying resource: ", filePath, " to: " + TEMP_DIR_NAME)
			err := CopyFile(filePath, TEMP_DIR_NAME + string(os.PathSeparator) + resourceFile)
			if (err != nil) {
				fmt.Println("Error occurred while copying the resource file: ", filePath, err)
			}
		}
	}
}

func isAZipFile(path string) bool {
	return strings.HasSuffix(path, ".zip")
}

func findMatches(patchLocation, distributionLocation string) {

	//fmt.Println("Matching files started ------------------------------------------------------------------------")
	termtables.EnableUTF8()
	overallViewTable := termtables.CreateTable()
	//overallViewTable.AddTitle("Summary")
	overallViewTable.AddHeaders("File/Folder", "Copied To")

	err := os.RemoveAll(TEMP_DIR_LOCATION)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	err = os.MkdirAll(TEMP_DIR_LOCATION, 0777)

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

			if len(distEntry.locationMap) > 1 {
				color.Set(color.FgRed)
				fmt.Println("\n\"" + patchEntryString +
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
							tempTable.AddRow(index, path)
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
								destPath := TEMP_DIR_LOCATION + tempFilePath + string(os.PathSeparator)
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
											found := stringInSlice(path, selectedPathsList)
											if found {
												overallViewTable.AddRow(patchEntryString, path)
												isFirst = false
											}
										} else {
											found := stringInSlice(path, selectedPathsList)
											if found {
												overallViewTable.AddRow("", path)
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
							overallViewTable.AddRow(patchEntryString, path)

							tempFilePath := strings.TrimPrefix(path, distributionLocation)

							src := path + string(os.PathSeparator) + patchEntryString
							destPath := TEMP_DIR_LOCATION + tempFilePath + string(os.PathSeparator)
							dest := destPath + patchEntryString

							log.Println("src : ", src)
							log.Println("dest: ", dest)

							err := os.MkdirAll(destPath, 0777)

							//newFile, err := os.Create(dest)
							if err != nil {
								fmt.Errorf("Y: ", err)
							}
							//newFile.Close()

							copyErr := CopyFile(src, dest)
							if copyErr != nil {
								fmt.Errorf("X: ", copyErr)
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

func stringInSlice(a string, list []string) bool {
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

func traverse(path string, entryMap map[string]Entry, isDist bool) {
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
			entryMap[f.Name()] = Entry{
				map[string]bool{
					path: isDir,
				},
			}
		}
		if f.IsDir() &&isDist {
			//log.Println("Is a dir: " + path + string(os.PathSeparator) + f.Name())
			traverse(path + string(os.PathSeparator) + f.Name(), entryMap, isDist)
		}
	}
}

func createPatchZip() {
	// Create a file to write the archive buffer to
	// Could also use an in memory buffer.

	log.Println("Creating patch zip file: ", PATCH_NAME)
	outFile, err := os.Create(PATCH_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	// Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)

	err = filepath.Walk(TEMP_DIR_NAME, func(path string, f os.FileInfo, err error) error {

		log.Println("Walking: ", path)

		if !f.IsDir() {
			// Create and write files to the archive, which in turn
			// are getting written to the underlying writer to the
			// .zip file we created at the beginning
			fileWriter, err := zipWriter.Create(strings.TrimPrefix(path, TEMP_DIR_NAME + string(os.PathSeparator)))
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
		fmt.Println("Directory Walk failed: ", err)
	} else {
		log.Println("Directory Walk completed successfully")
	}
	// Clean up
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
}

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
		defer zippedFile.Close()
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
