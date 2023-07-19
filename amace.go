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

	"go.chromium.org/tast-tests/cros/common/android/ui"
	"go.chromium.org/tast-tests/cros/local/arc"
	"go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace"
	"go.chromium.org/tast-tests/cros/local/chrome"
	"go.chromium.org/tast-tests/cros/local/chrome/ash"
	"go.chromium.org/tast-tests/cros/local/chrome/display"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto/nodewith"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto/restriction"

	"go.chromium.org/tast-tests/cros/local/input"
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
var hostIP = testing.RegisterVarString(
	"arc.amace.hostip",
	"192.168.1.1337",
	"Host device ip on local network to reach image server.",
)
var postURL = testing.RegisterVarString(
	"arc.amace.posturl",
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
var device = testing.RegisterVarString(
	"arc.amace.device",
	"na",
	"Run uuid for current run.",
)
var startat = testing.RegisterVarString(
	"arc.amace.startat",
	"na",
	"App index to start at.",
)
var account = testing.RegisterVarString(
	"arc.amace.account",
	"na",
	"Automation account.",
)

// var onlyAMACEStatus = testing.RegisterVarString(
// 	"arc.amace.account",
// 	"na",
// 	"Automation account.",
// )

type requestBody struct {
	BuildInfo    string                `json:"buildInfo"`
	DeviceInfo   string                `json:"deviceInfo"`
	AppName      string                `json:"appName"`
	PkgName      string                `json:"pkgName"`
	RunID        string                `json:"runID"`
	RunTS        string                `json:"runTS"`
	AppTS        int64                 `json:"appTS"`
	Status       amace.AppStatus       `json:"status"`
	BrokenStatus amace.AppBrokenStatus `json:"brokenStatus"`
	AppType      amace.AppType         `json:"appType"`
	AppVersion   string                `json:"appVersion"`
	AppHistory   *amace.AppHistory     `json:"history"`
	Logs         string                `json:"logs"`
}

var centerButtonClassName = "FrameCenterButton"

func AMACE(ctx context.Context, s *testing.State) {
	s.Log("########################################")
	s.Log("Account: ", account.Value())
	s.Log("Host IP: ", hostIP.Value())
	s.Log("Post URL: ", postURL.Value())
	s.Log("Device: ", device.Value())
	s.Log("Start at: ", startat.Value())
	s.Log("########################################")

	a := s.FixtValue().(*arc.PreData).ARC

	ax := s.FixtValue().(*arc.PreData)
	cr := s.FixtValue().(*arc.PreData).Chrome
	d := s.FixtValue().(*arc.PreData).UIDevice
	ax.ARC.ReadFile(ctx, "")

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

	buildInformation, err := amace.GetBuildInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get build info")
	}
	buildChannel, err := amace.GetBuildChannel(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info ")
	}
	deviceInformation, err := amace.GetDeviceInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info ")
	}
	arcVersion, err := amace.GetArcVerison(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info ")
	}

	buildInfo := fmt.Sprintf(("%s - %s (%s)"), buildInformation, buildChannel, arcVersion)
	deviceInfo := fmt.Sprintf(("%s - %s"), deviceInformation, device.Value())

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

	errorDetector := amace.NewErrorDetector(ctx, a, s)
	var appHistory amace.AppHistory
	var crash amace.ErrResult
	var status amace.AppStatus
	var finalLogs string
	var tmpAppType amace.AppType

	for _, appPack := range testApps {
		// Reset Final logs
		finalLogs = ""
		// Reset History
		appHistory = amace.AppHistory{}
		// Reset Logs
		crash = amace.ErrResult{}
		// New App TS
		appTS := time.Now().UnixMilli()

		// Signals a new app run to python parent manage-program
		s.Logf("--appstart@|~|%s|~|%s|~|%s|~|%s|~|%d|~|%v|~|%d|~|%s|~|%s|~|", runID.Value(), runTS.Value(), appPack.Pname, appPack.Aname, 0, false, appTS, buildInfo, deviceInfo)

		// ####################################
		// ####   Install APP           #######
		// ####################################
		s.Log("Installing app", appPack)

		if err := amace.InstallARCApp(ctx, s, a, d, appPack, strings.Split(account.Value(), ":")[1]); err != nil {
			s.Log("Failed to install app: ", appPack.Pname, "  - err: ", err)
			if err.Error() == "App purchased" {
				status = amace.PURCHASED
			} else if err.Error() == "Need to purchase app" {
				status = amace.PRICE
			} else if err.Error() == "device is not compatible with app" {
				status = amace.DEVICENOTCOMPAT
			} else if err.Error() == "chromebook not compat" {
				status = amace.CHROMEBOOKNOTCOMPAT
			} else if err.Error() == "app not compatible with this device" {
				status = amace.OLDVERSION
			} else if err.Error() == "too many attempts" {
				status = amace.INSTALLFAIL
			} else if err.Error() == "app not availble in your country" {
				status = amace.COUNTRYNA
			} else if err.Error() == "too many attempst: app failed to install" {
				status = amace.TOOMANYATTEMPTS
			} else {
				status = amace.Fail
			}
			// When an app is purchased an error is thrown but we dont want to report the error.. Instead continue with the rest of the check.
			if status != amace.PURCHASED {
				addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App failed to install.", false)
				res, err := postData(
					amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: status, BrokenStatus: amace.FailedInstall, AppType: amace.APP, AppVersion: "", AppHistory: &appHistory, Logs: finalLogs},
					s, buildInfo, secret, deviceInfo)
				if err != nil {
					s.Log("Error posting: ", err)

				}
				s.Log("Post res: ", res)

				continue
			} else {
				addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "Purchased app.", false)
			}
		}

		s.Log("App Installed", appPack)
		addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App Installed.", false)

		// ####################################
		// ####   Gather App Info       #######
		// ####################################
		appInfo := amace.NewAppInfo(ctx, tconn, s, d, a, appPack.Pname)
		s.Log("AppInfo version: ", appInfo.Info.Version)
		s.Log("AppInfo apptype: ", appInfo.Info.AppType)

		// ####################################
		// ####   Launch APP            #######
		// ####################################
		s.Log("Launching app", appPack)

		errorDetector.ResetStartTime()
		errorDetector.UpdatePackageName(appPack.Pname)

		if err := amace.LaunchApp(ctx, s, a, appPack.Pname); err != nil {
			// GoBigSleepLint Need to wait for act to start...
			testing.Sleep(ctx, 2*time.Second)
			// Check for misc Pop ups here.

			if err := amace.LaunchApp(ctx, s, a, appPack.Pname); err != nil {
				addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App failed to launch.", false)

				res, err := postData(
					amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: amace.LaunchFail, BrokenStatus: amace.FailedLaunch, AppType: amace.APP, AppVersion: "", AppHistory: &appHistory, Logs: finalLogs},
					s, buildInfo, secret, deviceInfo)
				if err != nil {
					s.Log("Error posting: ", err)

				}
				s.Log("Post res: ", res)
				s.Log("Error lanching app: ", err)
				if err := a.Uninstall(ctx, appPack.Pname); err != nil {
					if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
						s.Log("Failed to uninstall app: ", appPack.Aname)
					}
				}
				continue
			}
		}
		s.Log("App launched ", appPack)

		addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App Launched.", false)

		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 5*time.Second)

		// ####################################
		// ####   Check errors          #######
		// ####################################
		s.Log("Checking errors: ")
		errorDetector.DetectErrors()

		if !amace.IsAppOpen(ctx, a, appPack.Pname) {
			s.Log("App is NOT open!")

			addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App closed unexpectedly.", false)
			if crash = errorDetector.GetHighError(); len(crash.CrashLogs) > 0 {
				s.Logf("App has error logs: %s/n %s/n %s/n", crash.CrashType, crash.CrashMsg, crash.CrashLogs)

				finalLogs = amace.GetFinalLogs(crash)
				res, err := postData(
					amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: amace.Crashed, BrokenStatus: crash.CrashType, AppType: tmpAppType, AppVersion: "", AppHistory: &appHistory, Logs: finalLogs},
					s, buildInfo, secret, deviceInfo)
				if err != nil {
					s.Log("Error posting: ", err)

				}
				s.Log("Post res: ", res)
				if err := a.Uninstall(ctx, appPack.Pname); err != nil {
					if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
						s.Log("Failed to uninstall app: ", appPack.Aname)
					}
				}
				continue
			}
		} else {
			s.Log("App is still open!")
		}

		// ####################################
		// ####   Check Amace Window    #######
		// ####################################
		s.Log("Checking AMAC-E: ")
		arcWindow, status, err := checkAppStatus(ctx, tconn, s, d, appPack.Pname, appPack.Aname)
		if err != nil {
			s.Log("💥💥💥 App failed to check: ", appPack.Pname, err)
			res, err := postData(
				amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: amace.Fail, BrokenStatus: amace.FailedAmaceCheck, AppType: appInfo.Info.AppType, AppVersion: appInfo.Info.Version, AppHistory: &appHistory, Logs: finalLogs},
				s, buildInfo, secret, deviceInfo)
			if err != nil {
				s.Log("Error posting: ", err)

			}
			s.Log("Post res: ", res)
			if err := a.Uninstall(ctx, appPack.Pname); err != nil {
				if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
					s.Log("Failed to uninstall app: ", appPack.Aname)
				}
			}
			continue
		}

		if status == amace.PWA {
			tmpAppType = amace.PWAAPP
		} else {
			tmpAppType = appInfo.Info.AppType
		}
		addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App Window Status Verification Image.", true)

		// ####################################
		// ####   Check Errors Again    #######
		// ####################################
		// Check if app is still open after checking window status, if not open, check error.

		if crash = errorDetector.GetHighError(); len(crash.CrashLogs) > 0 {
			s.Logf("App has error logs: %s/n %s/n %s/n", crash.CrashType, crash.CrashMsg, crash.CrashLogs)

			finalLogs = amace.GetFinalLogs(crash)

			if !amace.IsAppOpen(ctx, a, appPack.Pname) {
				s.Log("App is NOT open!")

				addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App closed unexpectedly.", false)
				res, err := postData(
					amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: amace.Crashed, BrokenStatus: crash.CrashType, AppType: tmpAppType, AppVersion: "", AppHistory: &appHistory, Logs: finalLogs},
					s, buildInfo, secret, deviceInfo)
				if err != nil {
					s.Log("Error posting: ", err)

				}
				s.Log("Post res: ", res)
				if err := a.Uninstall(ctx, appPack.Pname); err != nil {
					if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
						s.Log("Failed to uninstall app: ", appPack.Aname)
					}
				}
				continue
			} else {
				// TODO() check screen shot of app for black screen.....
				s.Log("App is still open!") // HayDay stays open and has an error, black screen. Other apps are fine....
				windowBounds := arcWindow.BoundsInRoot
				isBlkScreen, err := amace.IsBlackScreen(ctx, tconn, windowBounds)
				if err != nil {
					testing.ContextLog(ctx, "Black screen error: ", err)

				} else if isBlkScreen {
					testing.ContextLog(ctx, "App HAS black screen: ")
					addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App crashed with black screen.", false)
					res, err := postData(
						amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: amace.Crashed, BrokenStatus: crash.CrashType, AppType: tmpAppType, AppVersion: "", AppHistory: &appHistory, Logs: finalLogs},
						s, buildInfo, secret, deviceInfo)
					if err != nil {
						s.Log("Error posting: ", err)

					}
					s.Log("Post res: ", res)
					if err := a.Uninstall(ctx, appPack.Pname); err != nil {
						if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
							s.Log("Failed to uninstall app: ", appPack.Aname)
						}
					}
					continue

				} else {
					testing.ContextLog(ctx, "App DOES NOT have black screen: ")
				}
				addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App open with detected error, check for black screen.", false)

			}

		}

		addHistoryWithImage(ctx, tconn, s, &appHistory, deviceInfo, appPack.Pname, "App isnt broken.", false)
		finalLogs = amace.GetFinalLogs(crash)

		// ####################################
		// ####   Post APP Results      #######
		// ####################################
		// // Create result and post
		ar := amace.AppResult{}

		// We only detect a PWA via Status (amace status), we need to override the app/game check to report its a PWA too.

		ar = amace.AppResult{App: appPack, RunID: runID.Value(), RunTS: runTS.Value(), AppTS: appTS, Status: status, BrokenStatus: amace.Pass, AppType: tmpAppType, AppVersion: appInfo.Info.Version, AppHistory: &appHistory, Logs: finalLogs}
		s.Log("💥✅❌✅💥 App Result: ", ar)

		res, err := postData(ar, s, buildInfo, secret, deviceInfo)
		if err != nil {
			s.Log("Error posting: ", err)
		}
		s.Log("Post res: ", res)

		// // Misc apps that have one off behavior that need to be dealt with.
		// checkMiscAppForKnownBehavior(ctx, keyboard, appPack.Pname)

		s.Log("Uninstalling app: ", appPack.Pname)
		if err := a.Uninstall(ctx, appPack.Pname); err != nil {
			if err := amace.UninstallApp(ctx, s, a, appPack.Pname); err != nil {
				s.Log("Failed to uninstall app: ", appPack.Aname)
			}
		}
	}
	s.Log("--~~rundone") // Signals python parent manage-program that the run is over.

}

func addHistoryWithImage(ctx context.Context, tconn *chrome.TestConn, s *testing.State, ah *amace.AppHistory, device, packageName, histMsg string, viaChrome bool) {
	hs := fmt.Sprint(len(ah.History))
	s.Log("Getting history len: ", ah.History, len(ah.History))
	imgPath := amace.PostSS(ctx, tconn, s, device, packageName, hs, runID.Value(), hostIP.Value(), viaChrome)
	ah.AddHistory(histMsg, imgPath)
}

// Check out WaitWindowFinishAnimating, might want to use this as well WaitWindowFinishAnimating
func checkAppStatus(ctx context.Context, tconn *chrome.TestConn, s *testing.State, d *ui.Device, pkgName, appName string) (*ash.Window, amace.AppStatus, error) {
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
		return nil, amace.Fail, errors.New(err)
	case <-time.After(time.Second * 5):
		// Handle timeout
		s.Log("Timeout occurred while getting ARC window state")
		return nil, amace.Fail, errors.New("Timeout while getting ARC window state")
	}
	if result == nil {
		s.Log("Window returned was nil")
		return nil, amace.Fail, errors.New("Window is nil")
	}

	if result.WindowType == ash.WindowTypeExtension {
		// App is PWA.
		if strings.ToLower(result.Title) != strings.ToLower(appName) {
			s.Logf("💥✅❌❌✅💥 Found PWA for %s but Window Title does not match appName: %s", result.Title, appName)
		}
		return result, amace.PWA, nil
	}

	isFullOrMax := result.State == ash.WindowStateMaximized || result.State == ash.WindowStateFullscreen
	if isFullOrMax && result.CanResize {
		// Potentail for FS => Amace
		// Minimize app and check for Amace Type
		isFullScreen = true
		s.Log("App is  Fullscreen, but can resize ")
	} else if isFullOrMax && !result.CanResize {
		s.Log("✅ App is O4C since its Fullscreen, no resize")
		return result, amace.O4CFullScreenOnly, nil
	}

	if isFullScreen {
		_, err := ash.SetARCAppWindowStateAndWait(ctx, tconn, pkgName, ash.WindowStateNormal)
		if err != nil {
			s.Log("Failed to set ARC window state: ", err)
			return result, amace.Fail, errors.New("continue")
		}
	}

	go getWindowState(ctx, windowChan, errorChan, tconn, s, pkgName)

	select {
	case result = <-windowChan:
		s.Log("result window State: ", result.State)
	case err := <-errorChan:
		// Handle the result
		s.Log("result window err: ", err)
		return nil, amace.Fail, errors.New(err)
	case <-time.After(time.Second * 5):
		// Handle timeout
		s.Log("Timeout occurred while getting ARC window state")
		return nil, amace.Fail, errors.New("Timeout while getting ARC window state")
	}
	if result == nil {
		s.Log("Window returned was nil")
		return nil, amace.Fail, errors.New("Window is nil")
	}

	// At this point, we have a restored/ Normal window
	if err := checkVisibility(ctx, tconn, centerButtonClassName, false /* visible */); err != nil {
		if err.Error() == "failed to start : failed to start activity: exit status 255" {
			s.Log("App error : ", err)
			return result, amace.Fail, errors.New("continue")
		}
		// If the error was not a failure error, we know the AMACE-E Label is present.
		ui := uiauto.New(tconn)
		centerBtn := nodewith.HasClass(centerButtonClassName)
		nodeInfo, err := ui.Info(ctx, centerBtn) // Returns info about the node, and more importantly, the window status
		if err != nil {
			s.Log("Failed to find the node info")
			return result, amace.Fail, errors.New("failed to find the node info")
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
					return result, amace.IsLockedPAmacE, nil
				}
				return result, amace.IsLockedTAmacE, nil
			}
		} else {
			return result, amace.Fail, errors.New("nodeInfo was nil")
		}

		if isFullScreen {
			return result, amace.IsFSToAmacE, nil
		}
		return result, amace.IsAmacE, nil
	}
	return result, amace.O4C, nil
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
		appResult.BrokenStatus,
		appResult.AppType,
		appResult.AppVersion,
		appResult.AppHistory,
		appResult.Logs,
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
