package util

import (
	"fmt"
	"io"
	"os"
	"strings"

	"io/ioutil"
	"github.com/fatih/color"
)

func HasZipExtension(path string) bool {
	return strings.HasSuffix(path, ".zip")
}
func HasJarExtension(path string) bool {
	return strings.HasSuffix(path, ".jar")
}
func GetParentDirectory(filepath string) string {
	parentDirectory := "./"
	if lastIndex := strings.LastIndex(filepath, string(os.PathSeparator)); lastIndex > -1 {
		parentDirectory = filepath[:lastIndex]
	}
	return parentDirectory
}

func DeleteDirectory(path string) error {
	return os.RemoveAll(path)
}

func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0700)
}

func IsYes(preference string) bool {
	if strings.ToLower(preference) == "yes" || (len(preference) == 1 && strings.ToLower(preference) == "y" ) {
		return true
	}
	return false
}

func IsNo(preference string) bool {
	if strings.ToLower(preference) == "no" || (len(preference) == 1 && strings.ToLower(preference) == "n" ) {
		return true
	}
	return false
}

func IsReenter(preference string) bool {
	if strings.ToLower(preference) == "reenter" || (len(preference) == 1 && strings.ToLower(preference) == "r" ) {
		return true
	}
	return false
}

func HandleError(err error, customMessage ...interface{}) {
	if err != nil {
		PrintErrorAndExit(append(customMessage, "Error Message: '" + err.Error() + "'")...)
	}
}

//Check whether the given string is in the given slice
func IsStringIsInSlice(a string, list []string) bool {
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
			return os.Chmod(dest, si.Mode())
		}

	}
	return
}

//Recursively copies a directory tree, attempting to preserve permissions.
//Source directory must exist, destination directory must *not* exist.
func CopyDir(source string, dest string) (err error) {
	// get properties of source dir
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return &CustomError{What: "Source is not a directory"}
	}
	//Create the destination folder if it does not exist
	_, err = os.Open(dest)
	if os.IsNotExist(err) {
		// create dest dir
		err = os.MkdirAll(dest, fi.Mode())
		if err != nil {
			return &CustomError{What: err.Error()}
		}
	}
	entries, err := ioutil.ReadDir(source)
	for _, entry := range entries {
		sfp := source + "/" + entry.Name()
		dfp := dest + "/" + entry.Name()
		if entry.IsDir() {
			err = CopyDir(sfp, dfp)
			if err != nil {
				return err
			}
		} else {
			// perform copy
			err = CopyFile(sfp, dfp)
			if err != nil {
				return err
			}
		}
	}
	return
}

//A struct for returning custom error messages
type CustomError struct {
	What string
}

//Returns the error message defined in What as a string
func (e *CustomError) Error() string {
	return e.What
}

//Check whether the given location points to a directory
func IsDirectoryExists(location string) (bool, error) {
	locationInfo, err := os.Stat(location)
	if err != nil {
		return false, err
	}
	if locationInfo.IsDir() {
		return true, nil
	} else {
		return false, nil
	}
}

//Check whether the given location points to a file
func IsFileExists(location string) (bool, error) {
	locationInfo, err := os.Stat(location)
	if err != nil {
		return false, err
	}
	if locationInfo.IsDir() {
		return false, nil
	} else {
		return true, nil
	}
}

//This is used to print failure messages
func PrintError(args ...interface{}) {
	color.Set(color.FgRed, color.Bold)
	fmt.Println(append(append([]interface{}{"\n[ERROR]"}, args...), "\n")...)
	color.Unset()
}

//This is used to print failure messages and exit
func PrintErrorAndExit(args ...interface{}) {
	//call the printFailure method and exit
	PrintError(args...)
	os.Exit(1)
}

//This is used to print warning messages
func PrintWarning(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Println(append(append([]interface{}{"[WARNING]"}, args...), "\n")...)
	color.Unset()
}

//This is used to print info messages
func PrintInfo(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Println(append(append([]interface{}{"[INFO]"}, args...), "\n")...)
	color.Unset()
}

//This is used to print success messages
func PrintSuccess(args ...interface{}) {
	color.Set(color.FgGreen, color.Bold)
	fmt.Println(append(append([]interface{}{"[INFO]"}, args...), "\n")...)
	color.Unset()
}

//This is used to print messages in Yellow color
func PrintInYellow(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Print(args...)
	color.Unset()
}

//This is used to print messages in Red color
func PrintInRed(args ...interface{}) {
	color.Set(color.FgRed, color.Bold)
	fmt.Print(args...)
	color.Unset()
}

//func ZipDirectory(filename string, directory string) error {
//	// Create a file to write the archive buffer to
//	// Could also use an in memory buffer.
//	outFile, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer outFile.Close()
//
//	// Create a zip writer on top of the file writer
//	zipWriter := zip.NewWriter(outFile)
//
//
//	//todo: write walk function
//	// Add files to archive
//	// We use some hard coded data to demonstrate,
//	// but you could iterate through all the files
//	// in a directory and pass the name and contents
//	// of each file, or you can take data from your
//	// program and write it write in to the archive
//	// without
//	var filesToArchive = []struct {
//		Name, Body string
//	}{
//		{"test.txt", "String contents of file"},
//		{"test2.txt", "\x61\x62\x63\n"},
//	}
//
//	// Create and write files to the archive, which in turn
//	// are getting written to the underlying writer to the
//	// .zip file we created at the beginning
//	for _, file := range filesToArchive {
//		fileWriter, err := zipWriter.Create(file.Name)
//		if err != nil {
//			return err
//		}
//		_, err = fileWriter.Write([]byte(file.Body))
//		if err != nil {
//			return err
//		}
//	}
//
//	// Clean up
//	err = zipWriter.Close()
//	if err != nil {
//		return err
//	}
//	return nil
//}

//func unzip(zipFileName, targetDir string) error {
//	// Create a reader out of the zip archive
//	zipReader, err := zip.OpenReader(zipFileName)
//	if err != nil {
//		return err
//	}
//	defer zipReader.Close()
//
//	// Iterate through each file/dir found in
//	for _, file := range zipReader.Reader.File {
//
//		logger.Debug("File: %s", file.Name)
//		// Open the file inside the zip archive
//		// like a normal file
//		zippedFile, err := file.Open()
//		if err != nil {
//			return err
//		}
//
//		// Specify what the extracted file name should be.
//		// You can specify a full path or a prefix
//		// to move it to a different directory.
//		// In this case, we will extract the file from
//		// the zip to a file of the same name.
//		extractedFilePath := filepath.Join(
//			targetDir,
//			file.Name,
//		)
//
//		// Extract the item (or create directory)
//		if file.FileInfo().IsDir() {
//			// Create directories to recreate directory
//			// structure inside the zip archive. Also
//			// preserves permissions
//			logger.Debug("Creating directory:", extractedFilePath)
//			os.MkdirAll(extractedFilePath, file.Mode())
//		} else {
//			// Extract regular file since not a directory
//			logger.Debug("Extracting file:", file.Name)
//
//			// Open an output file for writing
//			outputFile, err := os.OpenFile(
//				extractedFilePath,
//				os.O_CREATE,
//				file.Mode(),
//			)
//			if err != nil {
//				return err
//			}
//			outputFile.Close()
//
//			// "Extract" the file by copying zipped file
//			// contents to the output file
//			_, err = io.Copy(outputFile, zippedFile)
//			if err != nil {
//				return err
//			}
//		}
//		zippedFile.Close()
//	}
//	return nil
//}
