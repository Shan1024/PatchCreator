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

func (node *Node) PrintNode() {
	fmt.Println(node.name, "(", node.isDir, "):", node.relativeLocation)
	fmt.Println(node.childNodes)
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
		given directory. To generate the folder structure, it requires the
		product distribution also. This distribution can either be the zip
		file or the extracted directory.`)
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

	createCmd.Flags().BoolP("debug", "d", false, "Enable debug logs")
	viper.BindPFlag(constant.IS_DEBUG_ENABLED, createCmd.Flags().Lookup("debug"))

	createCmd.Flags().BoolP("trace", "t", false, "Enable trace logs")
	viper.BindPFlag(constant.IS_TRACE_ENABLED, createCmd.Flags().Lookup("trace"))

	createCmd.Flags().BoolP("validate", "v", viper.GetBool(constant.AUTO_VALIDATE), "Enable/Disable validating the content of update zip")
	viper.BindPFlag(constant.AUTO_VALIDATE, createCmd.Flags().Lookup("validate"))

	createCmd.Flags().BoolP("repository", "r", viper.GetBool(constant.UPDATE_REPOSITORY_ENABLED), "Enable/Disable repository")
	viper.BindPFlag(constant.UPDATE_REPOSITORY_ENABLED, createCmd.Flags().Lookup("repository"))

	createCmd.Flags().StringP("location", "l", viper.GetString(constant.UPDATE_REPOSITORY_LOCATION), "Override repository location")
	viper.BindPFlag(constant.UPDATE_REPOSITORY_LOCATION, createCmd.Flags().Lookup("location"))

	createCmd.Flags().BoolP("md5", "m", viper.GetBool(constant.CHECK_MD5), "Enable/Disable checking MD5 sum")
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
	logger.Debug("<create> command called")

	//Flow - First check whether the given locations exists and required files exists. Then start processing.
	//If one step fails, print error message and exit.

	//1) Check whether the given update directory exists
	exists, err := util.IsDirectoryExists(updateDirectoryPath)
	//todo: look for best practice
	util.HandleError(err, "Error occurred while reading the update directory")
	if !exists {
		util.PrintErrorAndExit("Update directory '" + updateDirectoryPath + "' does not exist.")
	}
	updateRoot := strings.TrimSuffix(updateDirectoryPath, "/")
	updateRoot = strings.TrimSuffix(updateRoot, "\\")
	fmt.Printf("updateRoot: %s\n", updateRoot)
	viper.Set(constant.UPDATE_ROOT, updateRoot)

	//2) Check whether the update-descriptor.yaml file exists
	//Construct the update-descriptor.yaml file location
	updateDescriptorPath := path.Join(updateDirectoryPath, constant.UPDATE_DESCRIPTOR_FILE)
	exists, err = util.IsFileExists(updateDescriptorPath)
	util.HandleError(err, "")
	if !exists {
		util.PrintError("'" + constant.UPDATE_DESCRIPTOR_FILE + "' not found at '" + updateDirectoryPath + "'.")
		util.PrintWhatsNext("Run 'wum-uc init " + updateDirectoryPath + "' to generate the '" + constant.UPDATE_DESCRIPTOR_FILE + "' template in the update location.")
		os.Exit(1)
	}
	logger.Debug("Descriptor Exists. Location %s", updateDescriptorPath)

	//3) Check whether the given distribution exists
	exists, err = util.IsFileExists(distributionPath)
	util.HandleError(err, "Distribution path '" + distributionPath + "' does not exist")
	if !exists {
		util.PrintErrorAndExit("Distribution does not exist at ", updateDirectoryPath)
	}
	//4) Read update-descriptor.yaml and set the update name which will be used when creating the update zip file.
	//This is used to read the update-descriptor.yaml file
	updateDescriptor, err := util.LoadUpdateDescriptor(constant.UPDATE_DESCRIPTOR_FILE, updateDirectoryPath)
	util.HandleError(err, "Error occurred when reading '" + constant.UPDATE_DESCRIPTOR_FILE + "' file")

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

	fmt.Println("++++++++++++++++++++++++++++++++++++++")
	fmt.Println("allFilesMap:", allFilesMap)
	fmt.Println("++++++++++++++++++++++++++++++++++++++")
	fmt.Println("directoryList:", rootLevelDirectoriesMap)
	fmt.Println("++++++++++++++++++++++++++++++++++++++")
	fmt.Println("fileList:", rootLevelFilesMap)
	fmt.Println("++++++++++++++++++++++++++++++++++++++")

	//err = readDirectoryStructure(updateDirectoryPath, &updateLocationInfo, ignoredFiles, true)
	//logger.Debug("updateLocationInfo:", updateLocationInfo)

	rootNode := CreateNewNode()
	if !strings.HasSuffix(distributionPath, ".zip") {
		fmt.Println("Entered path is not a zip file")
	}

	paths := strings.Split(distributionPath, constant.PATH_SEPARATOR)
	distributionName := strings.TrimSuffix(paths[len(paths) - 1], ".zip")
	viper.Set(constant.PRODUCT_NAME, distributionName)

	fmt.Println("Reading zip")
	rootNode, err = readZip(distributionPath)
	util.HandleError(err)
	fmt.Println("Reading zip finished")

	//cleanupChannel := util.HandleInterrupts(func() {
	//	logger.Debug("Cleaning up distributionRoot directory")
	//	util.CleanUpDirectory(distributionRoot)
	//})

	//rootNode.PrintNode()
	fmt.Println("-------------------------------------")
	for name, node := range rootNode.childNodes {
		fmt.Println(name, ":", node)
		fmt.Println("-------------------------------------")
	}

	//7) Find matches and copy files to the temp


	fmt.Println("=========================================")
	fmt.Println("\n\nChecking Directories:")

	matches := make(map[string]*Node)

	for directoryName := range rootLevelDirectoriesMap {
		matches = make(map[string]*Node)
		fmt.Println("DirectoryName:", directoryName)
		FindMatches(&rootNode, directoryName, true, matches)
		fmt.Println("matches:", matches)

		switch len(matches){
		case 0:
			fmt.Print("\n\nNo match found\n\n")

			err := handleNoMatch(directoryName, true, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		case 1:
			fmt.Print("\n\nSingle match found\n")

			var match *Node

			for _, node := range matches {
				match = node
			}
			err := handleSingleMatch(directoryName, match, true, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)

		default:
			fmt.Print("\n\nMultiple matches found\n\n")

			err := handleMultipleMatches(directoryName, true, matches, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		}

	}

	fmt.Println("\n\nChecking Files:")

	for fileName := range rootLevelFilesMap {
		matches = make(map[string]*Node)
		fmt.Println("FileName:", fileName)
		FindMatches(&rootNode, fileName, false, matches)
		fmt.Println("matches:", matches)

		switch len(matches){
		case 0:
			fmt.Print("\n\nNo match found\n\n")

			err := handleNoMatch(fileName, false, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)
		case 1:
			fmt.Print("\n\nSingle match found\n")

			var match *Node

			for _, node := range matches {
				match = node
			}
			err := handleSingleMatch(fileName, match, false, allFilesMap, &rootNode, updateDescriptor)
			util.HandleError(err)

		default:
			fmt.Print("\n\nMultiple matches found\n\n")

			//todo
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

	//cleanupChannel := util.HandleInterrupts(func() {
	//	logger.Debug("Cleaning up temp directory")
	//	util.CleanUpDirectory(constant.TEMP_DIR)
	//})

	updateZipName := updateName + ".zip"

	//9) Create the update zip file
	if viper.IsSet(constant.UPDATE_REPOSITORY_ENABLED) {

		fmt.Println("############### Update repository is enabled")

		updateRepository := viper.GetString(constant.UPDATE_REPOSITORY_LOCATION)
		util.HandleError(err)
		fmt.Println("Update repository location is set to:", updateRepository)

		err = util.CreateDirectory(updateRepository)
		util.HandleError(err)

		updateZipName = path.Join(updateRepository, updateZipName)
	} else {
		fmt.Println("############### Update repository is disabled")
	}

	err = archiver.Zip(updateZipName, []string{path.Join(constant.TEMP_DIR, updateName)})
	util.HandleError(err)

	//signal.Stop(cleanupChannel)

	//Remove the temp directories
	util.CleanUpDirectory(constant.TEMP_DIR)

	util.PrintInfo("'" + updateZipName + "' successfully created.")

	if viper.GetBool(constant.AUTO_VALIDATE) {
		fmt.Println("\n\nValidating '" + updateZipName + "'\n")
		startValidation(updateZipName, distributionPath)
	} else {
		util.PrintWhatsNext("Validate the update zip after any manual modification by running 'wum-uc validate " + updateName + ".zip " + distributionPath + "'")
	}
}

func handleNoMatch(filename string, isDir bool, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	logger.Debug("[NO MATCH]", filename)
	util.PrintInBold("'" + filename + "' not found in distribution.")
	for {
		util.PrintInBold("Do you want to add it as a new file? [y/N]: ")
		preference, err := util.GetUserInput()
		util.HandleError(err, "Error occurred while getting input from the user.")
		if util.IsYes(preference) {

			err = handleNewFile(filename, isDir, rootNode, allFilesMap, updateDescriptor)
			util.HandleError(err)

			//If no error, return nil
			return nil
		} else if util.IsNo(preference) {
			util.PrintWarning("Skipping copying", filename)
			return nil
		} else {
			util.PrintError("Invalid preference. Enter Y for Yes or N for No.")
		}
	}
	return nil
}

func handleNewFile(filename string, isDir bool, rootNode *Node, allFilesMap map[string]Data, updateDescriptor *util.UpdateDescriptor) error {
	fmt.Println("[HANDLE NEW] Update:", filename)
	//locationInUpdate, isDir, err := getSingleLocationFromMap(locationData.locationsInUpdate)
	//util.HandleError(err)
	//logger.Debug("[HANDLE NEW] Update:", locationInUpdate, ";", isDir)
	readDestinationLoop:
	for {
		util.PrintInBold("Enter destination directory relative to CARBON_HOME: ")
		relativeLocationInDistribution, err := util.GetUserInput()
		relativeLocationInDistribution = strings.TrimPrefix(relativeLocationInDistribution, constant.PATH_SEPARATOR)
		relativeLocationInDistribution = strings.TrimSuffix(relativeLocationInDistribution, constant.PATH_SEPARATOR)
		util.HandleError(err, "Error occurred while getting input from the user.")
		logger.Debug("relativePath:", relativeLocationInDistribution)
		//distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
		updateRoot := viper.GetString(constant.UPDATE_ROOT)
		//fullPath := path.Join(distributionRoot, relativeLocationInDistribution)
		//logger.Debug("fullPath:", fullPath)
		//Ignore error because we are only checking whether the given path exists or not

		var exists bool

		if isDir {
			fullPath := path.Join(relativeLocationInDistribution, filename)

			fmt.Println("Checking:", fullPath)
			exists = PathExists(rootNode, fullPath, true)
			fmt.Println(fullPath + " exists:", exists)
		} else {
			fmt.Println("Checking:", relativeLocationInDistribution)
			exists = PathExists(rootNode, relativeLocationInDistribution, true)
			fmt.Println(relativeLocationInDistribution + " exists:", exists)
		}

		if exists {

			if isDir {
				//fullPath := path.Join(relativeLocationInDistribution, filename)
				//fmt.Println("Checking all files for:", fullPath)
				allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
				fmt.Println("All matches:", allMatchingFiles)
				for _, match := range allMatchingFiles {
					//fmt.Println("match:", match)
					//if viper.GetBool(constant.CHECK_MD5) {
					//	data := allFilesMap[match]
					//	md5Matches := CompareMD5(rootNode, match, data.md5)
					//	if md5Matches {
					//		fmt.Println("MD5 matches. Ignoring file.")
					//		continue
					//	} else {
					//		fmt.Println("MD5 does not match. Copying the file.")
					//	}
					//}
					//fullPath := path.Join(relativeLocationInDistribution, match)
					//
					//fmt.Println("Copying match:",fullPath )
					fmt.Println("[Copy] " + match + " ; From:" + updateRoot + "; To:" + relativeLocationInDistribution)

					err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
					util.HandleError(err)
				}
			} else {
				fmt.Println("[Copy] " + filename + " ; From:" + updateRoot + "; To:" + relativeLocationInDistribution)

				err = copyFile(filename, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)
			}
			break
		} else if len(relativeLocationInDistribution) > 0 {
			util.PrintInBold("Entered relative path does not exist in the distribution. ")
			for {
				util.PrintInBold("Copy anyway? [y/n/R]: ")
				preference, err := util.GetUserInput()
				util.HandleError(err, "Error occurred while getting input from the user.")

				if util.IsYes(preference) {
					//todo: save the selected location to generate the final summary map
					//if isDir {
					//	//distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
					//	//updateUpdateDescriptor(path.Join(locationInUpdate, filename), path.Join(distributionRoot, relativeLocationInDistribution, filename), updateDescriptor)
					//} else {
					//	//newFile := strings.TrimPrefix(path.Join(relativeLocationInDistribution, filename), constant.PATH_SEPARATOR)
					//	//updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, newFile)
					//}


					updateRoot := viper.GetString(constant.UPDATE_ROOT)
					//distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
					allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
					fmt.Print("Copying all matches:\n\n", allMatchingFiles)

					for _, match := range allMatchingFiles {
						//source := path.Join(updateRoot, match)
						//destination := path.Join(distributionRoot, relativeLocationInDistribution)
						fmt.Println("[Copy] " + match + " ; From:" + updateRoot + "; To:" + relativeLocationInDistribution)

						err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
						util.HandleError(err)

					}

					//err = copyFile(filename, isDir, "", relativeLocationInDistribution)
					//util.HandleError(err)
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
			//distributionRoot := viper.GetString(constant.DISTRIBUTION_ROOT)
			allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
			fmt.Print("Copying all matches:\n\n", allMatchingFiles)

			for _, match := range allMatchingFiles {
				//source := path.Join(updateRoot, match)
				//destination := path.Join(distributionRoot, relativeLocationInDistribution)
				fmt.Println("[Copy] " + match + " ; From:" + updateRoot + "; To:" + relativeLocationInDistribution)

				err = copyFile(match, updateRoot, relativeLocationInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)

			}

			//err = copyFile(filename, isDir, "", relativeLocationInDistribution)
			//util.HandleError(err)
			break readDestinationLoop
		}
	}
	return nil
}

func handleSingleMatch(filename string, matchingNode *Node, isDir bool, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	fmt.Println("[SINGLE MATCH]", filename, "; match:", matchingNode.relativeLocation)

	updateRoot := viper.GetString(constant.UPDATE_ROOT)
	//fullPath := path.Join(distributionRoot, relativeLocationInDistribution)
	//logger.Debug("fullPath:", fullPath)
	//Ignore error because we are only checking whether the given path exists or not

	if isDir {
		//fullPath := path.Join(relativeLocationInDistribution, filename)
		//fmt.Println("Checking all files for:", fullPath)
		allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
		fmt.Println("All matches:", allMatchingFiles)
		for _, match := range allMatchingFiles {

			fmt.Println("match:", match)
			if viper.GetBool(constant.CHECK_MD5) {
				data := allFilesMap[match]
				md5Matches := CompareMD5(rootNode, path.Join(matchingNode.relativeLocation, match), data.md5)
				if md5Matches {
					fmt.Println("XXXXXXXXXXXXXXXXXXXXXX MD5 matches. Ignoring file.")
					continue
				} else {
					fmt.Println("MD5 does not match. Copying the file.")
				}
			}

			fmt.Println("match:", match)
			//fullPath := path.Join(relativeLocationInDistribution, match)
			//
			//fmt.Println("Copying match:",fullPath )
			fmt.Println("[Copy] " + match + " ; From:" + updateRoot + "; To:" + matchingNode.relativeLocation)

			err := copyFile(match, updateRoot, matchingNode.relativeLocation, rootNode, updateDescriptor)
			util.HandleError(err)
		}

	} else {
		fmt.Println("[Copy] " + filename + " ; From:" + updateRoot + "; To:" + matchingNode.relativeLocation)

		err := copyFile(filename, updateRoot, matchingNode.relativeLocation, rootNode, updateDescriptor)
		util.HandleError(err)
	}

	return nil
}

func handleMultipleMatches(filename string, isDir bool, matches map[string]*Node, allFilesMap map[string]Data, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	locationTable, indexMap := generateLocationTable(filename, matches)
	locationTable.Render()
	logger.Debug("indexMap:", indexMap)
	var selectedIndices []string
	for {
		util.PrintInBold("Enter preference(s)[Multiple selections separated by commas]: ")
		preferences, err := util.GetUserInput()
		util.HandleError(err)
		logger.Debug("preferences: %s", preferences)
		//Remove the new line at the end
		preferences = strings.TrimSpace(preferences)
		//Split the indices
		selectedIndices = strings.Split(preferences, ",");
		//Sort the locations
		sort.Strings(selectedIndices)
		logger.Debug("sorted: %s", preferences)

		length := len(indexMap)
		isValid, err := util.IsUserPreferencesValid(selectedIndices, length)
		if err != nil {
			util.PrintError("Invalid preferences. Please select indices where 1 <= index <= " + strconv.Itoa(length))
			continue
		}

		if !isValid {
			util.PrintError("Invalid preferences. Please select indices where 1 <= index <= " + strconv.Itoa(length))
		} else {
			logger.Debug("Entered preferences are valid.")
			break
		}
	}

	updateRoot := viper.GetString(constant.UPDATE_ROOT)

	if isDir {
		for _, selectedIndex := range selectedIndices {
			pathInDistribution := indexMap[selectedIndex]
			fmt.Println("[MULTIPLE MATCHES] Selected path:", selectedIndex, ";", pathInDistribution)

			allMatchingFiles := getAllMatchingFiles(filename, allFilesMap)
			fmt.Println("matchingFiles:", allMatchingFiles)

			for _, match := range allMatchingFiles {

				fmt.Println("match:", match)
				if viper.GetBool(constant.CHECK_MD5) {
					data := allFilesMap[match]
					md5Matches := CompareMD5(rootNode, path.Join(pathInDistribution, match), data.md5)
					if md5Matches {
						fmt.Println("XXXXXXXXXXXXXXXXXXXXXX MD5 matches. Ignoring file.")
						continue
					} else {
						fmt.Println("MD5 does not match. Copying the file.")
					}
				}

				//fullPath := filepath.Join(relativeLocationInDistribution, match)
				//
				//fmt.Println("Copying match:",fullPath )
				fmt.Println("[Copy] " + match + " ; From:" + updateRoot + "; To:" + pathInDistribution)

				err := copyFile(match, updateRoot, pathInDistribution, rootNode, updateDescriptor)
				util.HandleError(err)
			}
		}
	} else {
		for _, selectedIndex := range selectedIndices {
			pathInDistribution := indexMap[selectedIndex]
			fmt.Println("[MULTIPLE MATCHES] Selected path:", selectedIndex, ";", pathInDistribution)

			fmt.Println("[Copy] " + filename + " ; From:" + updateRoot + "; To:" + pathInDistribution)

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
		fmt.Println("[WALK] ", absolutePath, ";", fileInfo.IsDir())
		//check current file in ignored files map. This is useful to ignore update-descriptor.yaml, etc in update directory
		if ignoredFiles != nil {
			_, found := ignoredFiles[fileInfo.Name()]
			if found {
				return nil
			}
		}
		//get the parent directory path
		//parentDirectory := strings.TrimSuffix(absolutePath, fileInfo.Name())
		//Check for file / directory

		relativePath := strings.TrimPrefix(absolutePath, root + "/")
		info := Data{
			name: fileInfo.Name(),
			relativePath: relativePath,
		}

		if fileInfo.IsDir() {

			logger.Debug(absolutePath + " : " + fileInfo.Name())
			//locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), "")

			info.isDir = true

			if path.Join(root, fileInfo.Name()) == absolutePath {
				rootLevelDirectoriesMap[fileInfo.Name()] = true
			}
		} else {

			if path.Join(root, fileInfo.Name()) == absolutePath {
				rootLevelFilesMap[fileInfo.Name()] = false
			}

			logger.Debug("[MD5] Calculating MD5")
			//If it is a file, calculate md5 sum
			md5Sum, err := util.GetMD5(absolutePath)
			if err != nil {
				return err
			}
			logger.Debug(absolutePath + " : " + fileInfo.Name() + ": " + md5Sum)

			info.md5 = md5Sum
			info.isDir = false
			//locationMap.Add(fileInfo.Name(), parentDirectory, fileInfo.IsDir(), md5)
			//
			//fullPath := path.Join(root, constant.PLUGINS_DIRECTORY, fileInfo.Name())
			//logger.Trace("[COMPARE] " + fullPath + " ; " + absolutePath)
			//if (fullPath == absolutePath) && util.HasJarExtension(absolutePath) {
			//	logger.Debug("[PLUGIN] FilePath:", absolutePath)
			//	newFileName := strings.Replace(fileInfo.Name(), "_", "-", 1)
			//	logger.Debug("[PLUGIN] New Name:", newFileName)
			//	if index := strings.Index(newFileName, "_"); index != -1 {
			//		return errors.New(fileInfo.Name() + " is in " + constant.PLUGINS_DIRECTORY + ". But it has multiple _ in its name. Only one _ is expected." )
			//	}
			//	locationMap.Add(newFileName, parentDirectory, fileInfo.IsDir(), md5)
			//}

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
	fmt.Println("productName:", productName)
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

		//fmt.Println(file.Name)
		relativePath := strings.TrimPrefix(file.Name, productName + "/")
		//fmt.Println(relativePath,"\n")
		AddToRootNode(&rootNode, strings.Split(relativePath, "/"), file.FileInfo().IsDir(), md5Hash)
		if !file.FileInfo().IsDir() {
			fileMap[relativePath] = false
		}
	}
	return rootNode, nil
}

func AddToRootNode(root *Node, path []string, isDir bool, md5Hash string) *Node {
	//fmt.Println("Checking:", path[0], ":", path)
	if len(path) == 1 {
		//fmt.Println("end reached")
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
			//fmt.Println("Already exists")
			AddToRootNode(Node, path[1:], isDir, md5Hash)
		} else {
			//fmt.Println("New node")
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
	//fmt.Println("$$$$$$$$$$$$$$$$$$$$$$$$$$")
	//fmt.Println("All:", rootNode.childNodes)
	//fmt.Println("checking:", path[0])
	childNode, found := rootNode.childNodes[path[0]]

	if found {
		//fmt.Println(path[0] + " found")
		if len(path) > 1 {
			return NodeExists(childNode, path[1:], isDir)
		} else {
			return childNode.isDir == isDir
		}
	}
	//fmt.Println(path[0] + " NOT found")
	return false
}

func CompareMD5(rootNode *Node, relativePath string, md5 string) bool {
	fmt.Println("Checking MD5:", relativePath)
	return CheckMD5(rootNode, strings.Split(relativePath, "/"), md5)
}

func CheckMD5(rootNode *Node, path []string, md5 string) bool {
	//fmt.Println("$$$$$$$$$$$$$$$$$$$$$$$$$$")
	//fmt.Println("All:", rootNode.childNodes)
	//fmt.Println("checking:", path[0])
	childNode, found := rootNode.childNodes[path[0]]

	if found {
		fmt.Println(path[0] + " found")
		if len(path) > 1 {
			return CheckMD5(childNode, path[1:], md5)
		} else {
			return childNode.isDir == false && childNode.md5Hash == md5
		}
	}
	fmt.Println(path[0] + " NOT found")
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
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.MANDATORY) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.OPTIONAL) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.SKIP) {
		filesMap[file] = true
	}
	return filesMap
}

//This will return a map of files which would be copied to the temp directory before creating the update zip. Key is the
// file name and value is whether the file is mandatory or not.
func getResourceFiles() map[string]bool {
	filesMap := make(map[string]bool)
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.MANDATORY) {
		filesMap[file] = true
	}
	for _, file := range viper.GetStringSlice(constant.RESOURCE_FILES + "." + constant.OPTIONAL) {
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
				util.PrintWarning("'" + filename + "' not found.")
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
		fmt.Println("[TABLE] filepath:", distributionFilepath, "; isDir:", locationsInDistribution[distributionFilepath].isDir)
		indexMap[strconv.Itoa(index)] = distributionFilepath
		relativePath := path.Join("CARBON_HOME", distributionFilepath)
		locationTable.Append([]string{strconv.Itoa(index), path.Join(relativePath, filename)})
		index++
	}
	return locationTable, indexMap
}

////This function will copy the file/directory from update to temp location
func copyFile(filename string, locationInUpdate, relativeLocationInTemp string, rootNode *Node, updateDescriptor *util.UpdateDescriptor) error {
	//todo list
	//check type
	//if file, check path matches /plugin path
	//else dir, check files in the directory for matches

	//get the relative path in the distribution and join to the temp directory to get the destination directory
	fmt.Println("[FINAL][COPY ROOT] Name:", filename, "; IsDir: false", "; From:", locationInUpdate, "; To:", relativeLocationInTemp)
	updateName := viper.GetString(constant.UPDATE_NAME)
	source := path.Join(locationInUpdate, filename)
	carbonHome := path.Join(constant.TEMP_DIR, updateName, constant.CARBON_HOME)
	destination := path.Join(carbonHome, relativeLocationInTemp)
	isABundle := IsABundle(filename, relativeLocationInTemp)

	var fullPath string
	if isABundle {
		newName := ConstructBundleName(filename)
		fullPath = path.Join(destination, newName)
		err := util.CreateDirectory(util.GetParentDirectory(fullPath))
		util.HandleError(err)
		fmt.Println("[FINAL][COPY][TEMP] Name:", filename, "; From:", source, "; To:", fullPath)
		err = util.CopyFile(source, fullPath)
		util.HandleError(err, "temp1")

	} else {
		fullPath = path.Join(destination, filename)
		err := util.CreateDirectory(util.GetParentDirectory(fullPath))
		util.HandleError(err)
		fmt.Println("[FINAL][COPY][TEMP2] Name:", filename, "; From:", source, "; To:", fullPath)
		err = util.CopyFile(source, fullPath)
		util.HandleError(err, "temp2")
	}

	relativePath := strings.TrimPrefix(fullPath, carbonHome + constant.PATH_SEPARATOR)
	fmt.Println("relativePath:", relativePath)
	contains := PathExists(rootNode, relativePath, false)
	fmt.Println("contains:", contains)
	if contains {
		updateDescriptor.File_changes.Modified_files = append(updateDescriptor.File_changes.Modified_files, relativePath)
	} else {
		updateDescriptor.File_changes.Added_files = append(updateDescriptor.File_changes.Added_files, relativePath)
	}
	return nil
}

////This function detects whether a jar should be a bundle or not according to its path
func IsABundle(filename, location string) bool {
	//todo
	if location == constant.PLUGINS_DIRECTORY {
		return true
	}
	return false
}
//
////This constructs the bundle name of a normal jar
func ConstructBundleName(filename string) string {
	//todo
	return filename
}
