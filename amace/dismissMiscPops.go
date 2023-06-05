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
	testing.ContextLog(ctx, "Closing Wifi w/ back key")
	return k.TypeKeyAction(input.KEY_BACK)(ctx)
}
