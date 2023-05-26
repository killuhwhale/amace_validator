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
	"regexp"
	"strings"

	// "strings"
	"time"

	// "chromiumos/tast/common/testexec"

	"chromiumos/tast/common/android/ui"
	"chromiumos/tast/local/arc"
	"chromiumos/tast/local/arc/playstore"

	"chromiumos/tast/local/chrome"
	"chromiumos/tast/local/chrome/display"
	"chromiumos/tast/local/chrome/uiauto"
	"chromiumos/tast/local/chrome/uiauto/nodewith"
	"chromiumos/tast/local/input"

	"github.com/google/uuid"
	"go.chromium.org/tast/core/ctxutil"
	"go.chromium.org/tast/core/errors"

	// "go.chromium.org/tast/core/shutil"
	"go.chromium.org/tast/core/testing"
)

// Install app.
type AppPackage struct {
	pname string // Install app package name
	aname string // launch app name
}
type AppResult struct {
	app     AppPackage
	runID   string
	runTS   int64
	appTS   int64
	isAMACE bool
}

func (ar AppResult) String() string {
	return fmt.Sprintf("isAmac-e: %v, Name: %s", ar.isAMACE, ar.app.aname)
}

type RequestBody struct {
	BuildInfo  string `json:"buildInfo"`
	DeviceInfo string `json:"deviceInfo"`
	AppName    string `json:"appName"`
	PkgName    string `json:"pkgName"`
	RunID      string `json:"runID"`
	RunTS      int64  `json:"runTS"`
	AppTS      int64  `json:"appTS"`
	IsAMACE    bool   `json:"isAMACE"`
}

func init() {
	testing.AddTest(&testing.Test{
		Func: AMACE,
		Desc: "Checks that the date command prints dates as expected",
		Contacts: []string{
			"candaya@google.com", // Optional test contact
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

func AMACE(ctx context.Context, s *testing.State) {
	a := s.FixtValue().(*arc.PreData).ARC
	cr := s.FixtValue().(*arc.PreData).Chrome
	d := s.FixtValue().(*arc.PreData).UIDevice

	// tname := "O4C App"
	runID := uuid.New()
	runTS := time.Now().UnixMilli()
	buildInfo, err := getBuildInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get build info.")
	}
	deviceInfo, err := getDeviceInfo(ctx, s, a)
	if err != nil {
		s.Fatal("Failed to get device info.")
	}
	testApps, err := loadAppList(s)
	if err != nil {
		s.Fatal("Error loading App List.tsv: ", err)
	}
	arcV, err := a.GetProp(ctx, "ro.build.version.release")
	if err != nil {
		s.Fatal("Failed to get Arc Verson for device")
	}

	secret, err := loadSecret(s)
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

	var appResults []AppResult

	for _, appPack := range testApps {

		s.Log("Installing app", appPack)
		if err := installApp(ctx, s, a, d, appPack); err != nil {
			s.Log("Failed to install app: ", appPack.pname)
			continue
		}
		s.Log("App Installed", appPack)

		s.Log("Launching app", appPack)
		// defer faillog.DumpUITreeWithScreenshotOnError(cleanupCtx, s.OutDir(), s.HasError, cr, "ui_tree")

		if err := launchApp(ctx, s, a, appPack.pname); err != nil {
			// GoBigSleepLint Need to wait for act to start...
			testing.Sleep(ctx, 2*time.Second)
			if err := launchApp(ctx, s, a, appPack.pname); err != nil {
				s.Log("Error lanching app: ", err)
				if err := a.Uninstall(ctx, appPack.pname); err != nil {
					if err := uninstallApp(ctx, s, a, appPack.pname); err != nil {
						s.Log("Failed to uninstall app: ", appPack.aname)
					}
				}
				continue
			}
		}
		s.Log("App launched ", appPack)
		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 2*time.Second)

		s.Log("Checking AMAC-E: ")
		var ar AppResult
		appTS := time.Now().UnixMilli()
		if err := CheckVisibility(ctx, tconn, CenterButtonClassName, false /* visible */); err != nil {
			if err.Error() == "failed to start : failed to start activity: exit status 255" {
				s.Log("Failed to start app ", appPack)
				s.Log("App error : ", err)
				continue
			} else {
				ar = AppResult{appPack, runID.String(), runTS, appTS, true}
				s.Log("❌ App is AMAC-E:", appPack)
			}
		} else {
			ar = AppResult{appPack, runID.String(), runTS, appTS, false}
			s.Log("✅ App is O4C: ", appPack)
		}
		s.Logf("🚀 Pushing result for run id: %s - %v", runID.String(), ar)
		res, err := postData(ar, s, buildInfo, secret, deviceInfo)
		if err != nil {
			s.Log("Errir posting: ", err)

		}
		s.Log("Post res: ", res)

		appResults = append(appResults, ar)

		s.Log("Uninstalling app: ", appPack.pname)
		if err := a.Uninstall(ctx, appPack.pname); err != nil {
			if err := uninstallApp(ctx, s, a, appPack.pname); err != nil {
				s.Log("Failed to uninstall app: ", appPack.aname)
			}
		}
	}

	s.Log("App results: ", appResults)
}

func getDeviceInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.product.board")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return string(output), nil
}
func getBuildInfo(ctx context.Context, s *testing.State, a *arc.ARC) (string, error) {
	cmd := a.Command(ctx, "getprop", "ro.build.display.id")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s.Log("Output: ", output)
	return string(output), nil
}

func postData(appResults AppResult, s *testing.State, buildInfo, secret, deviceInfo string) (string, error) {
	url := "http://192.168.1.229:3000/api/amaceResult"
	// url := "https://appval-387223.wl.r.appspot.com/api/amaceResult"

	// Create the data to send in the request
	requestBody := RequestBody{
		buildInfo,
		deviceInfo,
		appResults.app.aname,
		appResults.app.pname,
		appResults.runID,
		appResults.runTS,
		appResults.appTS,
		appResults.isAMACE,
	}

	// Convert the data to JSON
	jsonData, err := json.Marshal(requestBody)
	s.Log("JSON data: ", requestBody, string(jsonData))
	if err != nil {
		fmt.Printf("Failed to marshal request body: %v\n", err)
		return "", err
	}

	// Create a new POST request with the JSON data
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

	// Print the response body

	return string(body), nil

}

func installApp(ctx context.Context, s *testing.State, a *arc.ARC, d *ui.Device, appPack AppPackage) error {
	if err := playstore.InstallApp(ctx, a, d, appPack.pname, &playstore.Options{TryLimit: 10, InstallationTimeout: 30}); err != nil {
		// GoBigSleepLint N...
		testing.Sleep(ctx, 2*time.Second)
		if err := playstore.InstallApp(ctx, a, d, appPack.pname, &playstore.Options{TryLimit: -1}); err != nil {
			s.Log("Failed to install app: ", appPack.pname)
			return err
		}
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

func loadSecret(s *testing.State) (string, error) {
	b, err := ioutil.ReadFile(s.DataPath("AMACE_secret.txt"))
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	return string(b), nil
}

func loadAppList(s *testing.State) ([]AppPackage, error) {
	// homeDir, err := os.UserHomeDir()
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return nil, err
	// }
	// filePath := filepath.Join("~/chromiumos", "app_list.tsv")
	// // Read the contents of the file
	// content, err := ioutil.ReadFile(filePath)
	b, err := ioutil.ReadFile(s.DataPath("AMACE_app_list.tsv"))
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	// Convert file content to string
	fileContent := string(b)

	// Print the file content
	fmt.Println(fileContent)
	var pgks []AppPackage
	lines := strings.Split(fileContent, "\n")
	// Split each line by tabs
	for _, line := range lines[0:60] {
		fields := strings.Split(line, "\t")
		pgks = append(pgks, AppPackage{fields[1], fields[0]})
		fmt.Println(fields)
	}

	return pgks, nil
}

func getActName(ctx context.Context, s *testing.State, arc *arc.ARC, arcV string) string {
	keyword := ""
	query := `.*{.*\s.*\s(?P<package_name>.*)/(?P<act_name>[\S\.]*)\s*.*}`
	re := regexp.MustCompile(query)

	if arcV == "9" {
		keyword = "mResumedActivity"
	} else if arcV == "11" {
		keyword = "mFocusedWindow"
	}

	cmd := arc.Command(ctx, "dumpsys", "activity")
	output, err := cmd.Output()
	if err != nil {
		s.Fatal("Failed to get dumpsys")
	}
	s.Log("keyword: ", keyword)
	// s.Log(string(output))

	actInfoLine := ""
	// regex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, keyword))
	escapedKeyword := regexp.QuoteMeta(keyword)
	q := fmt.Sprintf(`(?m)^.*\b%s\b.*$`, escapedKeyword)
	s.Log("Query: ", q)
	// regex := regexp.MustCompile(`(?m)^.*\bmFocusedWindow\b.*$`)
	regex := regexp.MustCompile(q)
	matches := regex.FindAllString(string(output), -1)
	if matches != nil {
		for _, match := range matches {
			s.Log("Found match: ", match)
			actInfoLine = string(match)
		}
	} else {
		s.Log("No matches found for info line")
	}

	result := make(map[string]string)
	if actInfoLine != "" {
		actPkgActMatch := re.FindStringSubmatch(actInfoLine)
		if actPkgActMatch != nil {
			groupNames := re.SubexpNames()

			for i, name := range groupNames {
				if i != 0 && name != "" {
					result[name] = actPkgActMatch[i]
				}
			}
			s.Log("Found result: ", result)
		} else {
			s.Log("No match found for act name")
		}

	}

	return result["act_name"]
}

// CheckVisibility checks whether the node specified by the given class name exists or not.
func CheckVisibility(ctx context.Context, tconn *chrome.TestConn, className string, visible bool) error {
	uia := uiauto.New(tconn)
	finder := nodewith.HasClass(className).First()
	if visible {
		return uia.WithTimeout(10 * time.Second).WaitUntilExists(finder)(ctx)
	}
	return uia.WithTimeout(10 * time.Second).WaitUntilGone(finder)(ctx)
}

// Dont really need anything below but nice for reference.
func testO4CAppSimple(ctx context.Context, tconn *chrome.TestConn, keyboard *input.KeyboardEventWriter, s *testing.State, testName, packageName, activityName string) error {
	return testNonResizeLockedSimple(ctx, tconn, keyboard, packageName, activityName, false /* checkRestoreMaximize */, s, testName)
}

// testNonResizeLocked verifies that the given app is not resize locked.
func testNonResizeLockedSimple(ctx context.Context, tconn *chrome.TestConn, keyboard *input.KeyboardEventWriter, packageName, activityName string, checkRestoreMaximize bool, s *testing.State, testName string) (retErr error) {
	// a := s.FixtValue().(*arc.PreData).ARC
	// // cr := s.FixtValue().(*arc.PreData).Chrome

	// activity, err := arc.NewActivity(a, packageName, activityName)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to create %s", activityName)
	// }
	// defer activity.Close(ctx)

	// if err := activity.Start(ctx, tconn); err != nil {
	// 	return errors.Wrapf(err, "failed to start %s", activityName)
	// }
	// defer activity.Stop(ctx, tconn)
	// s.Log("Act started")

	// // defer faillog.DumpUITreeWithScreenshotOnError(ctx, s.OutDir(), func() bool { return retErr != nil }, cr, "ui_dump_"+testName)
	s.Log("Verifying")
	// Verify the initial state of the given non-resize-locked app.

	if err := CheckCompatModeButton(ctx, tconn, NoneResizeLockMode); err != nil {
		return errors.Wrapf(err, "failed to verify the type of the compat mode button of %s", activityName)
	}

	return nil
}

// ResizeLockMode represents the high-level state of the app from the resize-lock feature's perspective.
type ResizeLockMode int

const (
	// PhoneResizeLockMode represents the state where an app is locked in a portrait size.
	PhoneResizeLockMode ResizeLockMode = iota
	// TabletResizeLockMode represents the state where an app is locked in a landscape size.
	TabletResizeLockMode
	// ResizableTogglableResizeLockMode represents the state where an app is not resize lock, and the resize lock state is togglable.
	ResizableTogglableResizeLockMode
	// NoneResizeLockMode represents the state where an app is not eligible for resize lock.
	NoneResizeLockMode
)

func (mode ResizeLockMode) String() string {
	switch mode {
	case PhoneResizeLockMode:
		return phoneButtonName
	case TabletResizeLockMode:
		return tabletButtonName
	case ResizableTogglableResizeLockMode:
		return resizableButtonName
	default:
		return ""
	}
}

const (
	// CenterButtonClassName is the class name of the caption center button.
	CenterButtonClassName = "FrameCenterButton"
	// Used to (i) find the resize lock mode buttons on the compat-mode menu and (ii) check the state of the compat-mode button
	phoneButtonName     = "Phone"
	tabletButtonName    = "Tablet"
	resizableButtonName = "Resizable"
)

// CheckCompatModeButton verifies the state of the compat-mode button of the given app.
// WHne passing in:
//   - NoneResizeLockMode: we get a pass when resize locks are not seen, nothing in middle == PASS
//   - NoneResizeLockMode: we get a fail when resize locks are seen
//     Need to test on more apps to see if this is reliable - it seesintally just calls is CheckVisibility
func CheckCompatModeButton(ctx context.Context, tconn *chrome.TestConn, mode ResizeLockMode) error {
	if mode == NoneResizeLockMode {
		return CheckVisibility(ctx, tconn, CenterButtonClassName, false /* visible */)
	}

	uia := uiauto.New(tconn)
	button := nodewith.HasClass(CenterButtonClassName)
	return testing.Poll(ctx, func(ctx context.Context) error {
		info, err := uia.Info(ctx, button)
		if err != nil {
			return errors.Wrap(err, "failed to find the compat-mode button")
		}

		if info.Name != mode.String() {
			return errors.Errorf("failed to verify the name of compat-mode button; got: %s, want: %s", info.Name, mode)
		}

		return nil
	}, &testing.PollOptions{Timeout: 10 * time.Second})
}
