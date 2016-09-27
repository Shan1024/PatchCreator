// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"fmt"
	"os"

	"github.com/ian-kent/go-log/layout"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
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

	setDefaultValues()

	viper.SetConfigName("config") // name of config file (without extension)
	//viper.SetConfigType("yaml")
	//viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.AddConfigPath(".")
	//viper.AddConfigPath("$HOME/work/src/github.com/wso2/wum-uc")
	viper.AddConfigPath("$HOME/.wum-uc")
	//viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("Config file not found")
	}

	fmt.Println("Config Values---------------------------")
	fmt.Println(viper.GetString(constant.PROCESS_README))
	fmt.Println(viper.GetString(constant.AUTO_VALIDATE))
	fmt.Println(viper.GetStringMapString(constant.DEFAULT_VALUES))
	fmt.Println(viper.GetString(constant.CHECK_MD5))
	fmt.Println(viper.GetString(constant.UPDATE_REPOSITORY + "." + constant.ENABLED))
	fmt.Println(viper.GetString(constant.UPDATE_REPOSITORY + "." + constant.LOCATION))
	fmt.Println(viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.MANDATORY))
	fmt.Println(viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.OPTIONAL))
	fmt.Println(viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.SKIP))
	fmt.Println("---------------------------------------")

}

//This function will set the log level
func setLogLevel() {
	//Setting default time format. This will be used in loggers. Otherwise complete date and time will be printed
	layout.DefaultTimeLayout = "15:04:05"
	//Setting new STDOUT layout to logger
	logger.Appender().SetLayout(layout.Pattern("[%d] [%p] %m"))
	//Set the log level. If the log level is not given, set the log level to default level

	isDebugLogsEnabled := viper.GetBool(constant.IS_DEBUG_ENABLED)
	isTraceLogsEnabled := viper.GetBool(constant.IS_TRACE_ENABLED)

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

func setDefaultValues() {
	viper.SetDefault(constant.PROCESS_README, util.ProcessReadMe)
	viper.SetDefault(constant.AUTO_VALIDATE, util.AutoValidate)
	viper.SetDefault(constant.DEFAULT_VALUES, map[string]string{
		constant.PLATFORM_NAME: util.PlatformName_Default,
		constant.PLATFORM_VERSION: util.PlatformVersion_Default,
		constant.BUG_FIXES: util.BugFixes_Default,
	})
	viper.SetDefault(constant.CHECK_MD5, util.CheckMd5)
	viper.SetDefault(constant.UPDATE_REPOSITORY + "." + constant.ENABLED, util.UpdateRepository_Enabled)
	viper.SetDefault(constant.UPDATE_REPOSITORY + "." + constant.LOCATION, util.UpdateRepository_Location)
	viper.SetDefault(constant.RESOURCE_FILES + "." + constant.MANDATORY, util.ResourceFiles_Mandatory)
	viper.SetDefault(constant.RESOURCE_FILES + "." + constant.OPTIONAL, util.ResourceFiles_Optional)
	viper.SetDefault(constant.RESOURCE_FILES + "." + constant.SKIP, util.ResourceFiles_Skip)
}
