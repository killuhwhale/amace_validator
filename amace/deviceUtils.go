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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
	"golang.org/x/net/html"
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

// IsGame detects if an app is a game or not.
func IsGame(ctx context.Context, s *testing.State, a *arc.ARC, packageName string) (bool, error) {
	cmd := a.Command(ctx, "dumpsys", "SurfaceFlinger", "--list")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	for _, str := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(str, "SurfaceView") && strings.Contains(str, packageName) {
			return true, nil
		}
		fmt.Println("String does not match the criteria.")
	}
	// str := "SurfaceView - com.roblox.client/com.roblox.client.ActivityNativeMain#0"

	// Define the regular expression pattern
	// patternWithSurface := fmt.Sprintf(`^SurfaceView\s*-\s*%s/[\w.#]*$`, packageName)
	// reSurface := regexp.MustCompile(patternWithSurface)

	// // Execute the adb shell command to get the list of surfaces
	// surfacesList := strings.TrimSpace(string(output))
	// last := ""

	// // Find matches using the regular expression pattern
	// matches := reSurface.FindAllStringSubmatch(surfacesList, -1)
	// s.Log("SurfacesList: ", surfacesList)
	// s.Log("Matches: ", matches)
	// for _, match := range matches {
	// 	s.Log("Found surface match:", match)
	// 	last = match[0]
	// }

	// if last != "" {
	// 	if packageName != last {
	// 		s.Log("Found match for wrong package")
	// 		return false, nil
	// 	}
	// 	return true, nil
	// }

	// Check Google Play for h2 About this Game
	exists, err := checkAboutGameTagExists(packageName)
	if err != nil {
		fmt.Println("Error:", err)
		return false, err
	}

	if exists {
		return true, nil
	}

	return false, nil
}

func checkAboutGameTagExists(packageName string) (bool, error) {
	url := "https://play.google.com/store/apps/details?id=" + packageName

	// Send GET request
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return false, err
	}

	// Search for the <h2>About this game</h2> tag
	found := false
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h2" && n.FirstChild != nil && n.FirstChild.Data == "About this game" {
			found = true
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	return found, nil
}
