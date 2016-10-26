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
	//Prefix of the update file and the root directory of the update zip
	UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"

	//Constants to store configs in viper
	DISTRIBUTION_ROOT = "DISTRIBUTION_ROOT"
	UPDATE_ROOT = "UPDATE_ROOT"
	UPDATE_NAME = "_UPDATE_NAME"
	PRODUCT_NAME = "_PRODUCT_NAME"

	UPDATE_NUMBER_REGEX = "^\\d{4}$"
	KERNEL_VERSION_REGEX = "^\\d+\\.\\d+\\.\\d+$"
	FILENAME_REGEX = "^WSO2-CARBON-UPDATE-\\d+\\.\\d+\\.\\d+-\\d{4}.zip$"

	OTHER = 0
	YES = 1
	NO = 2
	REENTER = 3

	SAMPLE = "SAMPLE"
	CHECK_MD5_DISABLED = "CHECK_MD5_DISABLED"
	//resource_files
	RESOURCE_FILES = "RESOURCE_FILES"
	MANDATORY = "MANDATORY"
	OPTIONAL = "OPTIONAL"
	SKIP = "SKIP"
	RESOURCE_FILES_MANDATORY = RESOURCE_FILES + "." + MANDATORY
	RESOURCE_FILES_OPTIONAL = RESOURCE_FILES + "." + OPTIONAL
	RESOURCE_FILES_SKIP = RESOURCE_FILES + "." + SKIP

	PLATFORM_VERSIONS = "PLATFORM_VERSIONS"

	PATCH_ID_REGEX = "WSO2-CARBON-PATCH-(\\d+\\.\\d+\\.\\d+)-(\\d{4})"
	APPLIES_TO_REGEX = "(?s)Applies To.*?:(.*)Associated JIRA|Applies To.*?:(.*)DESCRIPTION"
	ASSOCIATED_JIRAS_REGEX = "https:\\/\\/wso2\\.org\\/jira\\/browse\\/([A-Z]*?-\\d+)"
	DESCRIPTION_REGEX = "(?s)DESCRIPTION(.*)"
	JIRA_SUMMARY_REGEX = "summary\":(\"(.*?)\")"

	PATCH_REGEX = "(?m).*patch.*"

	JIRA_API_URL = "https://wso2.org/jira/rest/api/latest/issue/"

	UPDATE_NO_DEFAULT = "ADD_UPDATE_NUMBER_HERE"
	PLATFORM_NAME_DEFAULT = "ADD_PLATFORM_NAME_HERE"
	PLATFORM_VERSION_DEFAULT = "ADD_PLATFORM_VERSION_HERE"
	APPLIES_TO_DEFAULT = "ADD_APPLIES_TO_HERE"
	DESCRIPTION_DEFAULT = "ADD_DESCRIPTION_HERE\n"

	JIRA_KEY_DEFAULT = "ADD_JIRA_KEY_HERE"
	JIRA_NA = "N/A"
	JIRA_SUMMARY_DEFAULT = "ADD_JIRA_SUMMARY_HERE"
)
