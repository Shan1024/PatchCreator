package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"github.com/shan1024/uct/cmd"
	"runtime"
	"time"
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

	uctVersion = "1.0.0"
	buildDate string
)

func main() {
	setVersion()
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

func setVersion() {
	if len(buildDate) == 0 {
		buildDate = time.Now().Format(time.UnixDate)
	}
	version := ("WSO2 Update Creation Tool (UCT) version: " + uctVersion + "\n")
	version += ("Build date: " + buildDate + "\n")
	version += ("OS\\Arch: " + runtime.GOOS + "\\" + runtime.GOARCH + "\n")
	version += ("Go version: " + runtime.Version() + "\n\n")
	app.Version(version)
}
