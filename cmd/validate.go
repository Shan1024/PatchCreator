package cmd

import (
	"log"
	"io/ioutil"
	"os"
	"fmt"
	"archive/zip"
	"github.com/gosuri/uilive"
	"time"
	"strings"
	"github.com/fatih/color"
	"path/filepath"
	"gopkg.in/yaml.v2"
)

var (
	allResFiles  map[string]bool
	updatedFilesMap map[string]bool
	distFileMap map[string]bool
	addedFilesMap map[string]bool
)

//Entry point of the validate command
func Validate(updateLocation, distributionLocation string, logsEnabled bool) {

	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("validate command called")

	//Initialize variables
	initialize()

	//Update location should be a zip file
	log.Println("Update Loc: " + updateLocation)
	if !isAZipFile(updateLocation) {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE]: Update file should be a zip file.")
		color.Unset()
		os.Exit(1)
	}
	//Check whether the update location exists
	updateLocationExists := checkFile(updateLocation)
	if updateLocationExists {
		log.Println("Update location exists.")
	} else {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE]: Update location does not exist. Enter a valid file location.")
		color.Unset()
		os.Exit(1)
	}

	log.Println("Reading update zip...")
	readUpdateZip(updateLocation, logsEnabled)
	log.Println("Update zip successfully read.")
	log.Println("Entries in update zip: ", updatedFilesMap)

	log.Println("Distribution Loc: " + distributionLocation)
	//Check whether the distribution is a zip or a directory
	if isAZipFile(distributionLocation) {
		//Check whether the distribution zip exists
		zipFileExists := checkFile(distributionLocation)
		if zipFileExists {
			log.Println("Distribution location exists.")
			readDistZip(distributionLocation, logsEnabled)
		} else {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE]: Distribution zip does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	} else {
		//Check whether the distribution location exists
		distributionLocationExists := checkDir(distributionLocation)
		if distributionLocationExists {
			log.Println("Distribution location exists.")
			readDistDir(distributionLocation, logsEnabled)
		} else {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE]: Distribution location does not exist. Enter a valid location.")
			color.Unset()
			os.Exit(1)
		}
	}
	//Validate files
	validate()
}

//This initializes the variables
func initialize() {
	allResFiles = make(map[string]bool)
	allResFiles[_LICENSE_FILE_NAME] = true
	allResFiles[_NOT_A_CONTRIBUTION_FILE_NAME] = true
	allResFiles[_README_FILE_NAME] = true
	allResFiles[_UPDATE_DESCRIPTOR_FILE_NAME] = true
	allResFiles[_INSTRUCTIONS_FILE_NAME] = true

	updatedFilesMap = make(map[string]bool)
	distFileMap = make(map[string]bool)
	addedFilesMap = make(map[string]bool)
}

//This method validates the files
func validate() {
	//Iterate through all the files in the update. All files should be in the distribution unless they are newly
	// added files
	for updateLoc, _ := range updatedFilesMap {
		log.Println("Checking location: ", updateLoc)
		//Check whether the distribution has a file with the same name
		_, found := distFileMap[updateLoc]
		//If there is a file
		if found {
			log.Println(updateLoc, "found in distFileMap")
		} else {
			//If there is no file
			log.Println(updateLoc, "not found in distFileMap")
			//Check whether it is a newly added file
			_, found := addedFilesMap[updateLoc]
			//if it is a newly added file
			if found {
				log.Println(updateLoc, "found in addedFilesMap")
			} else {
				//If it is not a newly added file, print an error
				log.Println(updateLoc, "not found in addedFilesMap")
				log.Println("addedFilesMap: ", addedFilesMap)
				color.Set(color.FgRed)
				fmt.Println("[FAILURE]:", updateLoc, "not found in distribution and it is not a " +
				"newly added file.")
				fmt.Println("If it is a new file, please add an entry in", _UPDATE_DESCRIPTOR_FILE_NAME,
					"file.")
				fmt.Println("\nValidation FAILED\n")
				color.Unset()
				os.Exit(1)
			}
		}
	}
	color.Set(color.FgGreen)
	fmt.Println("\n[INFO] Validation SUCCESSFUL\n")
	color.Unset()
}

//This function reads the files of the given update zip
func readUpdateZip(zipLocation string, logsEnabled bool) {
	log.Println("Zip file reading started: ", zipLocation)

	updateName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(updateName, string(os.PathSeparator)); lastIndex > -1 {
		updateName = updateName[lastIndex + 1:]
	}
	log.Println("Update name: ", updateName)

	//Check whether the update name has the required prefix
	if !strings.HasPrefix(updateName, _UPDATE_NAME_PREFIX) {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Update file does not have", _UPDATE_NAME_PREFIX, "prefix")
		color.Unset()
		os.Exit(1)
	} else {
		log.Println("Update file does have", _UPDATE_NAME_PREFIX, "prefix")
	}

	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while reading zip:", err)
		color.Unset()
		log.Fatal(err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	log.Println("File count in zip: ", totalFiles)

	fileCount := 0
	//Create a new writer to show the progress
	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in the zip
	for _, file := range zipReader.Reader.File {
		fileCount++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from update zip: (%d/%d)\n", fileCount, totalFiles)
			time.Sleep(time.Millisecond * 2)
		}

		log.Println("Checking file: ", file.Name)

		//Every file should be in a root folder. Check for the os.PathSeparator character to identify this
		index := strings.Index(file.Name, "/")//string(os.PathSeparator) removed because it does not work
		// properly in windows
		if index == -1 {
			color.Set(color.FgRed)
			fmt.Println("[FAILURE] Update zip file should have a root folder called", updateName)
			color.Unset()
			os.Exit(1)
		} else {
			rootFolder := file.Name[:index]
			log.Println("RootFolder:", rootFolder)
			if rootFolder != updateName {
				color.Set(color.FgRed)
				fmt.Println("[FAILURE]", file.Name, "should be in", updateName, "root folder. But it " +
				"is in ", rootFolder, "folder")
				color.Unset()
				os.Exit(1)
			}
		}
		//Check whether the file is a resource file
		_, found := allResFiles[file.FileInfo().Name()]
		//If it is not a resource file
		if !found {
			//It should be in a carbon.home folder
			containsCarbonHome := strings.Contains(file.Name, _CARBON_HOME)
			if (!containsCarbonHome) {
				color.Set(color.FgRed)
				//string(os.PathSeparator) removed because it does not work properly in windows
				fmt.Println("[FAILURE] '" + file.Name + "' is not a known resource file. " +
				"It should be in '" + updateName + "/" + _CARBON_HOME + "/" + "' folder")
				color.Unset()
				os.Exit(1)
			}
			log.Println("Have a", _CARBON_HOME, "folder")
			//string(os.PathSeparator) removed because it does not work properly in windows
			temp := strings.TrimPrefix(file.Name, updateName + "/" + _CARBON_HOME)
			log.Println("Entry: ", temp)
			updatedFilesMap[temp] = true
		} else {
			//If the file is a resource file, delete the entry from allResFiles. This map is later used
			// to track missing resource files
			log.Println(file.FileInfo().Name(), "was found in resource map")
			delete(allResFiles, file.FileInfo().Name())
			log.Println(file.FileInfo().Name(), "was removed from the map")
			//If the file is update-descriptor.yaml file, we need to read the newly added files.
			// Otherwise there will be no match for these files and validation will be failed
			if file.FileInfo().Name() == _UPDATE_DESCRIPTOR_FILE_NAME {
				//Open the file
				yamlFile, err := file.Open()
				if err != nil {
					color.Set(color.FgRed)
					fmt.Println("[FAILURE] Error occurred while reading the", _UPDATE_DESCRIPTOR_FILE_NAME, "file:",
						err)
					color.Unset()
				}
				//Get the byte array
				data, err := ioutil.ReadAll(yamlFile)
				if err != nil {
					log.Fatal(err)
				}
				descriptor := update_descriptor{}
				//Get the values
				err = yaml.Unmarshal(data, &descriptor)
				if err != nil {
					color.Set(color.FgRed)
					fmt.Println("[FAILURE] Error occurred while unmarshalling the yaml:", err)
					color.Unset()
				}
				log.Println("descriptor:", descriptor)
				//Add all files to addedFilesMap
				for _, addedFile := range descriptor.File_changes.Added_files {
					addedFilesMap[addedFile] = true
				}
			}
		}
	}
	//Stop the writer
	writer.Stop()

	//Delete instructions.txt file if it is left in the map because it is optional
	delete(allResFiles, _INSTRUCTIONS_FILE_NAME)
	log.Println("Resource map:", allResFiles)
	log.Println(updatedFilesMap)

	//At this point, the size of the allResFiles should be zero. If one or more files are not found, that means
	// that some required files are missing
	if (len(allResFiles) != 0) {
		//Print the missing files
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Following resource file(s) were not found in the update zip: ")
		for key, _ := range allResFiles {
			fmt.Println("\t", "-", key)
		}
		color.Unset()
		os.Exit(1)
	}

	//Check whether all files are read
	log.Println("Zip file reading finished")
	log.Println("Total files read: ", fileCount)
	if totalFiles == fileCount {
		log.Println("All files read")
	} else {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] All files not read from zip file")
		color.Unset()
		os.Exit(1)
	}
}

//This function reads the files of the given distribution zip
func readDistZip(zipLocation string, logsEnabled bool) {
	log.Println("Zip file reading started: ", zipLocation)

	//Get the distribution name
	distName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(distName, string(os.PathSeparator)); lastIndex > -1 {
		distName = distName[lastIndex + 1:]
	}

	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while reading zip:", err)
		color.Unset()
		log.Fatal(err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	log.Println("File count in zip: ", totalFiles)

	fileCount := 0
	//Create a writer to show the progress
	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		fileCount++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from distribution zip: (%d/%d)\n", fileCount, totalFiles)
			time.Sleep(time.Millisecond * 2)
		}

		log.Println("Checking file: ", file.Name)

		//Get the relative path in the zip
		temp := strings.TrimPrefix(file.Name, distName)
		log.Println("Entry: ", temp)
		//Add to the map
		distFileMap[temp] = true

	}
	//Stop the writer
	writer.Stop()

	//Check whether all files are read
	log.Println("Zip file reading finished")
	log.Println("Total files read: ", fileCount)
	if totalFiles == fileCount {
		log.Println("All files read")
	} else {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] All files not read from zip file")
		color.Unset()
		os.Exit(1)
	}
}

func readDistDir(distributionLocation string, logsEnabled bool) {
	//Create a writer to show the progress
	writer := uilive.New()
	//start listening for updates and render
	writer.Start()
	fileCount := 0
	//Start the walk
	err := filepath.Walk(distributionLocation, func(path string, fileInfo os.FileInfo, err error) error {
		fileCount++;
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from distribution directory: %d files read\n", fileCount)
			time.Sleep(time.Millisecond * 2)
		}
		log.Println("Walking: ", path)
		//Get the relative path
		temp := strings.TrimPrefix(path, distributionLocation)
		log.Println("Entry: ", temp)
		//Add to the map
		distFileMap[temp] = true
		return nil
	})
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("[FAILURE] Error occurred while reading the zip file: ", err)
		os.Exit(1)
		color.Unset()
	}
	log.Println("Total files read:", fileCount)
	//stop the writer
	writer.Stop()
}
