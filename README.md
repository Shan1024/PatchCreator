# Update Creator Tool

### Introduction

Update Creator Tool(UCT for short) is a tool which would help to create new updates and validate the update zip file after any manual editing. This tool is written in GO language. One of the main advantages of writing this tool in GO is that we can compile the code directly to machine code so we don’t need something like JVM to run this code. Another advantage is we can directly cross compile the code.

This tool mainly provides 2 functions.

1. Creating a new updates
2. Validating an Update

### Installation

You need to install GO first to compile and run this tool. You can find instructions to how to download and install the GO from the [Official Website](https://golang.org/doc/install). 

Then run

`go get -u github.com/wso2/update-creator-tool`

This will download and install the packages along with their dependencies. 

There are two ways to run the tool.

1. Run `build.sh`. This will generate the executable files for various OS/Architecture combinations. These will be located at **/build/target/** directory. Extract the relevant zip file to your OS/Architecture. In the *bin* folder, you'll find the executable **uct** file.

2. Run `go install github.com/wso2/update-creator-tool`. This will generate the executable **uct** file in the **GOPATH/src** directory.

### Creating a update

This is done using the `create` command. Place all of your patched files in a directory. Lets call this directory **UPDATE_LOCATION**. This directory should following files.

1. All of the updated files
2. update-descriptor.yaml
3. LICENSE.txt

Optional Files -

1. NOT_A_CONTRIBUTION.txt
2. instructions.txt

You need to have the distribution locally as well. This is used to compare files and create the proper file structure in the update zip. The distribution location can be either the zip file or the extracted directory. Lets call this file/location **DIST_LOCATION**.

*create* command looks like this.

```
./uct create <update> <dist> [<flags>]

<update> - Location of the patched files.
<dist> - Location of the distribution. This can either be a zip file or a directory.
<flags> - Flags for the tool. Currently, supported flags are -d and -t which will print debug and trace logs.
```

In our case, the *create* command will look like this.

`uct create UPDATE_LOCATION DIST_LOCATION`

If the *UPDATE_LOCATION* contained the update 0001, by running this command, you will create a new zip file called **WSO2-CARBON-UPDATE-4.4.0–0001.zip** in the current working directory. Platform Version and Update Number are read from the *update-descriptor.yaml* file.

**NOTE:** You can run `./uct --help` get a list of available commands. Also you can run `./uct create --help` to find out more about arguments and flags of the create command.

### Validating an Update

After we create a update, we might want to unzip it and add more detail to the *update-descriptor.yaml* or the *README.txt*. After we do these changes, we can create a zip using the files again. We can use this validation command to verify that the file structure of the zip is is the same as the distribution.

*validation* command looks like this.

```
./uct validate <update> <dist> [<flags>]

<update> - Is the location of the update. This should be a zip file.
<dist> - Location of the distribution. This can either be a zip file or a directory.
<flags> - Flags for the tool. Currently, supported flags are -d and -t which will print debug and trace logs.
```

Sample usage:

`uct validate LOCATION_OF_THE_UPDATE_ZIP DIST_LOCATION`

This will compare the update zip’s folders and files with the distribution’s folders and files.

**NOTE:** Also you can run `./uct validate --help` to find out more about arguments and flags of the validate command.
