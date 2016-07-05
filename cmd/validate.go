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
	patchFilesMap map[string]bool
	distFileMap map[string]bool
	addedFilesMap map[string]bool
)

//Entry point of the validate command
func Validate(patchLocation, distributionLocation string, logsEnabled bool) {

	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("validate command called")

	//Initialize variables
	initialize()

	//patch location should be a zip file
	log.Println("Patch Loc: " + patchLocation)
	if !isAZipFile(patchLocation) {
		color.Set(color.FgRed)
		fmt.Println("Patch file should be a zip file.")
		color.Unset()
		os.Exit(1)
	}
	//Check whether the patch location exists
	patchLocationExists := checkFile(patchLocation)
	if patchLocationExists {
		log.Println("Patch location exists.")
	} else {
		color.Set(color.FgRed)
		fmt.Println("Patch location does not exist. Enter a valid file location.")
		color.Unset()
		os.Exit(1)
	}

	log.Println("Reading patch zip...")
	readPatchZip(patchLocation, logsEnabled)
	log.Println("Patch zip successfully read.")
	log.Println("Entries in patch zip: ", patchFilesMap)

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
			fmt.Println("Distribution zip does not exist. Enter a valid location.")
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
			fmt.Println("Distribution location does not exist. Enter a valid location.")
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

	patchFilesMap = make(map[string]bool)
	distFileMap = make(map[string]bool)
	addedFilesMap = make(map[string]bool)
}

//This method validates the files
func validate() {
	//Iterate through all the files in the patch. All files should be in the distribution unless they are newly
	// added files
	for patchLoc, _ := range patchFilesMap {
		log.Println("Checking location: ", patchLoc)
		//Check whether the distribution has a file with the same name
		_, found := distFileMap[patchLoc]
		//If there is a file
		if found {
			log.Println(patchLoc, "found in distFileMap")
		} else {
			//If there is no file
			log.Println(patchLoc, "not found in distFileMap")
			//Check whether it is a newly added file
			_, found := addedFilesMap[patchLoc]
			//if it is a newly added file
			if found {
				log.Println(patchLoc, "found in addedFilesMap")
			} else {
				//If it is not a newly added file, print an error
				log.Println(patchLoc, "not found in addedFilesMap")
				log.Println("addedFilesMap: ", addedFilesMap)
				color.Set(color.FgRed)
				fmt.Println(patchLoc, "not found in distribution and it is not a newly added file.")
				fmt.Println("If it is a new file, please add an entry in", _UPDATE_DESCRIPTOR_FILE_NAME,
					"file")
				fmt.Println("\nValidation FAILED\n")
				color.Unset()
				os.Exit(1)
			}
		}
	}
	fmt.Println("\nValidation SUCCESSFUL\n")
}

//This function reads the files of the given patch zip
func readPatchZip(zipLocation string, logsEnabled bool) {
	log.Println("Zip file reading started: ", zipLocation)

	patchName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(patchName, string(os.PathSeparator)); lastIndex > -1 {
		patchName = patchName[lastIndex + 1:]
	}
	log.Println("Patch name: ", patchName)

	//Check whether the patch name has the required prefix
	if !strings.HasPrefix(patchName, _UPDATE_NAME_PREFIX) {
		color.Set(color.FgRed)
		fmt.Println("Patch file does not have", _UPDATE_NAME_PREFIX, "prefix")
		color.Unset()
		os.Exit(1)
	} else {
		log.Println("Patch file does have", _UPDATE_NAME_PREFIX, "prefix")
	}

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

	fileCount := 0
	//Create a new writer to show the progress
	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in the zip
	for _, file := range zipReader.Reader.File {
		fileCount++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from patch zip: (%d/%d)\n", fileCount, totalFiles)
			time.Sleep(time.Millisecond * 5)
		}

		log.Println("Checking file: ", file.Name)

		//Every file should be in a root folder. Check for the os.PathSeparator character to identify this
		index := strings.Index(file.Name, string(os.PathSeparator))
		if index == -1 {
			color.Set(color.FgRed)
			fmt.Println("Patch zip file should have a root folder called", patchName)
			color.Unset()
			os.Exit(1)
		} else {
			rootFolder := file.Name[:index]
			log.Println("RootFolder:", rootFolder)
			if rootFolder != patchName {
				color.Set(color.FgRed)
				fmt.Println(file.Name, "should be in", patchName, "root folder. But it is in ",
					rootFolder, "folder")
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
				fmt.Println("Does not have a '" + _CARBON_HOME + "' folder in the root folder '" +
				patchName + "'")
				color.Unset()
				os.Exit(1)
			}
			log.Println("Have a", _CARBON_HOME, "folder")

			temp := strings.TrimPrefix(file.Name, patchName + string(os.PathSeparator) + _CARBON_HOME)
			log.Println("Entry: ", temp)
			patchFilesMap[temp] = true
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
					fmt.Println("Error occurred while reading the", _UPDATE_DESCRIPTOR_FILE_NAME, "file:",
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
					fmt.Println("Error occurred while unmarshalling the yaml:", err)
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
	log.Println(patchFilesMap)

	//At this point, the size of the allResFiles should be zero. If one or more files are not found, that means
	// that some required files are missing
	if (len(allResFiles) != 0) {
		//Print the missing files
		color.Set(color.FgRed)
		fmt.Println("Following resource file(s) were not found in the patch: ")
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
		fmt.Println("All files not read from zip file")
		color.Unset()
		os.Exit(1)
	}
}

//This function reads the files of the given distribution zip
func readDistZip(zipLocation string, logsEnabled bool) (bool, error) {
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
		fmt.Println("Error occurred while reading zip:", err)
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
		return true, nil
	} else {
		color.Set(color.FgRed)
		fmt.Println("All files not read from zip file")
		color.Unset()
		os.Exit(1)
		return false, nil
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
		fmt.Println("Error occurred while reading the zip file: ", err)
		os.Exit(1)
		color.Unset()
	}
	log.Println("Total files read:", fileCount)
	//stop the writer
	writer.Stop()
}
