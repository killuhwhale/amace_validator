// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/tast-tests/cros/common/android/ui"
	"go.chromium.org/tast-tests/cros/local/arc"
	"go.chromium.org/tast-tests/cros/local/chrome"
	"go.chromium.org/tast-tests/cros/local/chrome/ash"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto/nodewith"
	"go.chromium.org/tast-tests/cros/local/chrome/uiauto/role"
	"go.chromium.org/tast-tests/cros/local/input"
	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
)

type AppCred struct {
	L string
	P string
}

type AppCreds map[string]AppCred

// AddHistoryWithImage(ctx, tconn, &appHistory, deviceInfo, appPack.Pname, "App crashed with black screen.", runID.Value(), hostIP.Value(), false)
func AttemptLogins(ctx context.Context, a *arc.ARC, tconn *chrome.TestConn, d *ui.Device, cr *chrome.Chrome, keyboard *input.KeyboardEventWriter, ah *AppHistory, hostIP, accountEmail, pkgName, runID, deviceInfo string, appCreds AppCreds, initState ash.WindowStateType, preFBLogin bool) (bool, error) {
	// For each login method:
	// Close App, Clear app storage, open app
	// Reset Error detector time
	// if game: sleep 10 sec
	// attempt login for current method.
	AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, "Testing SS for login.", runID, hostIP, false)
	// TODO add error detector after each login method atempt
	var loggedIn bool
	var err error
	var msg string
	// Reset app window since we changed it when verfiying AMACE status
	// To reset we need to manually put it back to what it was

	if !preFBLogin {
		testing.ContextLog(ctx, "Closing app at beginning of Attempt Log in")
		CloseApp(ctx, a, pkgName)
		if err := LaunchApp(ctx, a, pkgName); err != nil {
			testing.ContextLog(ctx, "Failed  to launch app at beginning of Attempt Log in")
		}
		// GoBigSleepLint Need to wait for app to start...
		testing.Sleep(ctx, 5*time.Second)
		testing.ContextLogf(ctx, "%s's init window state is: %s ", pkgName, initState)
		err = restoreWindow(ctx, tconn, cr, pkgName, initState, keyboard)
		if err != nil {
			testing.ContextLog(ctx, "Failed to restore window: ", err)
		}

	}

	// Login Google
	if false && !preFBLogin {
		loggedIn, err := LoginGoogle(ctx, a, d, hostIP, accountEmail)
		if err != nil {
			// TODO add to LoginResults
			testing.ContextLog(ctx, "Failed to login with Google")
		}
		msg := loggedInOrNahMsg(loggedIn, "Google")
		testing.ContextLog(ctx, "Logged in google: ", msg)
		AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, msg, runID, hostIP, false)
		ClearApp(ctx, a, pkgName)
		CloseApp(ctx, a, pkgName)
		if err := LaunchApp(ctx, a, pkgName); err != nil {
			testing.ContextLog(ctx, "Failed  to launch app after google login")
		}
	}

	// Login Facebook
	if installed := IsAppInstalled(ctx, a, FacebookPackageName); installed && !preFBLogin {
		loggedIn, err = LoginFacebook(ctx, a, d, hostIP, accountEmail)
		if err != nil {
			// TODO add to LoginResults
			testing.ContextLog(ctx, "Failed to login with facebook")
		}
		msg = loggedInOrNahMsg(loggedIn, "Facebook")
		testing.ContextLog(ctx, "Logged in facebook: ", msg)
		AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, msg, runID, hostIP, false)
		ClearApp(ctx, a, pkgName)
		CloseApp(ctx, a, pkgName)
		if err := LaunchApp(ctx, a, pkgName); err != nil {
			testing.ContextLog(ctx, "Failed  to launch app after facebook login")
		}
	} else {
		testing.ContextLog(ctx, "Facebook not installed, skipping attemp to login with Facebook")
	}

	// Login Email
	if ac, exists := appCreds[pkgName]; exists {
		loggedIn, err = LoginEmail(ctx, a, d, keyboard, hostIP, runID, pkgName, deviceInfo, ac, tconn, ah)
		if err != nil {
			// TODO add to LoginResults
			testing.ContextLog(ctx, "Failed to login with Email")
		}
		msg = loggedInOrNahMsg(loggedIn, "Email")
		testing.ContextLog(ctx, "Logged in Email: ", msg)
		AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, msg, runID, hostIP, false)

	} else {
		testing.ContextLog(ctx, "No App Cred available")
	}

	return true, nil
}

func LoginGoogle(ctx context.Context, a *arc.ARC, d *ui.Device, hostIP, account string) (bool, error) {
	// Login method:
	// Allow 3 attempts - Retry entire attempt
	//         Allow 2 empty retries - retry SS if no detection is made

	//         While not submitted and attempts/ retries:
	//             close smart lock
	//             Get SS
	//             if no detection:
	//                 - Retry SS
	//             elif Google login in Detection results:
	//                 click button
	//                 rm result from result set.
	//                 if current_activity == '.common.account.AccountPickerActivity'
	//                     Find Email View to click: .className("android.widget.TextView").text(EmailAddress)

	//                     return True, nil
	//         return False, nil

	attempts := 3
	retries := 3
	SUBMITTED := false
	googleAct := "com.google.android.gms.common.account.AccountPickerActivity"
	for !SUBMITTED && attempts > 0 && retries > 0 {
		attempts--
		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 4*time.Second)

		// TODO() Check and close smart lock
		yr, err := YoloDetect(ctx, hostIP) // Returns a yoloResult
		if err != nil {
			testing.ContextLog(ctx, "Failed to get YoloResult: ", err)
		}
		hasDetection := len(yr.Data) > 0
		if !hasDetection {
			// Retry new detection
			yr, err = YoloDetect(ctx, hostIP) // Returns a yoloResult
			retries--
			continue
		} else if _, labelExists := yr.Data["GoogleAuth"]; labelExists {
			clicked := yr.Click(ctx, a, "GoogleAuth")
			testing.ContextLog(ctx, "Clicked Google Auth btn? ", clicked)
			if clicked {
				// GoBigSleepLint Need to wait for act to start...
				testing.Sleep(ctx, 5*time.Second)
				// check current act for google_act, sometimes, the app will auto login without presenting this view...
				if curAct := CurrentActivity(ctx, a); curAct == googleAct {
					// uidevice nonsense here for login view w/ Email
					testing.ContextLog(ctx, "Clicking Google Email View!")
					accountCreds := strings.Split(account, ":")
					accountEmail := accountCreds[0]
					emailView := d.Object(ui.ClassName("android.widget.TextView"), ui.TextMatches(fmt.Sprintf("(?i)%s", accountEmail)))
					if err := emailView.WaitForExists(ctx, 10*time.Second); err != nil {
						testing.ContextLog(ctx, "Failed waiting for exist: emailView ", err)
						return false, err
					}

					if err := emailView.Click(ctx); err != nil {
						testing.ContextLog(ctx, "Failed clicking: emailView", err)
						return false, err
					}
					testing.ContextLog(ctx, "Clicked Google Login Email View")
				}
				SUBMITTED = true
				return true, nil
			}

		} else if _, labelExists := yr.Data["Continue"]; labelExists {
			clicked := yr.Click(ctx, a, "Continue")
			testing.ContextLog(ctx, "Clicked cont btn? ", clicked)

		}
	}
	return false, nil
}

func LoginFacebook(ctx context.Context, a *arc.ARC, d *ui.Device, hostIP, account string) (bool, error) {
	attempts := 3
	retries := 3
	SUBMITTED := false
	facebookAct := ".gdp.ProxyAuthDialog"
	for !SUBMITTED && attempts > 0 && retries > 0 {
		attempts--
		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 4*time.Second)

		// TODO() Check and close smart lock
		yr, err := YoloDetect(ctx, hostIP) // Returns a yoloResult
		if err != nil {
			testing.ContextLog(ctx, "Failed to get YoloResult: ", err)
		}
		hasDetection := len(yr.Data) > 0
		if !hasDetection {
			// Retry new detection
			yr, err = YoloDetect(ctx, hostIP) // Returns a yoloResult
			retries--
			continue
		} else if _, labelExists := yr.Data["FBAuth"]; labelExists {
			clicked := yr.Click(ctx, a, "FBAuth")
			testing.ContextLog(ctx, "Clicked Facebook Auth btn? ", clicked)
			if clicked {
				// GoBigSleepLint Need to wait for act to start...
				testing.Sleep(ctx, 5*time.Second)
				// check current act for google_act, sometimes, the app will auto login without presenting this view...
				curAct := CurrentActivity(ctx, a)
				if curAct == facebookAct {
					testing.ContextLog(ctx, "Clicking Facebook continue View")

					continueView := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches(fmt.Sprintf("(?i)%s", "Continue")))
					if err := continueView.WaitForExists(ctx, 10*time.Second); err != nil {
						testing.ContextLog(ctx, "Failed waiting for exist: Facebook continueView ", err)
						return false, err
					}

					if err := continueView.Click(ctx); err != nil {
						testing.ContextLog(ctx, "Failed clicking: Facebook continueView", err)
						return false, err
					}
					testing.ContextLog(ctx, "Clicked Facebook continue")
				}
				testing.ContextLog(ctx, "curAct: ", curAct)
				SUBMITTED = true
				return true, nil
			}

		} else if _, labelExists := yr.Data["Continue"]; labelExists {
			clicked := yr.Click(ctx, a, "Continue")
			testing.ContextLog(ctx, "Clicked cont btn? ", clicked)

		}
	}
	return false, nil
}

func LoginEmail(ctx context.Context, a *arc.ARC, d *ui.Device, keyboard *input.KeyboardEventWriter, hostIP, runID, pkgName, deviceInfo string, appCred AppCred, tconn *chrome.TestConn, ah *AppHistory) (bool, error) {
	attempts := 7
	// retries := 7 // might not actually need...
	continueSubmitted := false
	loginEntered := false
	passwordEntered := false

	for !continueSubmitted && attempts > 0 {
		attempts--

		testing.ContextLog(ctx, "Attempts remaining: ", attempts)
		// GoBigSleepLint Need to wait for act to start...
		testing.Sleep(ctx, 4*time.Second)

		// TODO() Check and close smart lock
		yr, err := YoloDetect(ctx, hostIP) // Returns a yoloResult
		AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, strings.Join(yr.Keys(), " - "), runID, hostIP, false)
		if err != nil {
			testing.ContextLog(ctx, "Failed to get YoloResult: ", err)
		}
		hasDetection := len(yr.Data) > 0
		if !hasDetection {
			// Retry new detection
			// yr, err = YoloDetect(ctx, hostIP) // Returns a yoloResult
			// AddHistoryWithImage(ctx, tconn, ah, deviceInfo, pkgName, strings.Join(yr.Keys(), " - "), runID, hostIP, false)
			// retries--
			testing.Sleep(ctx, 2*time.Second)
			continue
		} else if _, labelExists := yr.Data["loginfield"]; labelExists && !loginEntered {
			// loginfield
			// passwordfield
			clicked := yr.Click(ctx, a, "loginfield")
			testing.ContextLog(ctx, "Clicked loginfield? ", clicked)
			if clicked {
				// GoBigSleepLint Need to wait for act to start...
				testing.Sleep(ctx, 2*time.Second)
				textSent := yr.SendTextCr(ctx, keyboard, appCred.L)
				testing.ContextLog(ctx, "Login textSent? ", textSent)
				loginEntered = true
			}

		} else if _, labelExists := yr.Data["passwordfield"]; labelExists && !passwordEntered {
			// loginfield
			// passwordfield
			clicked := yr.Click(ctx, a, "passwordfield")
			testing.ContextLog(ctx, "Clicked passwordfield? ", clicked)
			if clicked {
				// GoBigSleepLint Need to wait for act to start...
				testing.Sleep(ctx, 2*time.Second)
				textSent := yr.SendTextCr(ctx, keyboard, appCred.P)
				testing.ContextLog(ctx, "Password textSent? ", textSent)
				passwordEntered = true
			}

		} else if _, labelExists := yr.Data["Continue"]; labelExists {
			clicked := yr.Click(ctx, a, "Continue")
			testing.ContextLog(ctx, "Clicked cont btn? ", clicked)

			if loginEntered && passwordEntered {
				testing.ContextLog(ctx, "Submitted login form: ", clicked)
				continueSubmitted = true
				// TODO If Facebook App click 1 more continue button
				// fb_attempts = 3
				// while not self.__attempt_click_continue() and fb_attempts > 0:
				// 	fb_attempts -= 1

				return true, nil
			}

		}
	}
	return false, nil
}

func loggedInOrNahMsg(loggedIn bool, loginMethod string) string {
	msg := "App failed to log in: "
	if loggedIn {
		msg = "App logged in: "
	}
	return fmt.Sprintf("%s %s", msg, loginMethod)
}

func restoreWindow(ctx context.Context, tconn *chrome.TestConn, cr *chrome.Chrome, pkgName string, initState ash.WindowStateType, keyboard *input.KeyboardEventWriter) error {
	window, err := ash.GetARCAppWindowInfo(ctx, tconn, pkgName)
	if err != nil {
		return errors.Wrapf(err, "failed to get the ARC window infomation for package name %s", pkgName)
	}

	// If app initState was Full or Maxmimized, check for Maximize Button
	if initState == ash.WindowStateFullscreen || initState == ash.WindowStateMaximized {
		// We need to press maximized but first we need to check if the Maximize button exists...

		testing.ContextLog(ctx, "restoreWindow Window can resize: ", window.CanResize)
		testing.ContextLog(ctx, "restoreWindow Window target bounds: ", window.TargetBounds)

		// Make the app resizable to enable maximization.
		ToggleResizeLockMode(ctx, tconn, cr, ResizableTogglableResizeLockMode, DialogActionConfirmWithDoNotAskMeAgainChecked, InputMethodClick, keyboard)

	}

	// Restore app
	err = testing.Poll(ctx, func(ctx context.Context) error {
		_, err = ash.SetARCAppWindowStateAndWait(ctx, tconn, pkgName, initState)
		if err != nil {
			testing.ContextLog(ctx, "Failed to change app back to initState: ", err)
			return err
		}

		return nil
	}, &testing.PollOptions{Timeout: 60 * time.Second, Interval: 750 * time.Millisecond})

	if err != nil {
		testing.ContextLog(ctx, err)
		return err
	}
	// If exists, press or maximize
	// Else, Choose Amace option Resizeable and then Maximize...
	return nil

}

const (

	// BubbleDialogClassName is the class name of the bubble dialog.
	BubbleDialogClassName = "BubbleDialogDelegateView"

	// Used to (i) find the resize lock mode buttons on the compat-mode menu and (ii) check the state of the compat-mode button
	phoneButtonName     = "Phone"
	tabletButtonName    = "Tablet"
	resizableButtonName = "Resizable"
	// CenterButtonClassName is the class name of the caption center button.
	CenterButtonClassName  = "FrameCenterButton"
	overlayDialogClassName = "OverlayDialog"
	confirmButtonName      = "Allow"
	cancelButtonName       = "Cancel"
	checkBoxClassName      = "Checkbox"
)

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

// ConfirmationDialogAction represents the expected behavior and action to take for the resizability confirmation dialog.
type ConfirmationDialogAction int

const (
	// DialogActionNoDialog represents the behavior where resize confirmation dialog isn't shown when a window is resized.
	DialogActionNoDialog ConfirmationDialogAction = iota
	// DialogActionCancel represents the behavior where resize confirmation dialog is shown, and the cancel button should be selected.
	DialogActionCancel
	// DialogActionConfirm represents the behavior where resize confirmation dialog is shown, and the confirm button should be selected.
	DialogActionConfirm
	// DialogActionConfirmWithDoNotAskMeAgainChecked represents the behavior where resize confirmation dialog is shown, and the confirm button should be selected with the "Don't ask me again" option on.
	DialogActionConfirmWithDoNotAskMeAgainChecked
)

// InputMethodType represents how to interact with UI.
type InputMethodType int

const (
	// InputMethodClick represents the state where UI should be interacted with mouse click.
	InputMethodClick InputMethodType = iota
	// InputMethodKeyEvent represents the state where UI should be interacted with keyboard.
	InputMethodKeyEvent
)

func (mode InputMethodType) String() string {
	switch mode {
	case InputMethodClick:
		return "click"
	case InputMethodKeyEvent:
		return "keyboard"
	default:
		return "unknown"
	}
}

// ToggleResizeLockMode shows the compat-mode menu, selects one of the resize lock mode buttons on the compat-mode menu via the given method, and verifies the post state.
func ToggleResizeLockMode(ctx context.Context, tconn *chrome.TestConn, cr *chrome.Chrome, nextMode ResizeLockMode, action ConfirmationDialogAction, method InputMethodType, keyboard *input.KeyboardEventWriter) error {

	if err := ToggleCompatModeMenu(ctx, tconn, method, keyboard, true /* isMenuVisible */); err != nil {
		// return errors.Wrapf(err, "failed to show the compat-mode dialog of %s via %s", activity.ActivityName(), method)
		return errors.Wrapf(err, "failed to show the compat-mode dialog of %s via %s", "actname", method)
	}

	ui := uiauto.New(tconn)
	compatModeMenuDialog := nodewith.Role(role.Window).HasClass(BubbleDialogClassName)
	if err := ui.WithTimeout(10 * time.Second).WaitUntilExists(compatModeMenuDialog)(ctx); err != nil {
		return errors.Wrapf(err, "failed to find the compat-mode menu dialog of %s", "actName")
	}

	switch method {
	case InputMethodClick:
		if err := selectResizeLockModeViaClick(ctx, tconn, nextMode, compatModeMenuDialog); err != nil {
			return errors.Wrapf(err, "failed to click on the compat-mode dialog of %s via click", "actName")
		}
	case InputMethodKeyEvent:
		if err := shiftViaTabAndEnter(ctx, tconn, nodewith.Ancestor(compatModeMenuDialog).Role(role.MenuItem).Name(nextMode.String()), keyboard); err != nil {
			return errors.Wrapf(err, "failed to click on the compat-mode dialog of %s via keyboard", "actName")
		}
	}

	if action != DialogActionNoDialog {
		if err := waitForCompatModeMenuToDisappear(ctx, tconn); err != nil {
			return errors.Wrapf(err, "failed to wait for the compat-mode menu of %s to disappear", "actName")
		}

		confirmationDialog := nodewith.HasClass(overlayDialogClassName)
		if err := ui.WithTimeout(10 * time.Second).WaitUntilExists(confirmationDialog)(ctx); err != nil {
			return errors.Wrap(err, "failed to find the resizability confirmation dialog")
		}

		switch method {
		case InputMethodClick:
			if err := handleConfirmationDialogViaClick(ctx, tconn, nextMode, confirmationDialog, action); err != nil {
				return errors.Wrapf(err, "failed to handle the confirmation dialog of %s via click", "actName")
			}
		case InputMethodKeyEvent:
			if err := handleConfirmationDialogViaKeyboard(ctx, tconn, nextMode, confirmationDialog, action, keyboard); err != nil {
				return errors.Wrapf(err, "failed to handle the confirmation dialog of %s via keyboard", "actName")
			}
		}
	}

	// The compat-mode dialog stays shown for two seconds by default after resize lock mode is toggled.
	// Explicitly close the dialog using the Esc key.
	if err := ui.WithTimeout(5*time.Second).RetryUntil(func(ctx context.Context) error {
		if err := keyboard.Accel(ctx, "Esc"); err != nil {
			return errors.Wrap(err, "failed to press the Esc key")
		}
		return nil
	}, ui.Gone(nodewith.Role(role.Window).Name(BubbleDialogClassName)))(ctx); err != nil {
		return errors.Wrap(err, "failed to verify that the resizability confirmation dialog is invisible")
	}

	return nil
}

// selectResizeLockModeViaClick clicks on the given resize lock mode button.
func selectResizeLockModeViaClick(ctx context.Context, tconn *chrome.TestConn, mode ResizeLockMode, compatModeMenuDialog *nodewith.Finder) error {
	ui := uiauto.New(tconn)
	resizeLockModeButton := nodewith.Ancestor(compatModeMenuDialog).Role(role.MenuItem).Name(mode.String())
	if err := ui.WithTimeout(10 * time.Second).WaitUntilExists(resizeLockModeButton)(ctx); err != nil {
		return errors.Wrapf(err, "failed to find the %s button on the compat mode menu", mode)
	}
	return ui.LeftClick(resizeLockModeButton)(ctx)
}

// waitForCompatModeMenuToDisappear waits for the compat-mode menu to disappear.
// After one of the resize lock mode buttons are selected, the compat mode menu disappears after a few seconds of delay.
// Can't use chromeui.WaitUntilGone() for this purpose because this function also checks whether the dialog has the "Phone" button or not to ensure that we are checking the correct dialog.
func waitForCompatModeMenuToDisappear(ctx context.Context, tconn *chrome.TestConn) error {
	ui := uiauto.New(tconn)
	dialog := nodewith.ClassName(BubbleDialogClassName).Role(role.Window)
	phoneButton := nodewith.HasClass(phoneButtonName).Ancestor(dialog)
	return ui.WithTimeout(10 * time.Second).WaitUntilGone(phoneButton)(ctx)
}

// ToggleCompatModeMenu toggles the compat-mode menu via the given method
func ToggleCompatModeMenu(ctx context.Context, tconn *chrome.TestConn, method InputMethodType, keyboard *input.KeyboardEventWriter, isMenuVisible bool) error {
	switch method {
	case InputMethodClick:
		return toggleCompatModeMenuViaButtonClick(ctx, tconn, isMenuVisible)
	case InputMethodKeyEvent:
		return toggleCompatModeMenuViaKeyboard(ctx, tconn, keyboard, isMenuVisible)
	}
	return errors.Errorf("invalid InputMethodType is given: %s", method)
}

// toggleCompatModeMenuViaButtonClick clicks on the compat-mode button and verifies the expected visibility of the compat-mode menu.
func toggleCompatModeMenuViaButtonClick(ctx context.Context, tconn *chrome.TestConn, isMenuVisible bool) error {
	ui := uiauto.New(tconn)
	icon := nodewith.Role(role.Button).HasClass(CenterButtonClassName)
	if err := ui.WithTimeout(10 * time.Second).LeftClick(icon)(ctx); err != nil {
		return errors.Wrap(err, "failed to click on the compat-mode button")
	}

	return checkVisibility(ctx, tconn, BubbleDialogClassName, isMenuVisible)
}

// toggleCompatModeMenuViaKeyboard injects the keyboard shortcut and verifies the expected visibility of the compat-mode menu.
func toggleCompatModeMenuViaKeyboard(ctx context.Context, tconn *chrome.TestConn, keyboard *input.KeyboardEventWriter, isMenuVisible bool) error {
	ui := uiauto.New(tconn)
	accel := func(ctx context.Context) error {
		if err := keyboard.Accel(ctx, "Search+Alt+C"); err != nil {
			return errors.Wrap(err, "failed to inject Search+Alt+C")
		}
		return nil
	}
	dialog := nodewith.Role(role.Window).HasClass(BubbleDialogClassName)
	if isMenuVisible {
		return ui.WithTimeout(10*time.Second).WithInterval(2*time.Second).RetryUntil(accel, ui.Exists(dialog))(ctx)
	}
	return nil
}

// handleConfirmationDialogViaKeyboard does the given action for the confirmation dialog via keyboard.
func handleConfirmationDialogViaKeyboard(ctx context.Context, tconn *chrome.TestConn, mode ResizeLockMode, confirmationDialog *nodewith.Finder, action ConfirmationDialogAction, keyboard *input.KeyboardEventWriter) error {
	if action == DialogActionCancel {
		return shiftViaTabAndEnter(ctx, tconn, nodewith.Ancestor(confirmationDialog).Role(role.Button).Name(cancelButtonName), keyboard)
	} else if action == DialogActionConfirm || action == DialogActionConfirmWithDoNotAskMeAgainChecked {
		if action == DialogActionConfirmWithDoNotAskMeAgainChecked {
			if err := shiftViaTabAndEnter(ctx, tconn, nodewith.Ancestor(confirmationDialog).HasClass(checkBoxClassName), keyboard); err != nil {
				return errors.Wrap(err, "failed to select the checkbox of the resizability confirmation dialog via keyboard")
			}
		}
		return shiftViaTabAndEnter(ctx, tconn, nodewith.Ancestor(confirmationDialog).Role(role.Button).Name(confirmButtonName), keyboard)
	}
	return nil
}

// handleConfirmationDialogViaClick does the given action for the confirmation dialog via click.
func handleConfirmationDialogViaClick(ctx context.Context, tconn *chrome.TestConn, mode ResizeLockMode, confirmationDialog *nodewith.Finder, action ConfirmationDialogAction) error {
	ui := uiauto.New(tconn)
	if action == DialogActionCancel {
		cancelButton := nodewith.Ancestor(confirmationDialog).Role(role.Button).Name(cancelButtonName)
		return ui.WithTimeout(10 * time.Second).LeftClick(cancelButton)(ctx)
	} else if action == DialogActionConfirm || action == DialogActionConfirmWithDoNotAskMeAgainChecked {
		if action == DialogActionConfirmWithDoNotAskMeAgainChecked {
			checkbox := nodewith.HasClass(checkBoxClassName)
			if err := ui.WithTimeout(10 * time.Second).LeftClick(checkbox)(ctx); err != nil {
				return errors.Wrap(err, "failed to click on the checkbox of the resizability confirmation dialog")
			}
		}

		confirmButton := nodewith.Ancestor(confirmationDialog).Role(role.Button).Name(confirmButtonName)
		return ui.WithTimeout(10 * time.Second).LeftClick(confirmButton)(ctx)
	}
	return nil
}

// shiftViaTabAndEnter keeps pressing the Tab key until the UI element of interest gets focus, and press the Enter key.
func shiftViaTabAndEnter(ctx context.Context, tconn *chrome.TestConn, target *nodewith.Finder, keyboard *input.KeyboardEventWriter) error {
	ui := uiauto.New(tconn)
	if err := testing.Poll(ctx, func(ctx context.Context) error {
		if err := keyboard.Accel(ctx, "Tab"); err != nil {
			return errors.Wrap(err, "failed to press the Tab key")
		}
		if err := ui.Exists(target)(ctx); err != nil {
			return testing.PollBreak(errors.Wrap(err, "failed to find the node seeking focus"))
		}
		return ui.Exists(target.Focused())(ctx)
	}, &testing.PollOptions{Timeout: 10 * time.Second}); err != nil {
		return errors.Wrap(err, "failed to shift focus to the node to click on")
	}
	return keyboard.Accel(ctx, "Enter")
}

// TODO() move this somewhere because its used in amace.go as well...
// checkVisibility checks whether the node specified by the given class name exists or not.
func checkVisibility(ctx context.Context, tconn *chrome.TestConn, className string, visible bool) error {
	uia := uiauto.New(tconn)
	finder := nodewith.HasClass(className).First()
	if visible {
		return uia.WithTimeout(10 * time.Second).WaitUntilExists(finder)(ctx)
	}
	return uia.WithTimeout(10 * time.Second).WaitUntilGone(finder)(ctx)
}
