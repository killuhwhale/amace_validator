// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"chromiumos/tast/local/arc"
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.chromium.org/tast/core/testing"
)

// DeviceOnPower turns on the power
func DeviceOnPower(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "settings", "put", "global", "stay_on_while_plugged_in", "3")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return string(output), nil
}

// GetDeviceInfo gets information for DeviceInfo
func GetDeviceInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.product.board")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return string(output), nil
}

// GetBuildInfo gets information for BuildInfo
func GetBuildInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.build.display.id")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return string(output), nil
}

// IsGame detects if an app is a game or not.
func IsGame(ctx context.Context, s *testing.State, a *arc.ARC, packageName string) (bool, error) {
	cmd := a.Command(ctx, "dumpsys", "SurfaceFlinger", "--list")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Define the regular expression pattern
	patternWithSurface := fmt.Sprintf(`^SurfaceView\s*-\s*%s/[\w.#]*$`)
	reSurface := regexp.MustCompile(patternWithSurface)

	// Execute the adb shell command to get the list of surfaces
	surfacesList := strings.TrimSpace(string(output))
	last := ""

	// Find matches using the regular expression pattern
	matches := reSurface.FindAllStringSubmatch(surfacesList, -1)
	for _, match := range matches {
		fmt.Println("Found surface match:", match)
		last = match[0]
	}

	if last != "" {
		if packageName != last {
			fmt.Println("Found match for wrong package.")
			return false, nil
		}
		return true, nil
	}

	return false, nil
}
