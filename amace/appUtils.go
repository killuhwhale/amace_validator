// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"go.chromium.org/tast-tests/cros/common/android/ui"

	"go.chromium.org/tast-tests/cros/local/arc"
	"go.chromium.org/tast-tests/cros/local/chrome"

	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
	"golang.org/x/net/html"
)

const appVersiontimeoutUI = 30 * time.Second

type PackageInfo struct {
	Version string
	AppType AppType // AppTpye is App, Game, PWA => Propbably already typed
}

type AppInfo struct {
	PackageName string
	Info        PackageInfo
}

// NewAppInfo creates and populates app info.
func NewAppInfo(ctx context.Context, tconn *chrome.TestConn, s *testing.State, d *ui.Device, a *arc.ARC, packageName string) *AppInfo {
	appInfo := &AppInfo{
		PackageName: packageName,
		Info:        PackageInfo{},
	}
	appInfo.processApp(ctx, tconn, s, d, a)

	return appInfo
}

func (ai *AppInfo) processApp(ctx context.Context, tconn *chrome.TestConn, s *testing.State, d *ui.Device, a *arc.ARC) {
	if err := ai.openAppInfoPage(ctx, tconn, a, s); err != nil {
		s.Fatal("Failed to open app info page: ", err)
	}

	isGame, err := isGame(ctx, s, a, ai.PackageName)
	if err != nil {
		s.Log("Failed to check is game: ", ai.PackageName, err)
	}

	version, err := verifyAppVersion(ctx, d, tconn)
	if err != nil {
		s.Fatal("Failed verifying app Version: ", err)
	}

	// s.Log("Version / isGame: ", version, isGame)
	if isGame {
		ai.Info.AppType = GAME
	} else {
		ai.Info.AppType = APP
	}
	ai.Info.Version = version
}

// func (ai *AppInfo) Info() map[string]string {
// 	return ai.info
// }

// openAppInfoPage opens app info page of PlayStore.
func (ai *AppInfo) openAppInfoPage(ctx context.Context, tconn *chrome.TestConn, a *arc.ARC, s *testing.State) error {
	// cmd := exec.Command("adb", "shell", "am", "start", "-a", "android.settings.APPLICATION_DETAILS_SETTINGS", "-d", "package:"+packageName)
	cmd := a.Command(ctx, "am", "start", "-a", "android.settings.APPLICATION_DETAILS_SETTINGS", "-d", "package:"+ai.PackageName)
	output, err := cmd.Output()
	if err != nil {
		s.Log("Error, failed opening details page: ", err)
		return err
	}
	s.Log("Output: ", output)

	// // Open App info page by right click on Play Store App.
	// settings := nodewith.Name("Settings").Role(role.Window).First()
	// playstoreSubpage := "Play Store subpage back button"

	// ui := uiauto.New(tconn)

	// playstoreSubpageButton := nodewith.Name(playstoreSubpage).Role(role.Button).Ancestor(settings)
	// appInfoMenu := nodewith.Name("App info").Role(role.MenuItem)

	// openPlayStoreAppInfoPage := func() uiauto.Action {
	// 	return uiauto.Combine("check app context menu and settings",
	// 		ui.LeftClick(appInfoMenu),
	// 		ui.WaitUntilExists(playstoreSubpageButton))
	// }

	// moreSettingsButton := nodewith.Name("More settings and permissions").Role(role.Link)
	// if err := uiauto.Combine("check context menu of play store app on the shelf",
	// 	ash.RightClickApp(tconn, apps.PlayStore.Name),
	// 	openPlayStoreAppInfoPage(),
	// 	ui.LeftClick(moreSettingsButton))(ctx); err != nil {
	// 	return errors.Wrap(err, "failed to open app info for Play Store app")
	// }
	return nil
}

// verifyAppVersion check that version is present in app info page under Advanced.
func verifyAppVersion(ctx context.Context, d *ui.Device,
	tconn *chrome.TestConn) (string, error) {

	// Click on Advanced to expand it.
	advancedSettings := d.Object(ui.ClassName("android.widget.TextView"), ui.TextMatches("(?i)Advanced"), ui.Enabled(true))
	if err := advancedSettings.WaitForExists(ctx, appVersiontimeoutUI); err != nil {
		return "", errors.Wrap(err, "failed to find Advanced")
	}
	if err := advancedSettings.Click(ctx); err != nil {
		return "", errors.Wrap(err, "failed to click Advanced")
	}

	// Scroll until the version is displayed.
	scrollLayout := d.Object(ui.ClassName("android.support.v7.widget.RecyclerView"), ui.Scrollable(true))
	if t, ok := arc.Type(); ok && t == arc.VM {
		scrollLayout = d.Object(ui.ClassName("androidx.recyclerview.widget.RecyclerView"), ui.Scrollable(true))
	}
	system := d.Object(ui.ClassName("android.widget.TextView"), ui.TextContains("(?i)Modify system settings"), ui.Enabled(true))
	if err := scrollLayout.WaitForExists(ctx, appVersiontimeoutUI); err == nil {
		scrollLayout.ScrollTo(ctx, system)
	}

	// Verify version is not empty.
	versionText, err := d.Object(ui.ID("android:id/summary"), ui.TextStartsWith("version")).GetText(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to ger version")
	}
	if len(versionText) == 0 {
		return "", errors.Wrap(err, "version is empty")
	}
	testing.ContextLogf(ctx, "App Version = %s", versionText)
	return versionText, nil
}

// IsGame detects if an app is a game or not.
func isGame(ctx context.Context, s *testing.State, a *arc.ARC, packageName string) (bool, error) {
	cmd := a.Command(ctx, "dumpsys", "SurfaceFlinger", "--list")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	for _, str := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(str, "SurfaceView") && strings.Contains(str, packageName) {
			return true, nil
		}
		s.Log("String does not match the criteria.")
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
		s.Log("Error:", err)
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

// IsAppOpen returns true if the app is open
func IsAppOpen(ctx context.Context, a *arc.ARC, packageName string) bool {
	cmd := a.Command(ctx, "dumpsys", "activity", "processes", "|", "grep", "-i", packageName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	outStr := strings.TrimSpace(string(output))
	return len(outStr) > 0
}
