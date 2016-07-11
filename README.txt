Update Creation Tool
====================

Update Creation Tool(UCT for short) is a tool which would help to create new Updates and help converting old Patches to new Update format. This tool is written in GO language. One of the main advantages of writing this tool in GO is that we can compile the code directly to machine code so we don’t need something like JVM to run this code. Another advantage is we can cross compile the code.

This tool mainly provides 2 functionalities.

1) Creating a new updates / Converting an old patches into new update format.
2) Validating an Update.

Creating a new update
=====================

This is done using the create command. Place all of your patched files and update-descriptor.yaml in a directory. Lets call this directory UPDATE_LOCATION.
You need to have the distribution locally as well. This is used to compare files and create the proper file structure. This location can be either the zip file location of the distribution or the extracted location. Lets call this location DIST_LOCATION.
Updating a old patch file to new format is also similar. See examples at the end for more details.

Create command looks like this.

uct create [<flags>] <update> <dist>

<update> — Is the location of the patched files
<dist> — Location of the distribution. This can either be a zip file or a directory.
<flags> — Flags for the tool. Currently, only supported flag is -l which will print logs.

So the command will look like this.

uct create PATCH_LOCATION DIST_LOCATION

By running this command, you will create a new zip file called WSO2-CARBON-UPDATE-4.4.0–0001.zip in the current working directory. Kernel Version and Update Number are read from the update-descriptor.yaml file.

Validating an Update
====================
After we create a update, we might want to unzip it and add more detail to the update-descriptor or the readme. After we do these changes, we can zip the files. We can use this validation command to verify that the file structure of the update zip is not changed compared to the distribution.
Validation command looks like this.

uct validate [<flags>] <update> <dist>

<update> — Is the location of the update. This should be a zip file.
<dist> — Location of the distribution. This can either be a zip file or a directory.
<flags> — Flags for the tool. Currently, only supported flag is -l which will print logs.

Sample usage :
uct validate location_to_update_zip DIST_LOCATION

This will compare the update zip’s folders/files with the distribution’s folders/files.

Note: Don’t forget to add new files(if any) to update-descriptor.yaml under file_changes. Otherwise the validation will fail because there are no files like them in the distribution. See the Sample 2 for more details.

