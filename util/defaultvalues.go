package util

var ProcessReadMe = true
var AutoValidate = true
//default_values
var PlatformName = "wilkes"
var PlatformVersion = "4.4.0"
var BugFixes = "N/A"
var CheckMd5 = false
var UpdateRepository = "$HOME/Documents/Updates"
//resource_files
var Mandatory = []string{"update-descriptor.yaml", "LICENSE.txt"}
var Optional = []string{"instructions.txt", "NOT_A_CONTRIBUTION.txt"}
var Skip = []string{"README.txt"}
