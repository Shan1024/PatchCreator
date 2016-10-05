// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package cmd

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mholt/archiver"
	"github.com/olekukonko/tablewriter"
	"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wso2/wum-uc/constant"
	"github.com/wso2/wum-uc/util"
	"gopkg.in/yaml.v2"
)

type Data struct {
	name         string
	isDir        bool
	relativePath string
	md5          string
}
type Node struct {
	name             string
	isDir            bool
	relativeLocation string
	parent           *Node
	childNodes       map[string]*Node
	md5Hash          string
}

func CreateNewNode() Node {
	return Node{
		childNodes: make(map[string]*Node),
	}
}

var (
	createCmdUse = "create <update_dir> <dist_loc>"
	createCmdShortDesc = "Create a new update"
	createCmdLongDesc = dedent.Dedent(`
		This command will create a new update zip file from the files in the
		given directory. To generate the directory structure, it requires the
		product distribution zip file.`)
)

//createCmd represents the create command
var createCmd = &cobra.Command{
	Use: createCmdUse,
	Short: createCmdShortDesc,
	Long: createCmdLongDesc,
	Run: initializeCreateCommand,
}

func init() {
	RootCmd.AddCommand(createCmd)

	createCmd.Flags().BoolVarP(&isDebugLogsEnabled, "debug", "d", util.EnableDebugLogs, "Enable debug logs")
	createCmd.Flags().BoolVarP(&isTraceLogsEnabled, "trace", "t", util.EnableTraceLogs, "Enable trace logs")

	createCmd.Flags().BoolP("validate", "v", util.AutoValidate, "Disable validating the content of update zip")
	viper.BindPFlag(constant.AUTO_VALIDATE, createCmd.Flags().Lookup("validate"))

	createCmd.Flags().BoolP("repository", "r", viper.GetBool(constant.UPDATE_REPOSITORY_ENABLED), "Enable/Disable repository")
	viper.BindPFlag(constant.UPDATE_REPOSITORY_ENABLED, createCmd.Flags().Lookup("repository"))

	createCmd.Flags().StringP("location", "l", viper.GetString(constant.UPDATE_REPOSITORY_LOCATION), "Override repository location")
	viper.BindPFlag(constant.UPDATE_REPOSITORY_LOCATION, createCmd.Flags().Lookup("location"))

	createCmd.Flags().BoolP("md5", "m", util.CheckMd5, "Disable checking MD5 sum")
	viper.BindPFlag(constant.CHECK_MD5, createCmd.Flags().Lookup("md5"))
}

func initializeCreateCommand(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		util.PrintErrorAndExit("Invalid number of argumants. Run 'wum-uc create --help' to view help.")
	}
	createUpdate(args[0], args[1])
}

//main execution path
func createUpdate(updateDirectoryPath, distributionPath string) {

	//set debug level
	setLogLevel()
	logger.Debug("[create] command called")

	//Flow - First check whether the given locations exists and required files exists. Then start processing.
	//If one step fails, print error message and exit.

	//1) Check whether the given update directory exists
	exists, err := util.IsDirectoryExists(updateDirectoryPath)
	util.HandleError(err, "Error occurred while reading the update directory")
	if !exists {
		util.PrintErrorAndExit(fmt.Sprintf("Update directory (%s) does not exist.", updateDirectoryPath))
	}
	updateRoot := strings.TrimSuffix(updateDirectoryPath, "/")
	updateRoot = strings.TrimSuffix(updateRoot, "\\")
	logger.Debug(fmt.Sprintf("updateRoot: %s\n", updateRoot))
	viper.Set(constant.UPDATE_ROOT, updateRoot)

	//2) Check whether the update-descriptor.yaml file exists
	//Construct the update-descriptor.yaml file location
	updateDescriptorPath := path.Join(updateDirectoryPath, constant.UPDATE_DESCRIPTOR_FILE)
	exists, err = util.IsFileExists(updateDescriptorPath)
	util.HandleError(err, "")
	if !exists {
		util.PrintError(fmt.Sprintf("'%s' not found at '%s'.", constant.UPDATE_DESCRIPTOR_FILE, updateDirectoryPath))
		util.PrintWhatsNext(fmt.Sprintf("Run 'wum-uc init %s' to generate the '%s' template in the update location.", updateDirectoryPath, constant.UPDATE_DESCRIPTOR_FILE))
		os.Exit(1)
	}
	logger.Debug(fmt.Sprintf("Descriptor Exists. Location %s", updateDescriptorPath))

	//3) Check whether the given distribution exists
	exists, err = util.IsFileExists(distributionPath)
	util.HandleError(err, "Distribution path '" + distributionPath + "' does not exist")
	if !exists {
		util.PrintErrorAndExit(fmt.Sprintf("Distribution does not exist at '%s'", distributionPath))
	}
	if !strings.HasSuffix(distributionPath, ".zip") {
		util.PrintErrorAndExit(fmt.Sprintf("Entered update location(%s) does not have a 'zip' extention.", distributionPath))
	}
	//4) Read update-descriptor.yaml and set the update name which will be used when creating the update zip file.
	//This is used to read the update-descriptor.yaml file
	updateDescriptor, err := util.LoadUpdateDescriptor(constant.UPDATE_DESCRIPTOR_FILE, updateDirectoryPath)
	util.HandleError(err, fmt.Sprintf("Error occurred when reading '%s' file.", constant.UPDATE_DESCRIPTOR_FILE))

	//Validate the file format
	err = util.ValidateUpdateDescriptor(updateDescriptor)
	util.HandleError(err, "'" + constant.UPDATE_DESCRIPTOR_FILE + "' format is not correct")

	//set the update name
	updateName := util.GetUpdateName(updateDescriptor, constant.UPDATE_NAME_PREFIX)
	viper.Set(constant.UPDATE_NAME, updateName)

	//5) Traverse and read the update
	ignoredFiles := getIgnoredFilesInUpdate()
	//updateLocationInfo := FileLocationInfo{
	//	nameLocationInfoMap: make(map[string]LocationInfo),
	//}
	allFilesMap, rootLevelDirectoriesMap, rootLevelFilesMap, err := readDirectory(updateDirectoryPath, ignoredFiles)
	util.HandleError(err, "")

	logger.Trace(fmt.Sprintf("allFilesMap: %s\n", allFilesMap))
	logger.Trace(fmt.Sprintf("directoryList: %s\n", rootLevelDirectoriesMap))
	logger.Trace(fmt.Sprintf("fileList: %s\n", rootLevelFilesMap))

	//err = readDirectoryStructure(updateDirectoryPath, &updateLocationInfo, ignoredFiles, true)
	//logger.Debug("updateLocationInfo:", updateLocationInfo)

	rootNode := CreateNewNode()
	if !strings.HasSuffix(distributionPath, ".zip") {
		util.PrintErrorAndExit(fmt.Sprintf("Entered distribution path(%s) does not point to a zip file.", distributionPath))
	}

	paths := strings.Split(distributionPath, constant.PATH_SEPARATOR)
	distributionName := strings.TrimSuffix(paths[len(paths) - 1], ".zip")
	viper.Set(constant.PRODUCT_NAME, distributionName)

	logger.Debug("Reading zip")
	util.PrintInfo(fmt.Sprintf("Reading %s. Please wait...", distributionName))
	rootNode, err = readZip(distributionPath)
	util.HandleError(err)
	logger.Debug("Reading zip finished")

	logger.Debug("-------------------------------------")
	for name, node := range rootNode.childNodes {
		logger.Debug(name, ":", node)
	}
	logger.Debug("-------------------------------------")

	//todo: save the selected location to generate the final summary map
	//7) Find matches and copy files to the temp
	logger.Debug("\nChecking Directories:")
	matches := make(map[string]*Node)
	for directoryName := range rootLevelDirectoriesMap {
		matches = make(map[string]*Node)
		logger.Debug(fmt.Sprintf("DirectoryName: %s", directoryName))
		FindMatches(&rootNode, directoryName, true, matches)
		logger.Debug(fmt.Sprintf("matches: %v", matches))

		switch len(matches){
		case 0:
			logger.Debug("\nNo match found\n")
			err := handleNoMatch(directoryName, true, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		case 1:
			logger.Debug("\nSingle match found\n")
			var match *Node
			for _, node := range matches {
				match = node
			}
			err := handleSingleMatch(directoryName, match, true, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		default:
			logger.Debug("\nMultiple matches found\n")
			err := handleMultipleMatches(directoryName, true, matches, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		}
	}

	logger.Debug("\nChecking Files:")
	for fileName := range rootLevelFilesMap {
		matches = make(map[string]*Node)
		logger.Debug(fmt.Sprintf("FileName: %s", fileName))
		FindMatches(&rootNode, fileName, false, matches)
		logger.Debug(fmt.Sprintf("matches: %v", matches))

		switch len(matches){
		case 0:
			logger.Debug("\nNo match found\n")
			err := handleNoMatch(fileName, false, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		case 1:
			logger.Debug("\nSingle match found\n")
			var match *Node
			for _, node := range matches {
				match = node
			}
			err := handleSingleMatch(fileName, match, false, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		default:
			logger.Debug("\nMultiple matches found\n")
			err := handleMultipleMatches(fileName, false, matches, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		}
	}

	//8) Copy resource files (update-descriptor.yaml, etc)
	resourceFiles := getResourceFiles()
	err = copyResourceFiles(resourceFiles)
	util.HandleError(err, errors.New("Error occurred while copying resource files."))

	//Save the update-descriptor with the updated, newly added files to the temp directory
	data, err := marshalUpdateDescriptor(updateDescriptor)
	util.HandleError(err)
	err = saveUpdateDescriptor(constant.UPDATE_DESCRIPTOR_FILE, data)
	util.HandleError(err)

	updateZipName := updateName + ".zip"
	logger.Debug(fmt.Sprintf("Update repository is enabled: %v", viper.GetBool(constant.UPDATE_REPOSITORY_ENABLED)))
	//9) Create the update zip file
	if viper.GetBool(constant.UPDATE_REPOSITORY_ENABLED) {
		updateRepository := viper.GetString(constant.UPDATE_REPOSITORY_LOCATION)
		if updateRepository[:2] == "~/" {
			updateRepository = filepath.Join(util.HomeDirectory, updateRepository[2:])
		}
		logger.Debug(fmt.Sprintf("Update repository location is set to: %s", updateRepository))
		exists, err := util.IsDirectoryExists(updateRepository)
		util.HandleError(err)
		if !exists {
			err = util.CreateDirectory(updateRepository)
			util.HandleError(err)
		}
		updateZipName = path.Join(updateRepository, updateZipName)
	}

	logger.Debug(fmt.Sprintf("updateZipName: %s", updateZipName))
	err = archiver.Zip(updateZipName, []string{path.Join(constant.TEMP_DIR, updateName)})
	util.HandleError(err)

	//Remove the temp directories
	util.CleanUpDirectory(constant.TEMP_DIR)

	util.PrintInfo(fmt.Sprintf("'%s' successfully created.", updateZipName))
	if !viper.GetBool(constant.AUTO_VALIDATE) {
		util.PrintInfo(fmt.Sprintf("Validating '%s'\n", updateZipName))
		startValidation(updateZipName, distributionPath)
	} else {
		util.PrintWhatsNext(fmt.Sprintf("Validate the update zip after any manual modification by running 'wum-uc validate %s.zip %s'", updateZipName, distributionPath))
	}
}

func handleNoMatch(filename string, isDir bool, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug(fmt.Sprintf("[NO MATCH] %s", filename))
	util.PrintInBold(fmt.Sprintf("'%s' not found in distribution. ", filename))
	for {
		util.PrintInBold("Do you want to add it as a new file? [y/N]: ")
		preference, err := util.GetUserInput()
		if len(preference) == 0 {
			preference = "n"
		}
		util.HandleError(err, "Error occurred while getting input from the user.")
		if util.IsYes(preference) {
			err = handleNewFile(filename, isDir, rootNode, allFilesMap, updateDescriptor)
			util.HandleError(err)
			//If no error, return nil
			return nil
		} else if util.IsNo(preference) {
			util.PrintWarning(fmt.Sprintf("Skipping copying: %s", filename))
			return nil
		} else {
			util.PrintError("Invalid preference. Enter Y for Yes or N for No.")
		}
	}
	return nil
}

func handleNewFile(filename string, isDir bool, rootNode *Node, allFilesMap map[string]Data, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug(fmt.Sprintf("[HANDLE NEW] %s", filename))

	readDestinationLoop:
	for {
		util.PrintInBold("Enter destination directory relative to CARBON_HOME: ")
		relativeLocationInDistribution, err := util.GetUserInput()
		relativeLocationInDistribution = strings.TrimPrefix(relativeLocationInDistribution, constant.PATH_SEPARATOR)
		relativeLocationInDistribution = strings.TrimSuffix(relativeLocationInDistribution, constant.PATH_SEPARATOR)
		util.HandleError(err, "Error occurred while getting input from the user.")
		logger.Debug("relativePath:", relativeLocationInDistribution)

		updateRoot := viper.GetString(constant.UPDATE_ROOT)
		var exists bool
		if isDir {
			fullPath := path.Join(relativeLocationInDistribution, filename)
			logger.Debug(fmt.Sprintf("Checking: %s", fullPath))
			exists = PathExists(rootNode, fullPath, true)
			logger.Debug(fmt.Sprintf("%s exists: %s", fullPath, exists))
		} else {
			logger.Debug("Checking:", relativeLocationInDistribution)
			exists = PathExists(rootNode, relativeLocationInDistribution, true)
			logger.Debug(relativeLocationInDistribution + " exists:", exists)
		}

		if exists {
			if isDir {
				allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
				logger.Debug(fmt.Sprintf("All matches: %v", allMatchingFiles))
				for _, match := range allMatchingFiles {
					logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", match, updateRoot, relativeLocationInDistribution))
					err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
					util.HandleError(err)
				}
			} else {
				logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", filename, updateRoot, relativeLocationInDistribution))
				err = copyFile(filename, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)
			}
			break
		} else if len(relativeLocationInDistribution) > 0 {
			util.PrintInBold("Entered relative path does not exist in the distribution. ")
			for {
				util.PrintInBold("Copy anyway? [y/n/R]: ")
				preference, err := util.GetUserInput()
				if len(preference) == 0 {
					preference = "r"
				}
				util.HandleError(err, "Error occurred while getting input from the user.")
				if util.IsYes(preference) {
					updateRoot := viper.GetString(constant.UPDATE_ROOT)
					allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
					logger.Debug(fmt.Sprintf("Copying all matches:\n%s", allMatchingFiles))
					for _, match := range allMatchingFiles {
						logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", match, updateRoot, relativeLocationInDistribution))
						err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
						util.HandleError(err)
					}
					break readDestinationLoop
				} else if util.IsNo(preference) {
					util.PrintWarning("Skipping copying", filename)
					return nil
				} else if util.IsReenter(preference) {
					break
				} else {
					util.PrintError("Invalid preference. Enter Y for Yes or N for No or R for Re-enter.")
				}
			}
		} else {
			updateRoot := viper.GetString(constant.UPDATE_ROOT)
			allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
			logger.Debug(fmt.Sprintf("Copying all matches:\n%s", allMatchingFiles))
			for _, match := range allMatchingFiles {
				logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", match, updateRoot, relativeLocationInDistribution))
				err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)
			}
			break readDestinationLoop
		}
	}
	return nil
}

func handleSingleMatch(filename string, matchingNode *Node, isDir bool, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug(fmt.Sprintf("[SINGLE MATCH] %s ; match: %s", filename, matchingNode.relativeLocation))
	updateRoot := viper.GetString(constant.UPDATE_ROOT)
	if isDir {
		allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
		logger.Debug(fmt.Sprintf("All matches: %s", allMatchingFiles))
		for _, match := range allMatchingFiles {

			logger.Debug(fmt.Sprintf("match: %s", match))
			if !viper.GetBool(constant.CHECK_MD5) {
				data := allFilesMap[match]
				md5Matches := CompareMD5(rootNode, path.Join(matchingNode.relativeLocation, match), data.md5)
				if md5Matches {
					logger.Debug("MD5 matches. Ignoring file.")
					continue
				} else {
					logger.Debug("MD5 does not match. Copying the file.")
				}
			}
			logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", match, updateRoot, matchingNode.relativeLocation))
			err := copyFile(match, updateRoot, matchingNode.relativeLocation, rootNode, updateDescriptor)
			util.HandleError(err)
		}
	} else {
		logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", filename, updateRoot, matchingNode.relativeLocation))
		err := copyFile(filename, updateRoot, matchingNode.relativeLocation, rootNode, updateDescriptor)
		util.HandleError(err)
	}
	return nil
}

func handleMultipleMatches(filename string, isDir bool, matches map[string]*Node, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {

	util.PrintInfo(fmt.Sprintf("Multiple matches found for '%s' in the distribution.", filename))

	logger.Debug(fmt.Sprintf("[MULTIPLE MATCHES] %s", filename))
	locationTable, indexMap := generateLocationTable(filename, matches)
	locationTable.Render()
	logger.Debug(fmt.Sprintf("indexMap: %s", indexMap))
	skipCopying := false
	var selectedIndices []string
	for {
		util.PrintInBold("Enter preference(s)[Multiple selections separated by commas, 0 to skip copying]: ")
		preferences, err := util.GetUserInput()
		util.HandleError(err)
		logger.Debug(fmt.Sprintf("preferences: %s", preferences))
		//Remove the new line at the end
		preferences = strings.TrimSpace(preferences)
		//Split the indices
		selectedIndices = strings.Split(preferences, ",");
		//Sort the locations
		sort.Strings(selectedIndices)
		logger.Debug(fmt.Sprintf("sorted: %s", preferences))

		length := len(indexMap)
		isValid, err := util.IsUserPreferencesValid(selectedIndices, length)
		if err != nil {
			util.PrintError("Invalid preferences. Please select indices where 0 <= index <= " + strconv.Itoa(length))
			continue
		}
		if !isValid {
			util.PrintError("Invalid preferences. Please select indices where 0 <= index <= " + strconv.Itoa(length))
		} else {
			logger.Debug("Entered preferences are valid.")
			if selectedIndices[0] == "0" {
				skipCopying = true
			}
			break
		}
	}
	if skipCopying {
		logger.Debug(fmt.Sprintf("Skipping copying '%s'", filename))
		util.PrintWarning(fmt.Sprintf("0 entered. Skipping copying '%s'.", filename))
		return nil
	}
	updateRoot := viper.GetString(constant.UPDATE_ROOT)
	if isDir {
		for _, selectedIndex := range selectedIndices {
			pathInDistribution := indexMap[selectedIndex]
			logger.Debug(fmt.Sprintf("[MULTIPLE MATCHES] Selected path: %s ; %s", selectedIndex, pathInDistribution))

			allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
			logger.Debug(fmt.Sprintf("matchingFiles: %s", allMatchingFiles))

			for _, match := range allMatchingFiles {

				logger.Debug(fmt.Sprintf("match: %s", match))
				if !viper.GetBool(constant.CHECK_MD5) {
					data := allFilesMap[match]
					md5Matches := CompareMD5(rootNode, path.Join(pathInDistribution, match), data.md5)
					if md5Matches {
						logger.Debug("MD5 matches. Ignoring file.")
						continue
					}
					logger.Debug("MD5 does not match. Copying the file.")
				}
				logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", filename, updateRoot, pathInDistribution))
				err := copyFile(match, updateRoot, pathInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)
			}
		}
	} else {
		for _, selectedIndex := range selectedIndices {
			pathInDistribution := indexMap[selectedIndex]
			logger.Debug(fmt.Sprintf("[MULTIPLE MATCHES] Selected path: %s ; %s", selectedIndex, pathInDistribution))
			logger.Debug(fmt.Sprintf("[Copy] %s ; From: %s ; To: %s", filename, updateRoot, pathInDistribution))
			err := copyFile(filename, updateRoot, pathInDistribution, rootNode, updateDescriptor)
			util.HandleError(err)
		}
	}
	return nil
}

func getAllMatchingFiles(filepath string, allFilesMap map[string]Data) []string {
	matches := make([]string, 0)
	for filename, data := range allFilesMap {
		if !data.isDir && strings.HasPrefix(filename, filepath) && filename != filepath {
			matches = append(matches, filename)
		}
	}
	return matches
}

func readDirectory(root string, ignoredFiles map[string]bool) (map[string]Data, map[string]bool, map[string]bool, error) {
	allFilesMap := make(map[string]Data)
	rootLevelDirectoriesMap := make(map[string]bool)
	rootLevelFilesMap := make(map[string]bool)

	filepath.Walk(root, func(absolutePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//Ignore root directory
		if root == absolutePath {
			return nil
		}
		logger.Trace(fmt.Sprintf("[WALK] %s ; %s", absolutePath, fileInfo.IsDir()))
		//check current file in ignored files map. This is useful to ignore update-descriptor.yaml, etc in update directory
		if ignoredFiles != nil {
			_, found := ignoredFiles[fileInfo.Name()]
			if found {
				return nil
			}
		}
		relativePath := strings.TrimPrefix(absolutePath, root + "/")
		info := Data{
			name: fileInfo.Name(),
			relativePath: relativePath,
		}
		if fileInfo.IsDir() {
			logger.Trace(fmt.Sprintf("Directory: %s , %s", absolutePath, fileInfo.Name()))
			info.isDir = true
			if path.Join(root, fileInfo.Name()) == absolutePath {
				rootLevelDirectoriesMap[fileInfo.Name()] = true
			}
		} else {
			if path.Join(root, fileInfo.Name()) == absolutePath {
				rootLevelFilesMap[fileInfo.Name()] = false
			}
			logger.Trace("[MD5] Calculating MD5")
			//If it is a file, calculate md5 sum
			md5Sum, err := util.GetMD5(absolutePath)
			if err != nil {
				return err
			}
			logger.Trace(fmt.Sprintf("%s : %s = %s", absolutePath, fileInfo.Name(), md5Sum))
			info.md5 = md5Sum
			info.isDir = false
		}
		allFilesMap[relativePath] = info
		return nil
	})
	return allFilesMap, rootLevelDirectoriesMap, rootLevelFilesMap, nil
}

func readZip(filename string) (Node, error) {
	rootNode := CreateNewNode()
	fileMap := make(map[string]bool)
	// Create a reader out of the zip archive
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return rootNode, err
	}
	defer zipReader.Close()

	productName := viper.GetString(constant.PRODUCT_NAME)
	logger.Debug(fmt.Sprintf("productName: %s", productName))
	// Iterate through each file/dir found in
	for _, file := range zipReader.Reader.File {
		zippedFile, err := file.Open()
		if err != nil {
			return rootNode, err
		}
		data, err := ioutil.ReadAll(zippedFile)
		zippedFile.Close()

		hash := md5.New()
		hash.Write(data)
		md5Hash := hex.EncodeToString(hash.Sum(nil))

		logger.Trace(file.Name)
		relativePath := strings.TrimPrefix(file.Name, productName + "/")
		logger.Trace(relativePath)
		AddToRootNode(&rootNode, strings.Split(relativePath, "/"), file.FileInfo().IsDir(), md5Hash)
		if !file.FileInfo().IsDir() {
			fileMap[relativePath] = false
		}
	}
	return rootNode, nil
}

func AddToRootNode(root *Node, path []string, isDir bool, md5Hash string) *Node {
	logger.Trace("Checking: %s : %s", path[0], path)
	if len(path) == 1 {
		logger.Trace("End reached")
		newNode := CreateNewNode()
		newNode.name = path[0]
		newNode.isDir = isDir
		newNode.md5Hash = md5Hash
		if len(root.relativeLocation) == 0 {
			newNode.relativeLocation = path[0]
		} else {
			newNode.relativeLocation = root.relativeLocation + "/" + path[0]
		}
		newNode.parent = root
		root.childNodes[path[0]] = &newNode

	} else {
		Node, contains := root.childNodes[path[0]]
		if contains {
			logger.Trace("Already exists")
			AddToRootNode(Node, path[1:], isDir, md5Hash)
		} else {
			logger.Trace("New node")
			newNode := CreateNewNode()
			newNode.name = path[0]
			newNode.isDir = isDir
			newNode.md5Hash = md5Hash
			if len(root.relativeLocation) == 0 {
				newNode.relativeLocation = path[0]
			} else {
				newNode.relativeLocation = root.relativeLocation + "/" + path[0]
			}
			newNode.parent = root
			root.childNodes[path[0]] = &newNode
		}
	}
	return root
}

func PathExists(rootNode *Node, relativePath string, isDir bool) bool {
	return NodeExists(rootNode, strings.Split(relativePath, "/"), isDir)
}

func NodeExists(rootNode *Node, path []string, isDir bool) bool {
	logger.Trace(fmt.Sprintf("All: %s", rootNode.childNodes))
	logger.Trace(fmt.Sprintf("Checking: %s", path[0]))
	childNode, found := rootNode.childNodes[path[0]]
	if found {
		logger.Trace(fmt.Sprintf("%s found", path[0]))
		if len(path) > 1 {
			return NodeExists(childNode, path[1:], isDir)
		} else {
			return childNode.isDir == isDir
		}
	}
	logger.Trace(fmt.Sprintf("%s NOT found", path[0]))
	return false
}

func CompareMD5(rootNode *Node, relativePath string, md5 string) bool {
	logger.Trace(fmt.Sprintf("Checking MD5: %s", relativePath))
	return CheckMD5(rootNode, strings.Split(relativePath, "/"), md5)
}

func CheckMD5(rootNode *Node, path []string, md5 string) bool {
	logger.Trace(fmt.Sprintf("All: %s", rootNode.childNodes))
	logger.Trace(fmt.Sprintf("Checking: %s", path[0]))
	childNode, found := rootNode.childNodes[path[0]]
	if found {
		logger.Trace(fmt.Sprintf("%s found", path[0]))
		if len(path) > 1 {
			return CheckMD5(childNode, path[1:], md5)
		} else {
			return childNode.isDir == false && childNode.md5Hash == md5
		}
	}
	logger.Trace(fmt.Sprintf("%s NOT found", path[0]))
	return false
}

func FindMatches(root *Node, name string, isDir bool, matches map[string]*Node) {
	childNode, found := root.childNodes[name]
	if found {
		if isDir == childNode.isDir {
			matches[root.relativeLocation] = root
		}
	}
	for _, childNode := range root.childNodes {
		if childNode.isDir {
			FindMatches(childNode, name, isDir, matches)
		}
	}
}

//This will return a map of files which would be ignored when reading the update directory
func getIgnoredFilesInUpdate() map[string]bool {
	filesMap := make(map[string]bool)
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES_MANDATORY) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES_OPTIONAL) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES_SKIP) {
		filesMap[file] = true
	}
	return filesMap
}

//This will return a map of files which would be copied to the temp directory before creating the update zip. Key is the
// file name and value is whether the file is mandatory or not.
func getResourceFiles() map[string]bool {
	filesMap := make(map[string]bool)
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES_MANDATORY) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES_OPTIONAL) {
		filesMap[file] = false
	}
	return filesMap
}

func marshalUpdateDescriptor(updateDescriptor *util.UpdateDescriptor) ([]byte, error) {
	data, err := yaml.Marshal(&updateDescriptor)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func saveUpdateDescriptor(updateDescriptorFilename string, data []byte) error {
	updateName := viper.GetString(constant.UPDATE_NAME)
	destination := path.Join(constant.TEMP_DIR, updateName, updateDescriptorFilename)
	// Open a new file for writing only
	file, err := os.OpenFile(
		destination,
		os.O_WRONLY | os.O_TRUNC | os.O_CREATE,
		0600,
	)
	defer file.Close()
	if err != nil {
		return err
	}

	updatedData := strings.Replace(string(data), "\"", "", 2)
	modifiedData := []byte(updatedData)
	// Write bytes to file
	_, err = file.Write(modifiedData)
	if err != nil {
		return err
	}
	return nil
}

func copyResourceFiles(resourceFilesMap map[string]bool) error {
	//Create the directories if they are not available
	updateName := viper.GetString(constant.UPDATE_NAME)
	destination := path.Join(constant.TEMP_DIR, updateName, constant.CARBON_HOME)
	util.CreateDirectory(destination)
	for filename, isMandatory := range resourceFilesMap {
		updateRoot := viper.GetString(constant.UPDATE_ROOT)
		updateName := viper.GetString(constant.UPDATE_NAME)
		source := path.Join(updateRoot, filename)
		destination := path.Join(constant.TEMP_DIR, updateName, filename)
		err := util.CopyFile(source, destination)
		if err != nil {
			if isMandatory {
				return err
			} else {
				util.PrintInfo(fmt.Sprintf("Optional resource file '%s' not found.", filename))
			}
		}
	}
	return nil
}

//This will generate the location table and the index map which will be used to get user preference
func generateLocationTable(filename string, locationsInDistribution map[string]*Node) (*tablewriter.Table, map[string]string) {
	locationTable := tablewriter.NewWriter(os.Stdout)
	locationTable.SetAlignment(tablewriter.ALIGN_LEFT)
	locationTable.SetHeader([]string{"Index", "Matching Location"})

	allPaths := make([]string, 0)
	for distributionFilepath, _ := range locationsInDistribution {
		allPaths = append(allPaths, distributionFilepath)
	}
	sort.Strings(allPaths)

	index := 1
	indexMap := make(map[string]string)
	for _, distributionFilepath := range allPaths {
		logger.Debug(fmt.Sprintf("[TABLE] filepath: %s ; isDir: %s", distributionFilepath, locationsInDistribution[distributionFilepath].isDir))
		indexMap[strconv.Itoa(index)] = distributionFilepath
		relativePath := path.Join("CARBON_HOME", distributionFilepath)
		locationTable.Append([]string{strconv.Itoa(index), path.Join(relativePath, filename)})
		index++
	}
	return locationTable, indexMap
}

////This function will copy the file/directory from update to temp location
func copyFile(filename string, locationInUpdate, relativeLocationInTemp string, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug(fmt.Sprintf("[FINAL][COPY ROOT] Name: %s ; IsDir: false ; From: %s ; To: %s", filename, locationInUpdate, relativeLocationInTemp))
	updateName := viper.GetString(constant.UPDATE_NAME)
	source := path.Join(locationInUpdate, filename)
	carbonHome := path.Join(constant.TEMP_DIR, updateName, constant.CARBON_HOME)
	destination := path.Join(carbonHome, relativeLocationInTemp)

	fullPath := path.Join(destination, filename)
	err := util.CreateDirectory(util.GetParentDirectory(fullPath))
	util.HandleError(err)
	logger.Debug(fmt.Sprintf("[FINAL][COPY][TEMP] Name: %s; From: %s; To: %s", filename, source, fullPath))
	err = util.CopyFile(source, fullPath)
	util.HandleError(err, "temp2")

	relativePath := strings.TrimPrefix(fullPath, carbonHome + constant.PATH_SEPARATOR)
	logger.Debug(fmt.Sprintf("relativePath: %s", relativePath))
	contains := PathExists(rootNode, relativePath, false)
	logger.Debug(fmt.Sprintf("contains: %s", contains))
	if contains {
		updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, relativePath)
	} else {
		updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, relativePath)
	}
	return nil
}
