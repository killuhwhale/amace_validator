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

	// "strings"
	"time"

	// "chromiumos/tast/common/testexec"
	"chromiumos/tast/local/arc"
	"chromiumos/tast/local/bundles/cros/arc/amace"

	// "chromiumos/tast/local/arc/playstore"

	"chromiumos/tast/local/chrome"
	"chromiumos/tast/local/chrome/display"
	"chromiumos/tast/local/chrome/uiauto"
	"chromiumos/tast/local/chrome/uiauto/nodewith"
	"chromiumos/tast/local/input"

	"github.com/google/uuid"
	"go.chromium.org/tast/core/ctxutil"

	// "go.chromium.org/tast/core/shutil"
	"go.chromium.org/tast/core/testing"
)

// Install app.
// Your device isn.t compatible with this version.
type requestBody struct {
	BuildInfo  string          `json:"buildInfo"`
	DeviceInfo string          `json:"deviceInfo"`
	AppName    string          `json:"appName"`
	PkgName    string          `json:"pkgName"`
	RunID      string          `json:"runID"`
	RunTS      int64           `json:"runTS"`
	AppTS      int64           `json:"appTS"`
	Status     amace.AppStatus `json:"status"`
}

func init() {
	testing.AddTest(&testing.Test{
		Func: AMACE,
		Desc: "Checks a list of apps for AMAC-E window or O4C.",
		Contacts: []string{
			"candaya@google.com",
		},
		Attr:         []string{"group:mainline", "informational"},
		Data:         []string{"AMACE_app_list.tsv", "AMACE_secret.txt"},
		SoftwareDeps: []string{"chrome", "android_vm"},
		Timeout:      24 * 60 * time.Minute,
		Fixture:      "arcBootedWithPlayStore",
		BugComponent: "b:1234",
	})
}

// -var=arc.AccessVars.globalPackageName=com.duolingo -var=arc.AccessVars.globalActivityName=com.duolingo.app.LoginActivity
// -var=arc.AccessVars.globalPOSTURL="http://192.168.1.229:3000/api/amaceResult"
// postURL is "https://appval-387223.wl.r.appspot.com/api/amaceResult" || "http://192.168.1.229:3000/api/amaceResult"
var postURL = testing.RegisterVarString(
	"arc.AccessVars.globalPOSTURL",
	"https://appval-387223.wl.r.appspot.com/api/amaceResult",
	"Url for api endpoint.",
)

func AMACE(ctx context.Context, s *testing.State) {

	a := s.FixtValue().(*arc.PreData).ARC
	cr := s.FixtValue().(*arc.PreData).Chrome
	d := s.FixtValue().(*arc.PreData).UIDevice
	res, err := amace.DeviceOnPower(ctx, s, a)
	if err != nil {
		s.Log("Failed to turn screen on while plugged: ", err)

	} else {
		s.Log("Screen on while plugged in set: ", res)
	}

	// tname := "O4C App"

	runID := uuid.New()
	runTS := time.Now().UnixMilli()
	buildInfo, err := amace.GetBuildInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get build info")
	}
	deviceInfo, err := amace.GetDeviceInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info ")
	}
	testApps, err := amace.LoadAppList(s)
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
	s.Logf("arcV: %s, build: %s, device: %s, secret: %s", arcV, buildInfo, deviceInfo, secret)

	// cleanupCtx := ctx
	ctx, cancel := ctxutil.Shorten(ctx, 5*time.Second)
	defer cancel()

	tconn, err := cr.TestAPIConn(ctx)
	if err != nil {
		s.Fatal("Failed to create Test API connection: ", err)
	}

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

	var appResults []amace.AppResult
	var status amace.AppStatus
	for _, appPack := range testApps {

		s.Log("Installing app", appPack)
		if err := amace.InstallARCApp(ctx, s, a, d, appPack); err != nil {
			s.Log("Failed to install app: ", appPack.Pname, err)
			if err.Error() == "Need to purchase app" {
				status = amace.PRICE
			} else if err.Error() == "app not compatible with this device" {
				status = amace.OLDVERSION
			} else if err.Error() == "too many attempst: app failed to install" {
				status = amace.INSTALLFAIL
			}

			res, err := postData(
				amace.AppResult{App: appPack, RunID: runID.String(), RunTS: runTS, AppTS: runTS, Status: status},
				s, buildInfo, secret, deviceInfo)
			if err != nil {
				s.Log("Errir posting: ", err)

			}
			s.Log("Post res: ", res)

			continue
		}
		s.Log("App Installed", appPack)

		s.Log("Launching app", appPack)
		// defer faillog.DumpUITreeWithScreenshotOnError(cleanupCtx, s.OutDir(), s.HasError, cr, "ui_tree")

		if err := launchApp(ctx, s, a, appPack.Pname); err != nil {
			// GoBigSleepLint Need to wait for act to start...
			testing.Sleep(ctx, 2*time.Second)
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
		var ar amace.AppResult
		appTS := time.Now().UnixMilli()
		if err := checkVisibility(ctx, tconn, "FrameCenterButton", false /* visible */); err != nil {
			if err.Error() == "failed to start : failed to start activity: exit status 255" {
				s.Log("Failed to start app ", appPack)
				s.Log("App error : ", err)
				continue
			} else {
				ar = amace.AppResult{App: appPack, RunID: runID.String(), RunTS: runTS, AppTS: appTS, Status: amace.IsAmacE}
				s.Log("❌ App is AMAC-E:", appPack)
			}
		} else {
			ar = amace.AppResult{App: appPack, RunID: runID.String(), RunTS: runTS, AppTS: appTS, Status: amace.O4C}
			s.Log("✅ App is O4C: ", appPack)
		}

		res, err := postData(ar, s, buildInfo, secret, deviceInfo)
		if err != nil {
			s.Log("Errir posting: ", err)

		}
		s.Log("Post res: ", res)

		appResults = append(appResults, ar)

		s.Log("Uninstalling app: ", appPack.Pname)
		if err := a.Uninstall(ctx, appPack.Pname); err != nil {
			if err := uninstallApp(ctx, s, a, appPack.Pname); err != nil {
				s.Log("Failed to uninstall app: ", appPack.Aname)
			}
		}
	}

	s.Log("App results: ", appResults)
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
	}

	// Convert the data to JSON
	jsonData, err := json.Marshal(requestBody)
	s.Log("JSON data: ", requestBody, string(jsonData))
	if err != nil {
		fmt.Printf("Failed to marshal request body: %v\n", err)
		return "", err
	}

	// Create a new POST request with the JSON data

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
