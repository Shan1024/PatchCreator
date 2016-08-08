package cmd

import (
	"strings"
	"os"
	"io"
	"fmt"
	"io/ioutil"
	"github.com/fatih/color"
)

//Check whether the given path contain a zip file
func isAZipFile(path string) bool {
	return strings.HasSuffix(path, ".zip")
}

//Check whether the given string is in the given slice
func stringIsInSlice(a string, list []string) bool {
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
			fmt.Println("Error occurred while copying:", err)
			os.Exit(1)
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
				fmt.Println("Error occurred while copying:", err)
				os.Exit(1)
			}
		} else {
			// perform copy
			err = CopyFile(sfp, dfp)
			if err != nil {
				fmt.Println("Error occurred while copying:", err)
				os.Exit(1)
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
func directoryExists(location string) bool {
	logger.Trace("Checking Location: %s", location)
	locationInfo, err := os.Stat(location)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if locationInfo.IsDir() {
		return true
	}
	return false
}

//Check whether the given location points to a file
func fileExists(location string) bool {
	logger.Trace("Checking location: %s", location)
	locationInfo, err := os.Stat(location)
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

//This is used to print failure messages
func printFailure(args ...interface{}) {
	color.Set(color.FgRed, color.Bold)
	fmt.Println(append(append([]interface{}{"\n[FAILURE]"}, args...), "\n")...)
	color.Unset()
}
//This is used to print failure messages and exit
func printFailureAndExit(args ...interface{}) {
	printFailure(args...)
	os.Exit(1)
}

func printWarning(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Println(append(append([]interface{}{"[WARNING]"}, args...), "\n")...)
	color.Unset()
}

func printInfo(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Println(append(append([]interface{}{"[INFO]"}, args...), "\n")...)
	color.Unset()
}

func printSuccess(args ...interface{}) {
	color.Set(color.FgGreen, color.Bold)
	fmt.Println(append(append([]interface{}{"[INFO]"}, args...), "\n")...)
	color.Unset()
}

func printInYellow(args ...interface{}) {
	color.Set(color.FgYellow, color.Bold)
	fmt.Print(args...)
	color.Unset()
}
