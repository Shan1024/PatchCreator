package util

import (
	"path"
)

var EnableDebugLogs = false
var EnableTraceLogs = false
var PrintSampleSelected = false

var ProcessReadMe = false
var AutoValidate = false
//default_values
var PlatformName_Default = "wilkes"
var PlatformVersion_Default = "4.4.0"
var BugFixes_Default = "N/A"
var CheckMd5 = false
//update_repository
var UpdateRepository_Enabled = false
var UpdateRepository_Location = path.Join(HomeDirectory, "/Documents/Updates")
//resource_files
var ResourceFiles_Mandatory = []string{"update-descriptor.yaml", "LICENSE.txt"}
var ResourceFiles_Optional = []string{"instructions.txt", "NOT_A_CONTRIBUTION.txt"}
var ResourceFiles_Skip = []string{"README.txt"}

var PlatformVersions = map[string]string{
	"4.2.0": "turing",
	"4.3.0": "perlis",
	"4.4.0": "wilkes",
	"5.0.0": "hamming",
}
