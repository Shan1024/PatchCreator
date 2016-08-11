package cmd

import (
	"fmt"
	"github.com/ian-kent/go-log/log"
)

type Info struct {
	isDir bool
	md5   string
}

func (i Info) String() string {
	return fmt.Sprintf("{isDir: %v md5: %s}", i.isDir, i.md5)
}

//key - filePath, value - Info
type LocationInfo struct {
	filepathInfoMap map[string]Info
}

func (l *LocationInfo) Add(location string, isDir bool, md5 string) {
	info := Info{
		isDir:isDir,
		md5:md5,
	}
	l.filepathInfoMap[location] = info
}

//key - filename, value - Locations
type FileLocationInfo struct {
	nameLocationInfoMap map[string]LocationInfo
}

func (f *FileLocationInfo) Add(filename string, location string, isDir bool, md5 string) {
	locationMap, found := f.nameLocationInfoMap[filename]
	if found {
		locationMap.Add(location, isDir, md5)
	} else {
		newLocation := LocationInfo{
			filepathInfoMap: make(map[string]Info),
		}
		newLocation.Add(location, isDir, md5)
		f.nameLocationInfoMap[filename] = newLocation
	}
}

type LocationData struct {
	locationsInUpdate       map[string]bool
	locationsInDistribution map[string]bool
}
//key - filename , value - FileLocation
type Diff struct {
	files map[string]LocationData
}

func (d *Diff) Add(filename string, locationData LocationData) {
	d.files[filename] = locationData
}

//struct which is used to read update-descriptor.yaml
type update_descriptor struct {
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

var (
	//Create the logger
	logger = log.Logger()
	//Variable to store command line flags
	isDebugLogsEnabled bool
	isTraceLogsEnabled bool
)
