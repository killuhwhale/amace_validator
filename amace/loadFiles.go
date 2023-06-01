// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package amace

import (
	"fmt"
	"io/ioutil"
	"strings"

	"go.chromium.org/tast/core/testing"
)

// AppPackage holds App Info
type AppPackage struct {
	Pname string // Install app package name
	Aname string // launch app name
}

// LoadSecret secret from file to post to backend
func LoadSecret(s *testing.State) (string, error) {
	b, err := ioutil.ReadFile(s.DataPath("AMACE_secret.txt"))
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	return string(b), nil
}

// LoadAppList loads list to use to check status of AMAC-E
func LoadAppList(s *testing.State) ([]AppPackage, error) {
	// homeDir, err := os.UserHomeDir()
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return nil, err
	// }
	// filePath := filepath.Join("~/chromiumos", "app_list.tsv")
	// // Read the contents of the file
	// content, err := ioutil.ReadFile(filePath)
	b, err := ioutil.ReadFile(s.DataPath("AMACE_app_list.tsv"))
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	// Convert file content to string
	fileContent := string(b)

	// Print the file content
	fmt.Println(fileContent)
	var pgks []AppPackage
	lines := strings.Split(fileContent, "\n")
	// Split each line by tabs
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		pgks = append(pgks, AppPackage{fields[1], fields[0]})
		fmt.Println(fields)
	}

	return pgks[:10], nil
}
