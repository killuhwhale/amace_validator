// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package arc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"chromiumos/tast/common/android/ui"
	"chromiumos/tast/local/arc"
	"chromiumos/tast/local/bundles/cros/arc/amace"
	"chromiumos/tast/local/chrome"
	"chromiumos/tast/local/chrome/ash"
	"chromiumos/tast/local/chrome/display"
	"chromiumos/tast/local/chrome/uiauto"
	"chromiumos/tast/local/chrome/uiauto/nodewith"
	"chromiumos/tast/local/chrome/uiauto/restriction"
	"chromiumos/tast/local/input"

	"go.chromium.org/tast/core/ctxutil"
	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
)

func init() {
	testing.AddTest(&testing.Test{
		Func: AMACE,
		Desc: "Checks Apps for AMACE",
		Contacts: []string{
			"candaya@google.com", // Optional test contact
		},
		Attr:         []string{"group:mainline", "informational"},
		Data:         []string{"AMACE_app_list.tsv", "AMACE_secret.txt"},
		SoftwareDeps: []string{"chrome", "android_vm"},
		Timeout:      36 * 60 * time.Minute,
		Fixture:      "arcBootedWithPlayStore",
		BugComponent: "b:1234",
		LacrosStatus: testing.LacrosVariantUnneeded,
	})
}

// -var=arc.AccessVars.globalPOSTURL="http://192.168.1.229:3000/api/amaceResult"
// postURL is default "https://appval-387223.wl.r.appspot.com/api/amaceResult" || -var=arc.AccessVars.globalPOSTURL
var postURL = testing.RegisterVarString(
	"arc.AccessVars.globalPOSTURL",
	"https://appval-387223.wl.r.appspot.com/api/amaceResult",
	"Url for api endpoint.",
)
var runTS = testing.RegisterVarString(
	"arc.amace.runts",
	"na",
	"Run timestamp for current run.",
)
var runID = testing.RegisterVarString(
	"arc.amace.runid",
	"na",
	"Run uuid for current run.",
)
var startat = testing.RegisterVarString(
	"arc.amace.startat",
	"na",
	"App index to start at.",
)

type requestBody struct {
	BuildInfo  string          `json:"buildInfo"`
	DeviceInfo string          `json:"deviceInfo"`
	AppName    string          `json:"appName"`
	PkgName    string          `json:"pkgName"`
	RunID      string          `json:"runID"`
	RunTS      string          `json:"runTS"`
	AppTS      int64           `json:"appTS"`
	Status     amace.AppStatus `json:"status"`
	IsGame     bool            `json:"isGame"`
}

var centerButtonClassName = "FrameCenterButton"

func AMACE(ctx context.Context, s *testing.State) {

	a := s.FixtValue().(*arc.PreData).ARC
	cr := s.FixtValue().(*arc.PreData).Chrome
	d := s.FixtValue().(*arc.PreData).UIDevice

	// cleanupCtx := ctx
	ctx, cancel := ctxutil.Shorten(ctx, 5*time.Second)
	defer cancel()

	tconn, err := cr.TestAPIConn(ctx)
	if err != nil {
		s.Fatal("Failed to create Test API connection: ", err)
	}

	if err := amace.SetDeviceNoSleepOnPower(ctx, d, tconn, s, cr); err != nil {
		s.Fatal("Failed to turn off sleep on power: ", err)
	} else {
		s.Log("Turned off sleep on power: ", err)
	}

	if runID.Value() == "na" || runTS.Value() == "na" {
		s.Fatalf("Run info not provided: ID=%s TS=%s", runID.Value(), runTS.Value())
	}

	buildInfo, err := amace.GetBuildInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get build info")
	}
	deviceInfo, err := amace.GetDeviceInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info ")
	}
	testApps, err := amace.LoadAppList(s, startat.Value())
	if err != nil {
		s.Fatal("Error loading App List.tsv: ", err)
	}
	arcV, err := a.GetProp(ctx, "ro.build.version.release")
	if err != nil {
		s.Fatal("Failed to get Arc Verson for device")
	}

	secret, err := amace.LoadSecret(s)
	if err != nil {
		s.Fatal("Failed to get secret")
	}
	s.Logf("arcV: %s, build: %s, device: %s", arcV, buildInfo, deviceInfo)

	dispInfo, err := display.GetPrimaryInfo(ctx, tconn)
	if err != nil {
		s.Fatal("Failed to get primary display info: ", err)
	}
	fmt.Println("Display info: ", dispInfo.Name)

	keyboard, err := input.Keyboard(ctx)
	if err != nil {
		s.Fatal("Failed to create a keyboard: ", err)
	}
	defer keyboard.Close(ctx)

	var status amace.AppStatus
	for _, appPack := range testApps {
		appTS := time.Now().UnixMilli()
		// Signals a new app run to python parent manage-program
		s.Logf("--appstart@|~|%s|~|%s|~|%s|~|%s|~|%d|~|%v|~|%d|~|%s|~|%s|~|", runID.Value(), runTS.Value(), appPack.Pname, appPack.Aname, 0, false, appTS, buildInfo, deviceInfo)

		s.Log("Installing app", appPack)
		if err := amace.InstallARCApp(ctx, s, a, d, appPack); err != nil {
			s.Log("Failed to install app: ", appPack.Pname, err)
			if err.Error() == "Need to purchase app" {
				status = amace.PRICE
			} else if err.Error() == "device is not compatible with app" {
				status = amace.DEVICENOTCOMPAT
			} else if err.Error() == "app not compatible with this device" {
				status = amace.OLDVERSION
			} else if err.Error() == "too many attempst: app failed to install" {
				status = amace.INSTALLFAIL
			} else if err.Error() == "app not availble in your country" {
				status = amace.COUNTRYNA
			}

			res, err := postData(
				amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: status, IsGame: false},
				s, buildInfo, secret, deviceInfo)
			if err != nil {
				s.Log("Error posting: ", err)

			}
			s.Log("Post res: ", res)

			continue
		}
		s.Log("App Installed", appPack)

		s.Log("Launching app", appPack)
		if err := launchApp(ctx, s, a, appPack.Pname); err != nil {
			// GoBigSleepLint Need to wait for act to start...
			testing.Sleep(ctx, 2*time.Second)
			// Check for misc Pop ups here.

			if err := launchApp(ctx, s, a, appPack.Pname); err != nil {
				s.Log("Error lanching app: ", err)
				if err := a.Uninstall(ctx, appPack.Pname); err != nil {
					if err := uninstallApp(ctx, s, a, appPack.Pname); err != nil {
						s.Log("Failed to uninstall app: ", appPack.Aname)
					}
				}
				continue
			}
		}
		s.Log("App launched ", appPack)
		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 2*time.Second)

		s.Log("Checking AMAC-E: ")
		var isGame bool = false
		status, err = checkAppStatus(ctx, tconn, s, d, appPack.Pname, appPack.Aname)
		if err != nil {
			s.Log("💥💥💥 App failed to check: ", appPack.Pname, err)
			// TODO() post here
			continue
		}
		// GoBigSleepLint Need to wait for SurfaceView to be used/ created...
		testing.Sleep(ctx, 5*time.Second)
		isGame, err = amace.IsGame(ctx, s, a, appPack.Pname)
		if err != nil {
			s.Log("Failed to check is game: ", appPack.Pname, err)
		}
		s.Logf("✅💥 %s is a game %v", appPack.Pname, isGame)

		// Logging purposes only
		if status < 5 {
			s.Log("💥 App failed: ", appPack.Pname, status)
		}
		if status == amace.O4C || status == amace.O4CFullScreenOnly {
			s.Log("✅ App is O4C: ", appPack.Pname, status)
		}
		if status >= 7 && status < 11 {
			s.Log("❌ App is AMAC-E:", appPack.Pname, status)
		}
		if status == amace.PWA {
			s.Log("❌❌ App is PWA:", appPack.Pname, status)
		}

		// Create result and post
		ar := amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: status, IsGame: isGame}
		s.Log("💥✅❌✅💥 App Result: ", ar)
		res, err := postData(ar, s, buildInfo, secret, deviceInfo)
		if err != nil {
			s.Log("Error posting: ", err)
		}
		s.Log("Post res: ", res)

		// Misc apps that have one off behavior that need to be dealt with.
		checkMiscAppForKnownBehavior(ctx, keyboard, appPack.Pname)

		s.Log("Uninstalling app: ", appPack.Pname)
		if err := a.Uninstall(ctx, appPack.Pname); err != nil {
			if err := uninstallApp(ctx, s, a, appPack.Pname); err != nil {
				s.Log("Failed to uninstall app: ", appPack.Aname)
			}
		}
	}
	s.Log("--~~rundone") // Signals python parent manage-program that the run is over.
}

func checkAppStatus(ctx context.Context, tconn *chrome.TestConn, s *testing.State, d *ui.Device, pkgName, appName string) (amace.AppStatus, error) {
	// 1. Check window size
	// If launched Maximized:
	// Potentail candidate for FS -> Amace
	// Check to Minimized App
	// App minimized: Check for CenterFrameButton (checkVisibility())
	// [FS >  AMAC ]

	// Cannot Unmaximize
	// [FS only]

	// [Not AMACE]

	// If not launched in maximized,
	// Check for CenterFrameButton (checkVisibility())
	// Check if its disabled
	// [AMAC (disabled)]
	// [AMAC]
	// [Not AMACE]

	windowChan := make(chan *ash.Window, 1)
	errorChan := make(chan string, 1)
	var result *ash.Window
	var isFullScreen bool
	s.Log("Getting window state ")
	go getWindowState(ctx, windowChan, errorChan, tconn, s, pkgName)

	s.Log("Got window state")
	select {
	case result = <-windowChan:
		s.Log("result window State: ", result.State)
	case err := <-errorChan:
		// Handle the result
		s.Log("result window err: ", err)
	case <-time.After(time.Second * 5):
		// Handle timeout
		s.Log("Timeout occurred while getting ARC window state")
	}
	if result == nil {
		s.Log("Window returned was nil")
	}

	if result.WindowType == ash.WindowTypeExtension {
		// App is PWA.
		if strings.ToLower(result.Title) != strings.ToLower(appName) {
			s.Logf("💥✅❌❌✅💥 Found PWA for %s but Window Title does not match appName: %s", result.Title, appName)
		}
		return amace.PWA, nil
	}

	isFullOrMax := result.State == ash.WindowStateMaximized || result.State == ash.WindowStateFullscreen
	if isFullOrMax && result.CanResize {
		// Potentail for FS => Amace
		// Minimize app and check for Amace Type
		isFullScreen = true
		s.Log("App is  Fullscreen, but can resize ")
	} else if isFullOrMax && !result.CanResize {
		s.Log("✅ App is O4C since its Fullscreen, no resize")
		return amace.O4CFullScreenOnly, nil
	}

	if isFullScreen {
		_, err := ash.SetARCAppWindowStateAndWait(ctx, tconn, pkgName, ash.WindowStateNormal)
		if err != nil {
			s.Log("Failed to set ARC window state: ", err)
			return amace.Fail, errors.New("continue")
		}
	}

	go getWindowState(ctx, windowChan, errorChan, tconn, s, pkgName)

	select {
	case result = <-windowChan:
		s.Log("window state is now: ", result.State)
	case err := <-errorChan:
		// Handle the result
		s.Log("result window err: ", err)
	case <-time.After(time.Second * 5):
		// Handle timeout
		s.Log("Timeout occurred while getting ARC window state")
	}

	// At this point, we have a restored/ Normal window
	if err := checkVisibility(ctx, tconn, centerButtonClassName, false /* visible */); err != nil {
		if err.Error() == "failed to start : failed to start activity: exit status 255" {
			s.Log("App error : ", err)
			return amace.Fail, errors.New("continue")
		}
		// If the error was not a failure error, we know the AMACE-E Label is present.
		ui := uiauto.New(tconn)
		centerBtn := nodewith.HasClass(centerButtonClassName)
		nodeInfo, err := ui.Info(ctx, centerBtn) // Returns info about the node, and more importantly, the window status
		if err != nil {
			s.Log("Failed to find the node info")
			return amace.Fail, errors.New("failed to find the node info")
		}

		if nodeInfo != nil {
			s.Log("Node info: ", nodeInfo)
			s.Log("Node info: Restriction", nodeInfo.Restriction)
			s.Log("Node info: Checked", nodeInfo.Checked)
			s.Log("Node info: ClassName", nodeInfo.ClassName)
			s.Log("Node info: Description", nodeInfo.Description)
			s.Log("Node info: HTMLAttributes", nodeInfo.HTMLAttributes)
			s.Log("Node info: Location", nodeInfo.Location)
			s.Log("Node info: Name", nodeInfo.Name)
			s.Log("Node info: Restriction", nodeInfo.Restriction)
			s.Log("Node info: Role", nodeInfo.Role)
			s.Log("Node info: Selected", nodeInfo.Selected)
			s.Log("Node info: State", nodeInfo.State)
			s.Log("Node info: Value", nodeInfo.Value)

			if nodeInfo.Restriction == restriction.Disabled {
				if result.BoundsInRoot.Width < result.BoundsInRoot.Height {
					return amace.IsLockedPAmacE, nil
				}
				return amace.IsLockedTAmacE, nil
			}
		} else {
			return amace.Fail, errors.New("nodeInfo was nil")
		}

		if isFullScreen {
			return amace.IsFSToAmacE, nil
		}
		return amace.IsAmacE, nil
	}
	return amace.O4C, nil
}

func postData(appResult amace.AppResult, s *testing.State, buildInfo, secret, deviceInfo string) (string, error) {
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
		appResult.IsGame,
	}

	// Convert the data to JSON
	jsonData, err := json.Marshal(requestBody)
	s.Log("JSON data: ", requestBody, string(jsonData))
	if err != nil {
		fmt.Printf("Failed to marshal request body: %v\n", err)
		return "", err
	}

	// Create a new POST request with the JSON data
	s.Log("Posting to: ", postURL.Value())
	request, err := http.NewRequest("POST", postURL.Value(), bytes.NewBuffer(jsonData))
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

func checkMiscAppForKnownBehavior(ctx context.Context, k *input.KeyboardEventWriter, pkgName string) error {
	switch pkgName {
	case "bn.ereader":
		amace.CloseBNobleWifi(ctx, k)
	}

	return nil
}

func uninstallApp(ctx context.Context, s *testing.State, arc *arc.ARC, pname string) error {
	cmd := arc.Command(ctx, "uninstall", pname)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	s.Log("Output: ", output)
	return nil
}

func launchApp(ctx context.Context, s *testing.State, arc *arc.ARC, pname string) error {
	// cmd = ('adb','-t', transport_id, 'shell', 'monkey', '--pct-syskeys', '0', '-p', package_name, '-c', 'android.intent.category.LAUNCHER', '1')
	cmd := arc.Command(ctx, "monkey", "--pct-syskeys", "0", "-p", pname, "-c", "android.intent.category.LAUNCHER", "1")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	s.Log("Output: ", output)
	return nil
}

// checkVisibility checks whether the node specified by the given class name exists or not.
func checkVisibility(ctx context.Context, tconn *chrome.TestConn, className string, visible bool) error {
	uia := uiauto.New(tconn)
	finder := nodewith.HasClass(className).First()
	if visible {
		return uia.WithTimeout(10 * time.Second).WaitUntilExists(finder)(ctx)
	}
	return uia.WithTimeout(10 * time.Second).WaitUntilGone(finder)(ctx)
}

// checkResizability checks if window can resize.
func checkResizability(ctx context.Context, tconn *chrome.TestConn, s *testing.State, pkgName string) error {
	return testing.Poll(ctx, func(ctx context.Context) error {
		window, err := ash.GetARCAppWindowInfo(ctx, tconn, pkgName)
		if err != nil {
			return errors.Wrapf(err, "failed to get the ARC window infomation for package name %s", pkgName)
		}

		s.Log("Window state: ", window.State)
		s.Log("Window canResize: ", window.CanResize)

		return nil
	}, &testing.PollOptions{Timeout: 10 * time.Second})
}

// getWindowState returns the window state
func getWindowState(ctx context.Context, resultChan chan<- *ash.Window, errorChan chan<- string, tconn *chrome.TestConn, s *testing.State, pkgName string) {

	s.Log("Calling Arc Window state: ")
	window, err := ash.GetARCAppWindowInfo(ctx, tconn, pkgName)
	s.Log("Arc Window state: ", window, err)

	if err != nil && err.Error() == "couldn't find window: failed to find window" {
		pwawindow, pwaerr := ash.GetActiveWindow(ctx, tconn)
		s.Log("Arc Window not found because we have pwa most likely, check for ARCWindow?: ", window, err)

		if pwawindow != nil {
			s.Log("Window state: ", pwawindow.WindowType)
			s.Log("Window state: ", pwawindow.Name)
			s.Log("Window state: ", pwawindow.OverviewInfo)
			s.Log("Window state: ", pwawindow.Title) // TikTok
			s.Log("Window state: ", pwawindow.State)
			resultChan <- pwawindow
		}
		if pwaerr != nil {
			s.Log("Thoewing error on channel: ", pwaerr)
			// errorChan <- err.Error()
		}
		return
	}

	s.Log("Thoewing error on channel: ", err)
	if window != nil {
		s.Log("ARC Window state: ", window.State)
		s.Log("ARC Window state: ", window.WindowType)
		resultChan <- window
	}
	if err != nil {
		s.Log("Thoewing error on channel: ", err)
		// errorChan <- err.Error()
	}
}
