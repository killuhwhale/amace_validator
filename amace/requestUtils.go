// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"go.chromium.org/tast/core/testing"
	"golang.org/x/net/html"
)

type requestBody struct {
	BuildInfo    string          `json:"buildInfo"`
	DeviceInfo   string          `json:"deviceInfo"`
	AppName      string          `json:"appName"`
	PkgName      string          `json:"pkgName"`
	RunID        string          `json:"runID"`
	RunTS        int64           `json:"runTS"`
	AppTS        int64           `json:"appTS"`
	Status       AppStatus       `json:"status"`
	BrokenStatus AppBrokenStatus `json:"brokenStatus"`
	AppType      AppType         `json:"appType"`
	AppVersion   string          `json:"appVersion"`
	AppHistory   *AppHistory     `json:"history"`
	Logs         string          `json:"logs"`
	LoginResults int8            `json:"loginResults"`
	DSrcPath     string          `json:"dSrcPath"`
}

func PostData(appResult AppResult, s *testing.State, postURL, buildInfo, secret, deviceInfo string) (string, error) {
	s.Log("🚀 Pushing result for run id: ", appResult)
	// Create the data to send in the request
	requestBody := requestBody{
		buildInfo,
		deviceInfo,
		appResult.App.Aname,
		appResult.App.Pname,
		appResult.RunID,
		appResult.RunTS,
		appResult.AppTS,
		appResult.Status,
		appResult.BrokenStatus,
		appResult.AppType,
		appResult.AppVersion,
		appResult.AppHistory,
		appResult.Logs,
		appResult.LoginResults,
		appResult.DSrcPath,
	}

	// Convert the data to JSON
	jsonData, err := json.Marshal(requestBody)
	s.Log("JSON data: ", requestBody, string(jsonData))
	if err != nil {
		fmt.Printf("Failed to marshal request body: %v\n", err)
		return "", err
	}
	// return "Test Response", nil
	// Create a new POST request with the JSON data
	s.Log("Posting to: ", postURL)
	request, err := http.NewRequest("POST", postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to create the request: %v\n", err)
		return "", err
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", secret)

	// Send the POST request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Failed to make the request: %v\n", err)
		return "", err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read the response body: %v\n", err)
		return "", err
	}
	return string(body), nil
}

func CheckPackageExists(s *testing.State, packageName string) (bool, error) {
	s.Log(" Check Play Store to see if package exists: ", packageName)

	// return "Test Response", nil
	// Create a new POST request with the JSON data
	url := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s", packageName)
	s.Log("Posting to: ", url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Failed to create the request: %v\n", err)
		return false, err
	}

	// Set the Content-Type header
	request.Header.Set("Content-Type", "text/html")

	// Send the GET request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Failed to make the request: %v\n", err)
		return false, err
	}
	defer response.Body.Close()

	// Parse the HTML document
	doc, err := html.Parse(response.Body)
	if err != nil {
		fmt.Printf("Failed to parse the HTML document: %v\n", err)
		return false, err
	}

	// Initialize the flag
	isValid := true

	// Define a recursive function to traverse the parsed HTML tree
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if strings.Contains(n.Data, "We're sorry, the requested URL was not found on this server.") {
			s.Logf("Found HTML: %v, %v", n.Type, n.Data)
			isValid = false
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// Start traversing the document
	traverse(doc)
	s.Logf("CheckPackageExists: %s isValid: %v", packageName, isValid)
	return isValid, nil
}
