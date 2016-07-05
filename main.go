package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"github.com/shan1024/pct/cmd"
)

var (
	//Create a new app
	app = kingpin.New("pct", "A command-line patch creation tool.")
	//Create 'create' command
	createCommand = app.Command("create", "Create a patch")
	createPatchLoc = createCommand.Arg("patch", "Patch dir location").Required().String()
	createDistLoc = createCommand.Arg("dist", "Dist dir/zip location").Required().String()
	enableLogsForCreateCommand = createCommand.Flag("logs", "Enable debug logs").Short('l').Bool()
	//Create 'validate' command
	validateCommand = app.Command("validate", "Validates a patch")
	validatePatchLoc = validateCommand.Arg("patch", "Patch dir location").Required().String()
	validateDistLoc = validateCommand.Arg("dist", "Dist dir/zip location").Required().String()
	enableLogsForValidateCommand = validateCommand.Flag("logs", "Enable debug logs").Short('l').Bool()
)

func main() {
	//parse the args
	output := kingpin.MustParse(app.Parse(os.Args[1:]))
	//call corresponding command
	switch output{
	case createCommand.FullCommand():
		cmd.Create(*createPatchLoc, *createDistLoc, *enableLogsForCreateCommand)
	case validateCommand.FullCommand():
		cmd.Validate(*validatePatchLoc, *validateDistLoc, *enableLogsForValidateCommand)
	}
}
