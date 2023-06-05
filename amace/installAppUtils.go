// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"chromiumos/tast/common/android/ui"
	"chromiumos/tast/common/testexec"
	"chromiumos/tast/local/arc"
	"context"
	"fmt"
	"time"

	"go.chromium.org/tast/core/errors"
	"go.chromium.org/tast/core/testing"
)

// Options contains options used when installing or updating an app.
type Options struct {
	// TryLimit limits number of tries to install or update an app.
	// Default value is 3, and -1 means unlimited.
	TryLimit int

	// DefaultUITimeout is used when waiting for UI elements.
	// Default value is 20 sec.
	DefaultUITimeout time.Duration

	// ShortUITimeout is used when waiting for "Complete account setup" button.
	// Default value is 10 sec.
	ShortUITimeout time.Duration

	// InstallationTimeout is used when waiting for app installation.
	// Default value is 90 sec.
	InstallationTimeout time.Duration
}

type operation string

// InstallARCApp uses the Play Store to install or update an application.
func InstallARCApp(ctx context.Context, s *testing.State, a *arc.ARC, d *ui.Device, appPack AppPackage) error {
	if err := installApp(ctx, a, d, appPack.Pname, &Options{TryLimit: 3, InstallationTimeout: 30}); err != nil {
		s.Log("Failed to install app: ", appPack.Pname)
		return err
	}
	return nil
}

func installApp(ctx context.Context, a *arc.ARC, d *ui.Device, pkgName string, opt *Options) error {
	installed, err := a.PackageInstalled(ctx, pkgName)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	if err := installOrUpdate(ctx, a, d, pkgName, opt); err != nil {
		return err
	}

	// Ensure that the correct package is installed, just in case the Play Store ui changes again.
	installed, err = a.PackageInstalled(ctx, pkgName)
	if err != nil {
		return err
	}
	if !installed {
		return errors.Errorf("failed to install %s", pkgName)
	}
	return nil
}

// installOrUpdate uses the Play Store to install or update an application.
func installOrUpdate(ctx context.Context, a *arc.ARC, d *ui.Device, pkgName string, opt *Options) error {
	const (
		accountSetupText      = "Complete account setup"
		permissionsText       = "needs access to"
		versionText           = "Your device isn't compatible with this version."
		versionTextOldVersion = "This app isn't available for your device because it was made for an older version of Android."
		countryNA             = "This item isn't available in your country."
		linkPaypalAccountText = "Want to link your PayPal account.*"

		acceptButtonText   = "accept"
		continueButtonText = "continue"
		installButtonText  = "install"
		updateButtonText   = "update"
		openButtonText     = "open"
		playButtonText     = "play"
		priceRegex         = ".*\\$.*"

		retryButtonText    = "retry"
		tryAgainButtonText = "try again"
		skipButtonText     = "skip"
		noThanksButtonText = "No thanks"
	)

	o := *opt
	tryLimit := 3
	if o.TryLimit != 0 {
		tryLimit = o.TryLimit
	}
	defaultUITimeout := 20 * time.Second
	if o.DefaultUITimeout != 0 {
		defaultUITimeout = o.DefaultUITimeout
	}
	shortUITimeout := 10 * time.Second
	if o.ShortUITimeout != 0 {
		shortUITimeout = o.ShortUITimeout
	}
	installationTimeout := 90 * time.Second
	if o.InstallationTimeout != 0 {
		installationTimeout = o.InstallationTimeout
	}
	testing.ContextLogf(ctx, "Using TryLimit=%d, DefaultUITimeout=%s, ShortUITimeout=%s, InstallationTimeout=%s",
		tryLimit, defaultUITimeout, shortUITimeout, installationTimeout)

	testing.ContextLog(ctx, "Opening Play Store with Intent")
	if err := a.WaitIntentHelper(ctx); err != nil {
		return errors.Wrap(err, "failed to wait for ArcIntentHelper")
	}

	if err := openAppPage(ctx, a, pkgName); err != nil {
		return err
	}

	btnText := installButtonText

	// Wait for the app to install or update.
	testing.ContextLog(ctx, "Waiting for app to install")

	tries := 0
	pollTries := 0

	testing.ContextLog(ctx, "Polling app to install process return Poll")
	return testing.Poll(ctx, func(ctx context.Context) error {
		testing.ContextLog(ctx, "🔥 Current Poll Tries:  ", pollTries)
		if pollTries > 2 {
			return testing.PollBreak(errors.New("too many attempst: app failed to install"))
		}
		pollTries++
		if err := findAndDismissErrorDialog(ctx, d); err != nil {
			return testing.PollBreak(err)
		}

		// If the version isn't compatible with the device, no install button will be available.
		// Fail immediately.
		testing.ContextLog(ctx, "Checking Old version text ")
		if err := d.Object(ui.TextMatches(versionTextOldVersion)).Exists(ctx); err == nil {
			return testing.PollBreak(errors.New("app not compatible with this device"))
		}
		testing.ContextLog(ctx, "Checking valid country ")
		if err := d.Object(ui.TextMatches(countryNA)).Exists(ctx); err == nil {
			return testing.PollBreak(errors.New("app not availble in your country"))
		}
		testing.ContextLog(ctx, "App is valid in country ")

		// Make sure we are still on the Play Store installation page by checking whether the "open" or "play" button exists.
		// If not, reopen the Play Store page by sending the same intent again.
		testing.ContextLog(ctx, "Checking for Open/Play button ")
		if err := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches(fmt.Sprintf("(?i)(%s|%s)", openButtonText, playButtonText))).Exists(ctx); err != nil {
			// Check for version error
			testing.ContextLog(ctx, "App installation page disappeared; reopen it")
			if err := openAppPage(ctx, a, pkgName); err != nil {
				return err
			}
		}

		// If the install or update button is enabled, click it.
		if opButton, err := FindActionButton(ctx, d, btnText, 2*time.Second); err == nil {
			testing.ContextLog(ctx, "Found install button")
			// Limit number of tries to help mitigate Play Store rate limiting across test runs.
			if tryLimit == -1 || tries < tryLimit {
				tries++
				testing.ContextLogf(ctx, "Trying to hit the install button. Total attempts so far: %d", tries)
				if err := opButton.Click(ctx); err != nil {
					return err
				}
			} else {
				return testing.PollBreak(errors.Errorf("hit install attempt limit of %d times", tryLimit))
			}
		}

		// Grant permissions if necessary.
		if err := findAndDismissDialog(ctx, d, permissionsText, acceptButtonText); err != nil {
			return testing.PollBreak(err)
		}

		// Handle "Want to link your PayPal account" if necessary.
		testing.ContextLogf(ctx, "Checking existence of : %s", linkPaypalAccountText)
		if err := d.Object(ui.TextMatches("(?i)"+linkPaypalAccountText), ui.Enabled(true)).WaitForExists(ctx, defaultUITimeout); err == nil {
			testing.ContextLog(ctx, "Want to link your paypal account does exist")
			noThanksButton := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches("(?i)"+noThanksButtonText))
			if err := noThanksButton.WaitForExists(ctx, defaultUITimeout); err != nil {
				return testing.PollBreak(err)
			}
			if err := noThanksButton.Click(ctx); err != nil {
				return testing.PollBreak(err)
			}
			skipButton := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches("(?i)"+skipButtonText))
			if err := skipButton.WaitForExists(ctx, defaultUITimeout); err != nil {
				return testing.PollBreak(err)
			}
			if err := skipButton.Click(ctx); err != nil {
				return testing.PollBreak(err)
			}
		}

		// Complete account setup if necessary.
		testing.ContextLogf(ctx, "Checking existence of : %s", accountSetupText)
		if err := d.Object(ui.Text(accountSetupText), ui.Enabled(true)).WaitForExists(ctx, shortUITimeout); err == nil {
			testing.ContextLog(ctx, "Completing account setup")
			continueButton := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches("(?i)"+continueButtonText))
			if err := continueButton.WaitForExists(ctx, defaultUITimeout); err != nil {
				return testing.PollBreak(err)
			}
			if err := continueButton.Click(ctx); err != nil {
				return testing.PollBreak(err)
			}
			skipButton := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches("(?i)"+skipButtonText))
			if err := skipButton.WaitForExists(ctx, defaultUITimeout); err != nil {
				return testing.PollBreak(err)
			}
			if err := skipButton.Click(ctx); err != nil {
				return testing.PollBreak(err)
			}
		}

		// There are two possible of descriptions on the Play Store installation page.
		// One is "Download in progress", the other is "Install in progress".
		// If one of them exists, that means the installation is still in progress.

		// This text isnt found or print, probably outdated...
		// progress := d.Object(ui.DescriptionContains("in progress"))
		// if err := progress.WaitForExists(ctx, defaultUITimeout); err == nil {
		// 	// Print the percentage of app installed so far.
		// 	printPercentageOfAppInstalled(ctx, d)
		// 	testing.ContextLog(ctx, "Wait until download and install complete")
		// 	if err := progress.WaitUntilGone(ctx, installationTimeout); err != nil {
		// 		return errors.Wrap(err, "installation is still in progress")
		// 	}
		// }

		testing.ContextLog(ctx, "Checking Price Button ")
		if err := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches(priceRegex)).Exists(ctx); err != nil {
			testing.ContextLog(ctx, "Price button DNE ")
		} else {
			testing.ContextLog(ctx, "Price exists")
			return testing.PollBreak(errors.New("Need to purchase app"))
		}

		// If retry button appears, reopen the Play Store page by sending the same intent again.
		// (It tends to work better than clicking the retry button.)
		testing.ContextLog(ctx, "Checking Someting went wrong : try again or retry ")
		if err := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches(fmt.Sprintf("(?i)(%s|%s)", retryButtonText, tryAgainButtonText))).Exists(ctx); err == nil {
			testing.ContextLogf(ctx, "Retry button is shown. Trying to reopen the Play Store. Total attempts so far: %d (%d)", tries, tryLimit)
			if tryLimit == -1 || tries < tryLimit {
				tries++
				testing.ContextLogf(ctx, "Retry button is shown. Trying to reopen the Play Store. Total attempts so far: %d", tries)
				if err := openAppPage(ctx, a, pkgName); err != nil {
					return err
				}
			} else {
				return testing.PollBreak(errors.Errorf("reopen Play Store attempt limit of %d times", tryLimit))
			}
		}

		installed, err := a.PackageInstalled(ctx, pkgName)
		if err != nil {
			return errors.Wrap(err, "failed to check app installation status")
		}
		if !installed {
			return errors.New("app not yet installed")
		}

		return nil
	}, &testing.PollOptions{Interval: time.Second})
}

// openAppPage opens the detail page of an app in Play Store.
func openAppPage(ctx context.Context, a *arc.ARC, pkgName string) error {
	const (
		intentActionView    = "android.intent.action.VIEW"
		playStoreAppPageURI = "market://details?id="
	)

	if err := a.SendIntentCommand(ctx, intentActionView, playStoreAppPageURI+pkgName).Run(testexec.DumpLogOnError); err != nil {
		return errors.Wrap(err, "failed to send intent to open the Play Store")
	}

	return nil
}

// FindActionButton finds the action button on app detail page.
func FindActionButton(ctx context.Context, d *ui.Device, actionText string, timeout time.Duration) (*ui.Object, error) {
	var result *ui.Object

	err := testing.Poll(ctx, func(ctx context.Context) error {
		buttonClass := ui.ClassName("android.widget.Button")
		actionButton := d.Object(buttonClass, ui.TextMatches("(?i)"+actionText), ui.Enabled(true))
		if err := actionButton.WaitForExists(ctx, time.Second); err == nil {
			testing.ContextLog(ctx, "Found the button")
			result = actionButton
			return nil
		}

		viewClass := ui.ClassName("android.view.View")
		actionView := d.Object(viewClass, ui.DescriptionMatches("(?i)"+actionText), ui.Enabled(true))
		if err := actionView.WaitForExists(ctx, time.Second); err == nil {
			testing.ContextLog(ctx, "Found the view")
			result = actionView
			return nil
		}

		// Check for Price Button
		buttonClass = ui.ClassName("android.widget.Button")
		actionButton = d.Object(buttonClass, ui.TextMatches("(?i).*\\$.*"), ui.Enabled(true))
		if err := actionButton.WaitForExists(ctx, time.Second); err == nil {
			testing.ContextLog(ctx, "Found the button")
			result = actionButton
			return errors.New("found price button")
		}

		viewClass = ui.ClassName("android.view.View")
		actionView = d.Object(viewClass, ui.DescriptionMatches("(?i).*\\$.*"), ui.Enabled(true))
		if err := actionView.WaitForExists(ctx, time.Second); err == nil {
			testing.ContextLog(ctx, "Found the view")
			result = actionView
			return errors.New("found price button")
		}

		return errors.New("Did not find the button")
	}, &testing.PollOptions{Timeout: timeout, Interval: time.Second})

	return result, err
}

// findAndDismissErrorDialog finds and dismisses all possible intermittent errors in Play Store.
func findAndDismissErrorDialog(ctx context.Context, d *ui.Device) error {
	const (
		serverErrorText           = "Server busy.*|Server error|Error.*server.*|.*connection with the server.|Connection timed out."
		cantDownloadText          = "Can.t download.*"
		cantInstallText           = "Can.t install.*"
		compatibleText            = "Your device is not compatible with this item."
		openMyAppsText            = "Please open my apps.*"
		termsOfServiceText        = "Terms of Service"
		installAppsFromDeviceText = "Install apps from your devices"
		internalProblemText       = "There.s an internal problem with your device.*"
		itemNotFoundText          = ".*item.*could not be found.*"

		acceptButtonText       = "accept"
		gotItButtonText        = "got it"
		okButtonText           = "ok"
		noThanksButtonText     = "No thanks"
		tryAgainOrOkButtonText = "Try again|OK"
	)

	for _, val := range []struct {
		dialogText string
		buttonText string
	}{
		// Due to timing of propagation of policy, the UI may be enabled but the item is not available.
		{itemNotFoundText, okButtonText},
		// These are intermittent server side errors that can happen under load.
		{serverErrorText, tryAgainOrOkButtonText},
		// Sometimes a dialog of "Can't download <app name>" pops up. Press "Got it" to
		// dismiss the dialog. This check needs to be done before checking the
		// install button since the install button exists underneath.
		{cantDownloadText, gotItButtonText},
		// Similarly, press "Got it" button if "Can't install <app name>" dialog pops up.
		{cantInstallText, gotItButtonText},
		// Also, press Ok to dismiss the dialog if "Please open my apps" dialog pops up.
		{openMyAppsText, okButtonText},
		// Also, press "NO THANKS" to dismiss the dialog if "Install apps from your devices" dialog pops up.
		{installAppsFromDeviceText, noThanksButtonText},
		// When Play Store hits the rate limit it sometimes show "Your device is not compatible with this item." error.
		// This error is incorrect and should be ignored like the "Can't download <app name>" error.
		{compatibleText, okButtonText},
		// Somehow, playstore shows a ToS dialog upon opening even after playsore
		// optin finishes. Click "accept" button to accept and dismiss.
		{termsOfServiceText, acceptButtonText},
		// Press Ok to dismiss the dialog if "There\'s an internal problem with your device" dialog pops up.
		{internalProblemText, okButtonText},
	} {
		if err := findAndDismissDialog(ctx, d, val.dialogText, val.buttonText); err != nil {
			return err
		}
	}

	return nil
}

// findAndDismissDialog finds a dialog containing text with a corresponding button and presses the button.
func findAndDismissDialog(ctx context.Context, d *ui.Device, dialogText, buttonText string) error {
	if err := d.Object(ui.TextMatches("(?i)" + dialogText)).Exists(ctx); err == nil {
		testing.ContextLogf(ctx, `%q popup found. Skipping`, dialogText)
		okButton := d.Object(ui.ClassName("android.widget.Button"), ui.TextMatches("(?i)"+buttonText))
		if err := okButton.WaitForExists(ctx, time.Second); err != nil {
			return err
		}
		if err := okButton.Click(ctx); err != nil {
			return err
		}
	}
	return nil
}
