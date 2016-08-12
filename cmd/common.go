//todo: add copyright notice

package cmd

import (
	"github.com/ian-kent/go-log/log"
)

//todo: move to a separate package
type Info struct {
	isDir bool
	md5   string
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

//locationsInUpdate is a map to store the isDir
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

var (
	//Create the logger
	logger = log.Logger()
	//Variable to store command line flags
	isDebugLogsEnabled bool
	isTraceLogsEnabled bool
)
