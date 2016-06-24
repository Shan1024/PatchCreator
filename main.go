package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"github.com/shan1024/pct/cmd"
	"log"
)

var (
	app = kingpin.New("pct", "A command-line patch creation tool.")

	createCommand = app.Command("create", "Create a patch")
	createPatchLoc = createCommand.Arg("patch", "Patch dir location").Required().String()
	createDistLoc = createCommand.Arg("dist", "Dist dir/zip location").Required().String()
	enableLogsForCreateCommand = createCommand.Flag("logs", "Enable debug logs").Short('l').Bool()

	validateCommand = app.Command("validate", "Validates a patch")
	validatePatchLoc = validateCommand.Arg("patch", "Patch dir location").Required().String()
	validateDistLoc = validateCommand.Arg("dist", "Dist dir/zip location").Required().String()
	enableLogsForValidateCommand = validateCommand.Flag("logs", "Enable debug logs").Short('l').Bool()
)

func main() {
	output := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch output{
	case createCommand.FullCommand():
		cmd.Create(*createPatchLoc, *createDistLoc, *enableLogsForCreateCommand)
	case validateCommand.FullCommand():
		log.Println("validate command called")
	}
}
