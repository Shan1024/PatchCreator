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
	//Temporary directory location including carbon.home. All updated files will be copied to this location
	UPDATE_DIR_ROOT = TEMP_DIR + string(os.PathSeparator) + CARBON_HOME
	//Prefix of the update file and the root folder of the update zip
	UPDATE_NAME_PREFIX = "WSO2-CARBON-UPDATE"

	//Constants to store configs in viper
	UNZIP_DIRECTORY = "_UNZIP_DIRECTORY"
	DISTRIBUTION_ROOT = "DISTRIBUTION_ROOT"
	UPDATE_ROOT = "UPDATE_ROOT"
	UPDATE_NAME = "_UPDATE_NAME"

	INIT_EXAMPLE = `  update_number: 0001
  platform_version: 4.4.0
  platform_name: wilkes
  applies_to: All the products based on carbon 4.4.1
  bug_fixes:
    CARBON-15395: Upgrade Hazelcast version to 3.5.2
    <MORE_JIRAS_HERE>
  description: |
    This update contain the relavent fixes for upgrading Hazelcast version
    to its latest 3.5.2 version. When applying this update it requires a
    full cluster estart since if the nodes has multiple client versions of
    Hazelcast it can cause issues during connectivity.
  file_changes:
    added_files: []
    removed_files: []
    modified_files:
    - repository/components/plugins/hazelcast_3.5.0.wso2v1.jar`
)
