// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"go.chromium.org/tast/core/testing"
)

// possible keys to map
// Close
// Continue
// FBAuth
// GoogleAuth
// Slider
// Two
// loginfield
// passwordfield
type YoloResult struct {
	Data map[string][]struct {
		Coords [2][2]int `json:"coords"`
		Conf   float64   `json:"conf"`
	} `json:"data"`
}

func YoloDetect(ctx context.Context, hostIP string) (YoloResult, error) {
	start := time.Now()

	var yoloResult YoloResult
	ss, err := GetSS()
	if err != nil {
		testing.ContextLog(ctx, "Error getting SS for Yolo: ", err)
		return yoloResult, err
	}

	// Create a new multipart buffer
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add the screenshot file
	imageField, err := writer.CreateFormFile("image", "img.png")
	if err != nil {
		testing.ContextLog(ctx, "Err: ", err)
		return yoloResult, err
	}

	// Write the image data to the form file field
	if _, err = imageField.Write(ss); err != nil {
		testing.ContextLog(ctx, "Err: ", err)
		return yoloResult, err
	}
	// Close the multipart writer
	writer.Close()

	// testing.ContextLog(ctx, "JSON data: ", body)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8000/yolo/", hostIP), body)

	// Set the Content-Type header to the multipart form data boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		testing.ContextLog(ctx, "Error unexpected: ", err)
		return yoloResult, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		testing.ContextLog(ctx, "Error: ", fmt.Errorf("unexpected status code: %d", resp.StatusCode))
		// return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return yoloResult, errors.New("Unexpected status code while getting yolo result")
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		testing.ContextLog(ctx, "Error:", err)
		return yoloResult, err
	}

	testing.ContextLog(ctx, "bodyBytes: ", string(bodyBytes))

	err = json.Unmarshal([]byte(bodyBytes), &yoloResult)
	if err != nil {
		testing.ContextLog(ctx, "Error w/ yolo result:", err)
		return yoloResult, err
	}

	testing.ContextLog(ctx, "")
	testing.ContextLog(ctx, "")
	testing.ContextLog(ctx, "")
	testing.ContextLog(ctx, "Yolo: ")
	testing.ContextLog(ctx, "num diff names/labels: ", len(yoloResult.Data), yoloResult.Data["Continue"])

	if len(yoloResult.Data) == 0 {
		// GoBigSleepLint Wait for app to load some more and potentially fail...
		testing.Sleep(ctx, 5*time.Second)
	}

	for key, values := range yoloResult.Data {
		if key == "Continue" {
			testing.ContextLog(ctx, "Contine Button Found")
			testing.ContextLog(ctx, "Found %d buttons.", len(values))
			for _, button := range values {
				topLeft := button.Coords[0]
				bottomRight := button.Coords[1]

				testing.ContextLogf(ctx, "Found button at: %v, %v w/ conf: %.3f", topLeft, bottomRight, button.Conf)

			}

		}
	}
	testing.ContextLog(ctx, "")
	testing.ContextLog(ctx, "")
	testing.ContextLog(ctx, "")
	elapsed := time.Since(start)
	testing.ContextLogf(ctx, "Detection took: %s\n", elapsed)
	return yoloResult, nil

}
