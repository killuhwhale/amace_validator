// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"chromiumos/tast/common/android/ui"
	"chromiumos/tast/local/arc"
	"chromiumos/tast/local/chrome"
	"chromiumos/tast/local/chrome/uiauto"
	"chromiumos/tast/local/chrome/uiauto/nodewith"
	"chromiumos/tast/local/chrome/uiauto/ossettings"
	"chromiumos/tast/local/chrome/uiauto/role"
	"context"
	"strings"

	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
)

// SetDeviceNoSleepOnPower set the setttings for a run.
func SetDeviceNoSleepOnPower(ctx context.Context, d *ui.Device, tconn *chrome.TestConn, s *testing.State, cr *chrome.Chrome) error {
	ui := uiauto.New(tconn)
	settings, err := ossettings.LaunchAtPage(ctx, tconn, nodewith.Name("Power").Role(role.Link))
	if err != nil {
		return errors.Wrap(err, "failed to launch os-settings Power page")
	}
	defer settings.Close(ctx)

	idleActionWhileCharging := nodewith.Name("Idle action while charging").Role(role.ComboBoxSelect)
	if err := ui.LeftClick(idleActionWhileCharging)(ctx); err != nil {
		return errors.Wrap(err, "failed to left click on idle action while charging in combo box")
	}

	keepDisplayOnListBox := nodewith.Name("Keep display on").Role(role.ListBoxOption)
	if err := ui.LeftClick(keepDisplayOnListBox)(ctx); err != nil {
		return errors.Wrap(err, "failed to left click on keep display in list box")
	}

	return nil
}

// GetDeviceInfo gets information for DeviceInfo
func GetDeviceInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.product.board")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return strings.ReplaceAll(string(output), "\n", ""), nil
}

// GetBuildInfo gets information for BuildInfo
func GetBuildInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.build.display.id")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return strings.ReplaceAll(string(output), "\n", ""), nil
}

// GetArcVerison gets information for BuildInfo
func GetArcVerison(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.build.version.release")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return strings.ReplaceAll(string(output), "\n", ""), nil
}

// GetArcVerison gets information for BuildInfo
func GetBuildChannel(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.boot.chromeos_channel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return strings.ReplaceAll(string(output), "\n", ""), nil
}
