package main

//pct /home/shan/work/test/patch /home/shan/work/test/product-dss-3.5.0.zip

import (
	"fmt"
	"os"
	"archive/zip"
	"path/filepath"
	"io"
	"strings"
	"log"
	"io/ioutil"
)

type Entry struct{

}
func main() {
	fmt.Println("Welcome to Patch Creation Tool")
	args := os.Args
	if len(args) < 3 {
		log.Fatal("Missing arguments. Requires 2 arguments")
	}
	patchLocation := args[1]
	log.Println("Patch   Loc: " + patchLocation)
	distLocation := args[2]
	log.Println("Product Loc: " + distLocation)

	var unzipLocation string
	if strings.HasSuffix(distLocation, ".zip") {
		log.Println("Distribution location is a zip file. Extracting zip file")
		unzipLocation = strings.TrimSuffix(distLocation, ".zip")
		log.Println("Unzip Location: " + unzipLocation)
		unzipSuccessful := unzip(distLocation)
		if unzipSuccessful {
			log.Println("Zip file successfully unzipped")

			patchLocationExists := locationExists(patchLocation)
			if patchLocationExists {
				log.Println("Patch location exists. Reading files")
				traverse(patchLocation, 0)
			} else {
				log.Println("Patch location does not exist")
			}

			distLocationExists := locationExists(unzipLocation)
			if distLocationExists {
				log.Println("Distribution location exists. Reading files")
			} else {
				log.Println("Distribution location does not exist")
			}
			//compare(patchLocation, unzipLocation)
		} else {
			log.Println("Error occurred while unzipping")
		}
	} else {
		log.Println("Distribution location is not a zip file")
		//compare(patchLocation, distLocation)
	}
	//reader := bufio.NewReader(os.Stdin)
	//fmt.Print("Enter text: ")
	//text, _ := reader.ReadString('\n')
	//fmt.Println(text)
}
//  	/home/shan/work/test/wso2carbon-kernel-5.1.0.zip


func locationExists(location string) bool {
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

func traverse(path string, indent int) {
	//log.Println("Root: " + path)
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		for i := 0; i < indent; i++ {
			fmt.Print("-")
		}
		fmt.Println(path + string(os.PathSeparator) + f.Name())
		if f.IsDir() {
			//log.Println("Is a dir: " + path + string(os.PathSeparator) + f.Name())
			traverse(path + string(os.PathSeparator) + f.Name(), indent + 1)
		}
	}
	//patchStat, err := os.Stat(path)
	//
	//if err != nil {
	//	if os.IsNotExist(err) {
	//		log.Fatal("Patch file does not exist")
	//	}
	//}
	//
	//if patchStat.IsDir() {
	//	log.Println("Is a directory")
	//}else{
	//	log.Println("Is a file")
	//}
}

func unzip(zipLocation string) bool {
	log.Println("Unzipping started")
	unzipSuccessful := true
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(zipLocation)
	if err != nil {
		log.Fatal(err)
	}
	defer zipReader.Close()

	targetDir := "./"
	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
		targetDir = zipLocation[:lastIndex]
	}
	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		// Open the file inside the zip archive
		// like a normal file
		zippedFile, err := file.Open()
		if err != nil {
			unzipSuccessful = false
			log.Println(err)
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
			log.Println("Creating directory:", extractionPath)
			os.MkdirAll(extractionPath, file.Mode())
		} else {
			// Extract regular file since not a directory
			log.Println("Extracting file:", file.Name)
			// Open an output file for writing
			outputFile, err := os.OpenFile(
				extractionPath,
				os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
				file.Mode(),
			)
			if err != nil {
				unzipSuccessful = false
				log.Println(err)
			}
			if outputFile != nil {
				// "Extract" the file by copying zipped file
				// contents to the output file
				_, err = io.Copy(outputFile, zippedFile)
				outputFile.Close()

				if err != nil {
					unzipSuccessful = false
					log.Println(err)
				}
			}
		}
	}
	log.Println("Unzipping finished")
	return unzipSuccessful
}
