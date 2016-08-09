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
	createUpdateLoc = createCommand.Arg("update_loc", "Update dir location").Required().String()
	createDistLoc = createCommand.Arg("dist_loc", "Distribution dir/zip location").Required().String()
	enableDebugLogsForCreateCommand = createCommand.Flag("debug", "Enable debug logs").Short('d').Bool()
	enableTraceLogsForCreateCommand = createCommand.Flag("trace", "Enable debug logs").Short('t').Bool()
	//Create 'validate' command
	validateCommand = app.Command("validate", "Validates an update zip")
	validateUpdateLoc = validateCommand.Arg("update_loc", "Update zip location").Required().String()
	validateDistLoc = validateCommand.Arg("dist_loc", "Distribution dir/zip location").Required().String()
	enableDebugLogsForValidateCommand = validateCommand.Flag("debug", "Enable debug logs").Short('d').Bool()
	enableTraceLogsForValidateCommand = validateCommand.Flag("trace", "Enable debug logs").Short('t').Bool()
	//set the default version
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
		cmd.Create(*createUpdateLoc, *createDistLoc, *enableDebugLogsForCreateCommand, *enableTraceLogsForCreateCommand)
	case validateCommand.FullCommand():
		cmd.Validate(*validateUpdateLoc, *validateDistLoc, *enableDebugLogsForValidateCommand, *enableTraceLogsForValidateCommand)
	}
}

//This function sets the version details which will be displayed when --version flag is entered
func setVersion() {
	if len(buildDate) == 0 {
		buildDate = time.Now().Format(time.UnixDate)
	}
	version := ("WSO2 Update Creation Tool (UCT) version: " + uctVersion + "\n")
	version += ("Build date: " + buildDate + "\n")
	version += ("OS\\Arch: " + runtime.GOOS + "\\" + runtime.GOARCH + "\n")
	version += ("Go version: " + runtime.Version() + "\n")
	app.Version(version)
}
