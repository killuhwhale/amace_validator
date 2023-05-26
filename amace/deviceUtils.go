// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"chromiumos/tast/local/arc"
	"context"

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
