package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"github.com/shan1024/uct/cmd"
)

var (
	//Create a new app
	app = kingpin.New("uct", "A command-line Update Creator Tool.")
	//Create 'create' command
	createCommand = app.Command("create", "Create an update zip")
	createPatchLoc = createCommand.Arg("patch", "Patch dir location").Required().String()
	createDistLoc = createCommand.Arg("dist", "Dist dir/zip location").Required().String()
	enableLogsForCreateCommand = createCommand.Flag("logs", "Enable debug logs").Short('l').Bool()
	//Create 'validate' command
	validateCommand = app.Command("validate", "Validates an update zip")
	validatePatchLoc = validateCommand.Arg("update", "Update zip location").Required().String()
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
