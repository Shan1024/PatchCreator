package main

//pct /home/shan/work/test/patch /home/shan/work/test/product-dss-3.5.0.zip

import (
	"os"
	"archive/zip"
	"path/filepath"
	"io"
	"strings"
	"log"
	"io/ioutil"
	"github.com/gosuri/uilive"
	"fmt"
	"time"
)

type Entry struct {
	//locations []string
	locationMap map[string]bool
}

var patchEntries map[string]Entry
var distEntries map[string]Entry

func (entry *Entry) add(path string) {
	//entry.locations = append(entry.locations, path)
	entry.locationMap[path] = true
}

//func (entry *Entry) String() string {
//	str := ""
//	for _, path := range entry.locations {
//		str += str + path + "\n"
//	}
//	return fmt.Sprintf(str)
//}

func main() {
	log.Println("Welcome to Patch Creation Tool")
	args := os.Args
	if len(args) < 3 {
		log.Fatal("Missing arguments. Requires 2 arguments")
	}
	patchLocation := args[1]
	log.Println("Patch   Loc: " + patchLocation)
	distLocation := args[2]
	log.Println("Product Loc: " + distLocation)

	patchEntries = make(map[string]Entry)
	distEntries = make(map[string]Entry)

	var unzipLocation string
	if strings.HasSuffix(distLocation, ".zip") {
		log.Println("Distribution location is a zip file. Extracting zip file")
		unzipLocation = strings.TrimSuffix(distLocation, ".zip")
		log.Println("Unzip Location: " + unzipLocation)

		unzipSuccessful := unzip(distLocation)
		if unzipSuccessful {
			log.Println("Zip file successfully unzipped")

			patchLocationExists := checkLocation(patchLocation)
			if patchLocationExists {
				log.Println("Patch location exists. Reading files")
				traverse(patchLocation, patchEntries)
				//for key, value := range patchEntries {
				//	log.Print("Key:", key, " Value:")
				//	log.Println(value)
				//}
				//log.Println("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++###")
			} else {
				log.Println("Patch location does not exist")
			}

			distLocationExists := checkLocation(unzipLocation)
			if distLocationExists {
				log.Println("Distribution location exists. Reading files")
				//traverse(unzipLocation, &distEntries)
				traverse(unzipLocation, distEntries)
				//for key, value := range distEntries {
				//	if len(value.locationMap) > 1 {
				//		log.Print("Key:", key, " Value:")
				//		log.Println(value)
				//	}
				//}
			} else {
				log.Println("Distribution location does not exist")
			}
			//compare(patchLocation, unzipLocation)

			findMatches()
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

func findMatches() {
	log.Println("\n\n\nMatching files -----------------------------------------------------------------")
	for patchEntryString, patchEntry := range patchEntries {
		distEntry, ok := distEntries[patchEntryString]
		if ok {
			log.Println("Match found for ", patchEntryString)
			log.Println("Location(s) in Dist: ", distEntry)
		} else {
			log.Println("No match found for ", patchEntryString)
			log.Println("Location(s) in Patch: ", patchEntry)
		}
	}

}

func checkLocation(location string) bool {
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

func traverse(path string, entryMap map[string]Entry) {
	//log.Println("Root: " + path)
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		//for i := 0; i < indent; i++ {
		//	fmt.Print("-")
		//}
		//log.Println(path + string(os.PathSeparator) + f.Name())
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
		if f.IsDir() {
			//log.Println("Is a dir: " + path + string(os.PathSeparator) + f.Name())
			traverse(path + string(os.PathSeparator) + f.Name(), entryMap)
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

	totalFiles := len(zipReader.Reader.File)
	log.Println("Count: ", totalFiles)

	extractedFiles := 0

	writer := uilive.New()
	// start listening for updates and render
	writer.Start()

	targetDir := "./"
	if lastIndex := strings.LastIndex(zipLocation, string(os.PathSeparator)); lastIndex > -1 {
		targetDir = zipLocation[:lastIndex]
	}
	// Iterate through each file/dir found in

	for _, file := range zipReader.Reader.File {
		// Open the file inside the zip archive
		// like a normal file

		extractedFiles++

		fmt.Fprintf(writer, "Extracting files .. (%d/%d)\n", extractedFiles, totalFiles)
		time.Sleep(time.Millisecond * 1)

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

	writer.Stop() // flush and stop rendering

	log.Println("Extracted: ", extractedFiles)
	if totalFiles == extractedFiles {
		log.Println("Equal: true")
	} else {
		log.Println("Equal: false")
	}

	log.Println("Unzipping finished")
	return unzipSuccessful
}
