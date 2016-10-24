// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package util

// Default values used in the application
var (
	EnableDebugLogs = false
	EnableTraceLogs = false
	// We only check md5 if -m flag is not found. If -m is set, it's value by default is true. That means we don't
	// want to check md5 if this value is true. By default we want to check. So that's why we have set
	// CheckMd5Disabled to false here.
	CheckMd5Disabled = false
	ResourceFiles_Mandatory = []string{"update-descriptor.yaml", "LICENSE.txt"}
	ResourceFiles_Optional = []string{"instructions.txt", "NOT_A_CONTRIBUTION.txt"}
	ResourceFiles_Skip = []string{"README.txt"}
	PlatformVersions = map[string]string{
		"4.2.0": "turing",
		"4.3.0": "perlis",
		"4.4.0": "wilkes",
		"5.0.0": "hamming",
	}
)
