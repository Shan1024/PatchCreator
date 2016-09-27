package util

var ProcessReadMe = true
var AutoValidate = true
//default_values
var PlatformName_Default = "wilkes"
var PlatformVersion_Default = "4.4.0"
var BugFixes_Default = "N/A"
var CheckMd5 = false
//update_repository
var UpdateRepository_Enabled = false
var UpdateRepository_Location = "$HOME/Documents/Updates"
//resource_files
var ResourceFiles_Mandatory = []string{"update-descriptor.yaml", "LICENSE.txt"}
var ResourceFiles_Optional = []string{"instructions.txt", "NOT_A_CONTRIBUTION.txt"}
var ResourceFiles_Skip = []string{"README.txt"}
