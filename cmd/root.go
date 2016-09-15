// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/ian-kent/go-log/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ian-kent/go-log/layout"
	"github.com/ian-kent/go-log/levels"
	"github.com/wso2/wum-uc/constant"
)

var (
	Version string
	BuildDate string
)

type Info struct {
	isDir bool
	md5   string
}

//key - filePath, value - Info
type LocationInfo struct {
	filepathInfoMap map[string]Info
}

func (l *LocationInfo) Add(location string, isDir bool, md5 string) {
	info := Info{
		isDir:isDir,
		md5:md5,
	}
	l.filepathInfoMap[location] = info
}

//key - filename, value - Locations
type FileLocationInfo struct {
	nameLocationInfoMap map[string]LocationInfo
}

func (f *FileLocationInfo) Add(filename string, location string, isDir bool, md5 string) {
	locationMap, found := f.nameLocationInfoMap[filename]
	if found {
		locationMap.Add(location, isDir, md5)
	} else {
		newLocation := LocationInfo{
			filepathInfoMap: make(map[string]Info),
		}
		newLocation.Add(location, isDir, md5)
		f.nameLocationInfoMap[filename] = newLocation
	}
}

//locationsInUpdate is a map to store the isDir
type LocationData struct {
	locationsInUpdate       map[string]bool
	locationsInDistribution map[string]bool
}
//key - filename , value - FileLocation
type Diff struct {
	files map[string]LocationData
}

func (d *Diff) Add(filename string, locationData LocationData) {
	d.files[filename] = locationData
}

var (
	//Create the logger
	logger = log.Logger()
	//Variable to store command line flags
	isDebugLogsEnabled bool
	isTraceLogsEnabled bool
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: "wum-uc",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".wum-uc") // name of config file (without extension)
	viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.AutomaticEnv()          // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

//This function will set the log level
func setLogLevel() {
	//Setting default time format. This will be used in loggers. Otherwise complete date and time will be printed
	layout.DefaultTimeLayout = "15:04:05"
	//Setting new STDOUT layout to logger
	logger.Appender().SetLayout(layout.Pattern("[%d] [%p] %m"))
	//Set the log level. If the log level is not given, set the log level to default level
	if isDebugLogsEnabled {
		logger.SetLevel(levels.DEBUG)
		logger.Debug("Debug logs enabled")
	} else if isTraceLogsEnabled {
		logger.SetLevel(levels.TRACE)
		logger.Trace("Trace logs enabled")
	} else {
		logger.SetLevel(constant.DEFAULT_LOG_LEVEL)
	}
	logger.Debug("[LOG LEVEL]", logger.Level())
}


