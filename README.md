# Update Creator Tool

### Introduction

Update Creator Tool is a tool which would help to create new updates and validate the update zip file after any manual editing. This tool is written in GO language. One of the main advantages of writing this in GO is that we can compile the code directly to machine code so we don’t need something like JVM to run this tool. Another advantage is we can directly cross compile the code.

This tool mainly provides 2 functions.

1. Creating a new updates
2. Validating an Update

### Installation

First you need to install GO to compile and run this tool. You can find instructions to how to download and install the GO from the [Official Website](https://golang.org/doc/install). 

Then run the following command.

`go get -u github.com/wso2/update-creator-tool`

This will download and install the packages along with their dependencies. 

Then run `build.sh`. This will generate the executable files for various OS/Architecture combinations. These will be located at **/build/target/** directory. Extract the relevant zip file to your OS/Architecture. In the *bin* directory, you'll find the executable **wum-uc** file. 

**Note:** Add this path to your system path variables. Then you'll be able to call **wum-uc** from anywhere.

### Creating an update

This is done using the `create` command. Place all of your updated files in a directory. Lets call this directory **UPDATE_LOCATION**. This directory should following files.

1. All of the updated files
2. update-descriptor.yaml
3. LICENSE.txt

Optional Files -

1. NOT_A_CONTRIBUTION.txt
2. instructions.txt

**Note:** You can generate the **update-descriptor.yaml** file using the **init** command as shown in the next section. You need to have the product distribution(which you are creating the update for) locally as well. This is used to compare files and create the proper file structure in the update zip. The distribution location can be either the zip file or the extracted directory.
#### init command

This command will generate the **update-descriptor.yaml** file. If no arguments are provided, this will init the current working directory. You can provide a directory path as an argument otherwise. If there is a README.txt in the old patch format in the initializing directory, this command will try to parse the necessary details from the README.txt file and use them to populate **update-descriptor.yaml** file. Otherwise, the fields will have a default value which you need to edit manually.

```bash
wum-uc init [<directory>]

<directory> - Directory which needs to be initialized. If this is not provided, current working directory will be initialized.
<flags> - Flags for the tool. Currently, supported flags are -d and -t which will print debug logs, trace logs.
```

#### create command

This command will create a new update.

```bash
wum-uc create <update_loc> <dist_loc> [<flags>]

<update_loc> - Location of the updated files.
<dist_loc> - Location of the distribution. This can either be a zip file or a directory.
<flags> - Flags for the tool. Currently, supported flags are -d and -t which will print debug logs, trace logs.
```

If the *UPDATE_LOCATION* contained the update 0001, by running this command, you will create a new zip file called **WSO2-CARBON-UPDATE-4.4.0–0001.zip** in the current working directory. Platform Version and Update Number are read from the *update-descriptor.yaml* file.

**NOTE:** You can run `wum-uc --help` get a list of available commands. Also you can run `wum-uc create --help` to find out more about the create command.

#### validation command

After we create a update, we might want to unzip it and add more detail to the *update-descriptor.yaml* like removed files. After we do these changes, we can use this validation command to verify that the file structure of the zip is is the same as the distribution.

```bash
wum-uc validate <update_loc> <dist_loc> [<flags>]

<update_loc> - Location of the update. This should be a zip file.
<dist_loc> - Location of the distribution. This can either be a zip file or a directory.
<flags> - Flags for the tool. Currently, supported flags are -d and -t which will print debug logs, trace logs.
```

This will compare the update zip’s directories and files with the distribution’s directories and files.

**NOTE:** Also you can run `wum-uc validate --help` to view the help.
