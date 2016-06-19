package main

import (
	"fmt"
	"os"
	"archive/zip"
	"path/filepath"
	"io"
	"strings"
	"log"
)

func main() {
	fmt.Println("Welcome to Patch Creation Tool")
	args := os.Args
	patchLoc := args[1]
	prodLocation := args[2]
	fmt.Println("Patch Loc: " + patchLoc)
	fmt.Println("Product Loc: " + prodLocation)
	var unzipLocation string
	if strings.HasSuffix(prodLocation, ".zip") {
		fmt.Println("Product location is a zip")
		unzipLocation = strings.TrimSuffix(prodLocation, ".zip")
		fmt.Println("Unzip Location: " + unzipLocation)
		unzip(prodLocation)
	} else {
		fmt.Println("Product location is not a zip")
	}
	//reader := bufio.NewReader(os.Stdin)
	//fmt.Print("Enter text: ")
	//text, _ := reader.ReadString('\n')
	//fmt.Println(text)
}
///home/shan/work/test/wso2carbon-kernel-5.1.0.zip

func unzip(zipLocation string) {
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
			log.Fatal(err)
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
				log.Fatal(err)
			}

			// "Extract" the file by copying zipped file
			// contents to the output file
			_, err = io.Copy(outputFile, zippedFile)
			outputFile.Close()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

//func unzip(src, dest string) error {
//	r, err := zip.OpenReader(src)
//	if err != nil {
//		return err
//	}
//	defer r.Close()
//
//	for _, f := range r.File {
//		rc, err := f.Open()
//		if err != nil {
//			return err
//		}
//		defer rc.Close()
//
//		fpath := filepath.Join(dest, f.Name)
//		if f.FileInfo().IsDir() {
//			os.MkdirAll(fpath, f.Mode())
//		} else {
//			var fdir string
//			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
//				fdir = fpath[:lastIndex]
//			}
//
//			err = os.MkdirAll(fdir, f.Mode())
//			if err != nil {
//				log.Fatal(err)
//				return err
//			}
//			f, err := os.OpenFile(
//				fpath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, f.Mode())
//			if err != nil {
//				return err
//			}
//			defer f.Close()
//
//			_, err = io.Copy(f, rc)
//			if err != nil {
//				return err
//			}
//		}
//	}
//	return nil
//}

/*
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}*/
