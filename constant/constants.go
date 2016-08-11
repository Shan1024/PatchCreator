package constant

import (
	"os"
	"github.com/ian-kent/go-log/levels"
)

const (
	DEFAULT_LOG_LEVEL = levels.WARN

	PLUGINS_DIRECTORY = "repository" + string(os.PathSeparator) + "components" + string(os.PathSeparator) + "plugins" + string(os.PathSeparator)

	//constants to store resource files
	README_FILE = "README.txt"
	LICENSE_FILE = "LICENSE.txt"
	NOT_A_CONTRIBUTION_FILE = "NOT_A_CONTRIBUTION.txt"
	INSTRUCTIONS_FILE = "instructions.txt"
	UPDATE_DESCRIPTOR_FILE = "update-descriptor.yaml"

	//Temporary directory to copy files before creating the new zip
	TEMP_DIR = "temp"
	//This is used to store carbon.home string
	CARBON_HOME = "carbon.home"
	//Temporary directory location including carbon.home. All updated files will be copied to this location
	UPDATE_DIR_ROOT = TEMP_DIR + string(os.PathSeparator) + CARBON_HOME
	//Prefix of the update file and the root folder of the update zip
	UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"

	UNZIP_DIRECTORY = "_UNZIP_DIRECTORY"
	DISTRIBUTION_ROOT = "_DISTRIBUTION_ROOT"
	UPDATE_ROOT = "UPDATE_ROOT"
	UPDATE_NAME = "_UPDATE_NAME"

	IS_DEBUG_LOGS_ENABLED = "IS_DEBUG_LOGS_ENABLED"
	IS_TRACE_LOGS_ENABLED = "IS_TRACE_LOGS_ENABLED"
)
