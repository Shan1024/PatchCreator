package cmd

import "os"

const (
	//constants to store resource files
	_README_FILE = "README.txt"
	_LICENSE_FILE = "LICENSE.txt"
	_NOT_A_CONTRIBUTION_FILE = "NOT_A_CONTRIBUTION.txt"
	_INSTRUCTIONS_FILE = "instructions.txt"
	_UPDATE_DESCRIPTOR_FILE = "update-descriptor.yaml"

	//Temporary directory to copy files before creating the new zip
	_TEMP_DIR = "temp"
	//This is used to store carbon.home string
	_CARBON_HOME = "carbon.home"
	//Temporary directory location including carbon.home. All updated files will be copied to this location
	_UPDATE_DIR_ROOT = _TEMP_DIR + string(os.PathSeparator) + _CARBON_HOME
	//Prefix of the update file and the root folder of the update zip
	_UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"

	_UNZIP_DIRECTORY = "_UNZIP_DIRECTORY"
	_DISTRIBUTION_ROOT = "_DISTRIBUTION_ROOT"
)
