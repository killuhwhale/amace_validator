// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"chromiumos/tast/local/input"
	"context"

	"go.chromium.org/tast/core/testing"
)

// CloseBNobleWifi will check and close WiFi popup.
func CloseBNobleWifi(ctx context.Context, k *input.KeyboardEventWriter) error {
	// maxTries := 3
	// pollTries := 0
	testing.ContextLog(ctx, "Closing Wifi w/ back key")

	return k.TypeKeyAction(input.KEY_BACK)(ctx)

	// return testing.Poll(ctx, func(ctx context.Context) error {
	// 	testing.ContextLog(ctx, "🔥 Current BN WiFi Pop up Tries:  ", pollTries)

	// 	pollTries++
	// 	if pollTries > maxTries {
	// 		return testing.PollBreak(errors.New("too many attempst: app failed to install"))
	// 	}

	// 	var err error
	// 	if err = d.Object(ui.TextMatches("Connect to Wi-Fi networks?")).Exists(ctx); err == nil {
	// 		testing.ContextLog(ctx, "BN Text exists")
	// 		// Text exists, click No button
	// 		if opButton, err := FindActionButton(ctx, d, "No", 2*time.Second); err == nil {
	// 			// Limit number of tries to help mitigate Play Store rate limiting across test runs.
	// 			testing.ContextLog(ctx, "BN btn found")
	// 			if pollTries < maxTries {

	// 				testing.ContextLogf(ctx, "Trying to hit the install button. Total attempts so far: %d", pollTries)
	// 				if err := opButton.Click(ctx); err != nil {
	// 					return err
	// 				}
	// 			} else {
	// 				return testing.PollBreak(errors.Errorf("hit install attempt limit of %d times", maxTries))
	// 			}
	// 		}
	// 		testing.ContextLog(ctx, "'No' btn not found")

	// 		return err

	// 		// return testing.PollBreak(errors.New("app not compatible with this device"))
	// 	}
	// 	testing.ContextLog(ctx, "Connect to wifi not found...")

	// 	return err
	// }, &testing.PollOptions{Interval: time.Second})

}
