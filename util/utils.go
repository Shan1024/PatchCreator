// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package util

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/ian-kent/go-log/log"
	"gopkg.in/yaml.v2"
)

//todo: Move to a separate package?
//struct which is used to read update-descriptor.yaml
type UpdateDescriptor struct {
	Update_number    string
	Platform_version string
	Platform_name    string
	Applies_to       string
	Bug_fixes        map[string]string
	Description      string
	File_changes     struct {
				 Added_files    []string
				 Removed_files  []string
				 Modified_files []string
			 }
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


//This will return the md5 hash of the file in the given filepath
func GetMD5(filepath string) (string, error) {
	var result []byte
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(result)), nil
}

//todo: check for const
//This function will set the update name which will be used when creating the update zip
func GetUpdateName(updateDescriptor *UpdateDescriptor, updateNamePrefix string) string {
	//Read the corresponding details
	platformVersion := updateDescriptor.Platform_version
	updateNumber := updateDescriptor.Update_number
	updateName := updateNamePrefix + "-" + platformVersion + "-" + updateNumber
	return updateName
}

//This checks whether the distribution directory/zip exists
func IsDistributionExists(distributionPath string) (bool, error) {
	if strings.HasSuffix(distributionPath, ".zip") {
		return IsFileExists(distributionPath)
	}
	return IsDirectoryExists(distributionPath)
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
	if strings.ToLower(preference) == "reenter" || strings.ToLower(preference) == "re-enter" || (len(preference) == 1 && strings.ToLower(preference) == "r" ) {
		return true
	}
	return false
}

func GetUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	preference, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(preference), nil
}

func IsUserPreferencesValid(preferences []string, availableChoices int) (bool, error) {
	length := len(preferences)
	if length == 0 {
		return false, &CustomError{What:"No preferences entered."}
	}
	first, err := strconv.Atoi(preferences[0])
	if err != nil {
		return false, err
	}
	message := "Invalid preferences. Please select indices where " + strconv.Itoa(availableChoices) + ">= index >=1."
	if first < 1 {
		return false, &CustomError{What:message}
	}
	last, err := strconv.Atoi(preferences[length - 1])
	if err != nil {
		return false, err
	}
	if last > availableChoices {
		return false, &CustomError{What:message}
	}
	return true, nil
}

//This function will read update-descriptor.yaml
func LoadUpdateDescriptor(filename, updateDirectoryPath string) (*UpdateDescriptor, error) {

	//Construct the file path
	updateDescriptorPath := filepath.Join(updateDirectoryPath, filename)
	log.Debug("updateDescriptorPath:", updateDescriptorPath)

	//Read the file
	updateDescriptor := UpdateDescriptor{}
	yamlFile, err := ioutil.ReadFile(updateDescriptorPath)
	if err != nil {
		return nil, &CustomError{What: err.Error()}
	}
	//Un-marshal the update-descriptor file to updateDescriptor struct
	err = yaml.Unmarshal(yamlFile, &updateDescriptor)
	if err != nil {
		return nil, &CustomError{What: err.Error()}
	}
	return &updateDescriptor, nil
}

//todo: change error to
func ValidateUpdateDescriptor(updateDescriptor *UpdateDescriptor) error {
	if len(updateDescriptor.Update_number) == 0 {
		return &CustomError{What: "'update_number' field not found." }
	}
	//todo: use regex to validate Update_number format

	//todo: trim
	if len(updateDescriptor.Platform_version) == 0 {
		return &CustomError{What: "'platform_version' field not found." }
	}
	//todo: use regex to validate Platform_version format
	if len(updateDescriptor.Platform_name) == 0 {
		return &CustomError{What: "'platform_name' field not found." }
	}
	if len(updateDescriptor.Applies_to) == 0 {
		return &CustomError{What: "'applies_to' field not found." }
	}
	if len(updateDescriptor.Bug_fixes) == 0 {
		return &CustomError{What: "'bug_fixes' field not found." }
	}
	if len(updateDescriptor.Description) == 0 {
		return &CustomError{What: "'description' field not found." }
	}
	return nil
}

func PrintUpdateDescriptor(updateDescriptor *UpdateDescriptor) {
	fmt.Println("----------------------------------------------------------------")
	fmt.Printf("update_number: %s\n", updateDescriptor.Update_number)
	fmt.Printf("kernel_version: %s\n", updateDescriptor.Platform_version)
	fmt.Printf("platform_version: %s\n", updateDescriptor.Platform_name)
	fmt.Printf("applies_to: %s\n", updateDescriptor.Applies_to)
	fmt.Printf("bug_fixes: %s\n", updateDescriptor.Bug_fixes)
	fmt.Printf("file_changes: %s\n", updateDescriptor.File_changes)
	fmt.Printf("description: %s\n", updateDescriptor.Description)
	fmt.Println("----------------------------------------------------------------")
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

func HandleError(err error, customMessage ...interface{}) {
	if err != nil {
		PrintErrorAndExit(append(customMessage, "Error Message: '" + err.Error() + "'")...)
	}
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
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
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
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
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
	color.Set(color.Bold)
	fmt.Println(append([]interface{}{"[WARNING]"}, args...)...)
	color.Unset()
}

//This is used to print info messages
func PrintInfo(args ...interface{}) {
	color.Set(color.Bold)
	fmt.Println(append(append([]interface{}{"\n[INFO]"}, args...), "\n")...)
	color.Unset()
}
//
////This is used to print success messages
//func PrintSuccess(args ...interface{}) {
//	color.Set(color.FgGreen, color.Bold)
//	fmt.Println(append(append([]interface{}{"[INFO]"}, args...), "\n")...)
//	color.Unset()
//}
//
////This is used to print messages in Green color
//func PrintInGreen(args ...interface{}) {
//	//color.Set(color.FgGreen, color.Bold)
//	color.Set(color.Bold)
//	fmt.Print(args...)
//	color.Unset()
//}
//
////This is used to print messages in Yellow color
//func PrintInYellow(args ...interface{}) {
//	color.Set(color.FgYellow, color.Bold)
//	fmt.Print(args...)
//	color.Unset()
//}
//
////This is used to print messages in Red color
//func PrintInRed(args ...interface{}) {
//	color.Set(color.FgRed, color.Bold)
//	fmt.Print(args...)
//	color.Unset()
//}

func PrintInBold(args ...interface{}) {
	color.Set(color.Bold)
	fmt.Print(args...)
	color.Unset()
}

