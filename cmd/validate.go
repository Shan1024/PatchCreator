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
)

var allResFiles = map[string]bool{"LICENSE.txt":true, "NOT_A_CONTRIBUTION.txt":true, "README.txt":true,
	"update-descriptor.yaml":true}

var (
	patchFilesMap map[string]bool
	distFileMap map[string]bool
)

func Validate(patchLocation, distributionLocation string, logsEnabled bool) {

	if (!logsEnabled) {
		log.SetOutput(ioutil.Discard)
	} else {
		log.Println("Logs enabled")
	}
	log.Println("validate command called")

	log.Println("Patch Loc: " + patchLocation)
	if !isAZipFile(patchLocation) {
		fmt.Println("Patch file should be a zip file.")
		os.Exit(1)
	}
	patchLocationExists := checkFile(patchLocation)
	if patchLocationExists {
		log.Println("Patch location exists.")
	} else {
		fmt.Println("Patch location does not exist. Enter a valid file location.")
		os.Exit(1)
	}

	log.Println("Product Loc: " + distributionLocation)
	if isAZipFile(distributionLocation) {
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

	patchFilesMap = make(map[string]bool)
	distFileMap = make(map[string]bool)

	fmt.Println("Reading patch zip...")
	readPatchZip(patchLocation, logsEnabled)
}

//This function unzips a zip file at given location
func readPatchZip(zipLocation string, logsEnabled bool) (bool, error) {
	log.Println("Zip file reading started: ", zipLocation)

	patchName := strings.TrimSuffix(zipLocation, ".zip")
	if lastIndex := strings.LastIndex(patchName, string(os.PathSeparator)); lastIndex > -1 {
		patchName = patchName[lastIndex + 1:]
	}

	log.Println("Patch name: ", patchName)

	if !strings.HasPrefix(patchName, _PATCH_NAME_PREFIX) {
		fmt.Println("Patch file does not have", _PATCH_NAME_PREFIX, "prefix")
		os.Exit(0)
	} else {
		log.Println("Patch file does have", _PATCH_NAME_PREFIX, "prefix")
	}

	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.Reader.File)
	log.Println("File count in zip: ", totalFiles)

	filesReaded := 0

	writer := uilive.New()
	//start listening for updates and render
	writer.Start()

	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		filesReaded++
		if (!logsEnabled) {
			fmt.Fprintf(writer, "Reading files .. (%d/%d)\n", filesReaded, totalFiles)
			time.Sleep(time.Millisecond * 5)
		}

		log.Println("Checking file: ", file.Name)
		_, found := allResFiles[file.FileInfo().Name()]

		if !found {
			containsCarbonHome := strings.Contains(file.Name, _CARBON_HOME)
			if (!containsCarbonHome) {
				fmt.Println("Does not have a", _CARBON_HOME, "folder")
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
		}
	}

	log.Println("Resource map:", allResFiles)
	log.Println(patchFilesMap)
	if (len(allResFiles) != 0) {
		fmt.Println("Following resource file(s) were not found in the patch: ")
		for key, _ := range allResFiles {
			fmt.Println("\t", key)
		}
		os.Exit(1)
	}

	writer.Stop()
	log.Println("Zip file reading finished")
	log.Println("Total files read: ", filesReaded)
	if totalFiles == filesReaded {
		log.Println("All files read")
		return true, nil
	} else {
		log.Println("All files not read")
		return false, nil
	}
}
