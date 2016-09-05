//todo: add copyright notice

package main

import "github.com/wso2/wum-uc/cmd"

// wum-uc version. Value is set during the build process.
var version string

// Build date of the particular build. Value is set during the build process
var buildDate string

func main() {
	cmd.Version = version
	cmd.BuildDate = buildDate

	cmd.Execute()
}
