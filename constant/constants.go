// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package constant

import (
	"os"
	"github.com/ian-kent/go-log/levels"
)

const (
	DEFAULT_LOG_LEVEL = levels.WARN

	PATH_SEPARATOR = string(os.PathSeparator)
	PLUGINS_DIRECTORY = "repository" + PATH_SEPARATOR + "components" + PATH_SEPARATOR + "plugins" + PATH_SEPARATOR

	//constants to store resource file names
	README_FILE = "README.txt"
	LICENSE_FILE = "LICENSE.txt"
	NOT_A_CONTRIBUTION_FILE = "NOT_A_CONTRIBUTION.txt"
	INSTRUCTIONS_FILE = "instructions.txt"
	UPDATE_DESCRIPTOR_FILE = "update-descriptor.yaml"

	//Temporary directory to copy files before creating the new zip
	TEMP_DIR = "temp"
	//This is used to store carbon.home string
	CARBON_HOME = "carbon.home"
	//Prefix of the update file and the root folder of the update zip
	UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"

	//Constants to store configs in viper
	DISTRIBUTION_ROOT = "DISTRIBUTION_ROOT"
	UPDATE_ROOT = "UPDATE_ROOT"
	UPDATE_NAME = "_UPDATE_NAME"
	PRODUCT_NAME = "_PRODUCT_NAME"

	UPDATE_NUMBER_REGEX = "^\\d{4}$"
	KERNEL_VERSION_REGEX = "^\\d+\\.\\d+\\.\\d+$"
	FILENAME_REGEX = "^WSO2-CARBON-UPDATE-\\d+\\.\\d+\\.\\d+-\\d{4}.zip$"

	YES = "yes"
	Y = "y"
	NO = "no"
	N = "n"
	REENTER = "reenter"
	RE_ENTER = "re-enter"
	R = "r"

	IS_DEBUG_ENABLED = "IS_DEBUG_ENABLED"
	IS_TRACE_ENABLED = "IS_TRACE_ENABLED"

	SAMPLE = "SAMPLE"
	PROCESS_README = "PROCESS_README"
	AUTO_VALIDATE = "AUTO_VALIDATE"
	//default_values
	DEFAULT_VALUES = "DEFAULT_VALUES"
	PLATFORM_NAME = "PLATFORM_NAME"
	PLATFORM_VERSION = "PLATFORM_VERSION"
	BUG_FIXES = "BUG_FIXES"
	CHECK_MD5 = "CHECK_MD5"
	//update_repository
	UPDATE_REPOSITORY = "UPDATE_REPOSITORY"
	ENABLED = "ENABLED"
	LOCATION = "LOCATION"
	UPDATE_REPOSITORY_ENABLED = UPDATE_REPOSITORY + "." + ENABLED
	UPDATE_REPOSITORY_LOCATION = UPDATE_REPOSITORY + "." + LOCATION
	//resource_files
	RESOURCE_FILES = "RESOURCE_FILES"
	MANDATORY = "MANDATORY"
	OPTIONAL = "OPTIONAL"
	SKIP = "SKIP"
	PLATFORM_VERSIONS = "PLATFORM_VERSIONS"

	PATCH_ID_REGEX = "WSO2-CARBON-PATCH-(\\d+\\.\\d+\\.\\d+)-(\\d{4})"
	APPLIES_TO_REGEX = "(?s)Applies To.*?:(.*)Associated JIRA"
	ASSOCIATED_JIRAS_REGEX = "https:\\/\\/wso2\\.org\\/jira\\/browse\\/(\\S*)"
	DESCRIPTION_REGEX = "(?s)DESCRIPTION(.*)"
)


