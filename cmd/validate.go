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

func Validate(patchLocation, distributionLocation string, logsEnabled bool) {

	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("validate command called")

	initialize()

	log.Println("Patch Loc: " + patchLocation)
	if !isAZipFile(patchLocation) {
		color.Set(color.FgRed)
		fmt.Println("Patch file should be a zip file.")
		color.Unset()
		os.Exit(1)
	}
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
	if isAZipFile(distributionLocation) {
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
	validate()
}

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

func validate() {
	for patchLoc, _ := range patchFilesMap {
		log.Println("Checking location: ", patchLoc)
		_, found := distFileMap[patchLoc]
		if found {
			log.Println(patchLoc, "found in distFileMap")
		} else {
			log.Println(patchLoc, "not found in distFileMap")
			_, found := addedFilesMap[patchLoc]
			if found {
				log.Println(patchLoc, "found in addedFilesMap")
			} else {
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
func readPatchZip(zipLocation string, logsEnabled bool) (bool, error) {
	log.Println("Zip file reading started: ", zipLocation)

	patchName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(patchName, string(os.PathSeparator)); lastIndex > -1 {
		patchName = patchName[lastIndex + 1:]
	}

	log.Println("Patch name: ", patchName)

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

	filesRead := 0

	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		filesRead++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from patch zip: (%d/%d)\n", filesRead, totalFiles)
			time.Sleep(time.Millisecond * 5)
		}

		log.Println("Checking file: ", file.Name)

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

		_, found := allResFiles[file.FileInfo().Name()]

		if !found {
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
			log.Println(file.FileInfo().Name(), "was found in resource map")
			delete(allResFiles, file.FileInfo().Name())
			log.Println(file.FileInfo().Name(), "was removed from the map")

			if file.FileInfo().Name() == _UPDATE_DESCRIPTOR_FILE_NAME {
				yamlFile, err := file.Open()
				if err != nil {
					color.Set(color.FgRed)
					fmt.Println("Error occurred while reading the", _UPDATE_DESCRIPTOR_FILE_NAME, "file:",
						err)
					color.Unset()
				}
				data, err := ioutil.ReadAll(yamlFile)
				if err != nil {
					log.Fatal(err)
				}
				descriptor := update_descriptor{}

				err = yaml.Unmarshal(data, &descriptor)
				if err != nil {
					color.Set(color.FgRed)
					fmt.Println("Error occurred while unmarshalling the yaml:", err)
					color.Unset()
				}

				log.Println("descriptor:", descriptor)

				for _, addedFile := range descriptor.File_changes.Added_files {

					addedFilesMap[addedFile] = true
					//_, found := allResFiles[addedFile]
					//if found {
					//	log.Println(file, "found in patch")
					//
					//} else {
					//	fmt.Println(file, "not found in patch. But it is added as an entry in" +
					//	"'added_file' section in", _DESCRIPTOR_YAML_NAME)
					//	os.Exit(1)
					//}
				}
			}
		}
	}

	//Delete instructions.txt file if it is left in the map
	delete(allResFiles,_INSTRUCTIONS_FILE_NAME)

	log.Println("Resource map:", allResFiles)
	log.Println(patchFilesMap)
	if (len(allResFiles) != 0) {
		writer.Stop()
		color.Set(color.FgRed)
		fmt.Println("Following resource file(s) were not found in the patch: ")
		for key, _ := range allResFiles {
			fmt.Println("\t", "-", key)
		}
		color.Unset()
		os.Exit(1)
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

//This function reads the files of the given distribution zip
func readDistZip(zipLocation string, logsEnabled bool) (bool, error) {
	log.Println("Zip file reading started: ", zipLocation)

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

		log.Println("Checking file: ", file.Name)

		temp := strings.TrimPrefix(file.Name, distName)
		log.Println("Entry: ", temp)
		distFileMap[temp] = true

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

func readDistDir(distributionLocation string, logsEnabled bool) {

	writer := uilive.New()
	//start listening for updates and render
	writer.Start()
	filesRead := 0
	err := filepath.Walk(distributionLocation, func(path string, fileInfo os.FileInfo, err error) error {
		filesRead++;
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files from distribution directory: %d files read\n", filesRead)
			time.Sleep(time.Millisecond * 2)
		}

		log.Println("Walking: ", path)

		//if !fileInfo.IsDir() {
		// Create and write files to the archive, which in turn
		// are getting written to the underlying writer to the
		// .zip file we created at the beginning


		//header, err := zip.FileInfoHeader(fileInfo)
		//if err != nil {
		//	color.Set(color.FgRed)
		//	fmt.Println("Error occurred while creating the zip file: ", err)
		//	return err
		//	color.Unset()
		//}
		//
		//header.Name = _PATCH_NAME + string(os.PathSeparator) + strings.TrimPrefix(path, _TEMP_DIR_NAME + string(os.PathSeparator))

		temp := strings.TrimPrefix(path, distributionLocation)
		log.Println("Entry: ", temp)
		distFileMap[temp] = true


		//}
		//log.Printf("Visited: %s\n", path)
		return nil
	})

	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("Error occurred while reading the zip file: ", err)
		os.Exit(1)
		color.Unset()
	}
	log.Println("Total files read:", filesRead)
	writer.Stop()
}
