// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import "fmt"

// AppStatus indicates the final status of checking the app
type AppStatus int

// When updating AppStatus, frontend needs to be updated as well
// 1. pages/amace/processStats/reasons and add tally & graph
// 2. componenets/AmaceRResultTable/status_reasons update descriptive display text.
const (
	// Fail indicates failure
	Fail AppStatus = iota // 0
	// PRICE indicates needs purchase
	PRICE // 1
	// OLDVERSION indicates target SDK is old
	OLDVERSION // 2
	// INSTALLFAIL indicates App install failed, usually due to Invalid App (hangouts, country, old version, or other manifest compat issues)
	INSTALLFAIL // 3
	// INSTALLFAIL indicates app not availble in country
	COUNTRYNA // 4
	// O4C indicates O4C
	O4C // 5
	// O4CFullScreenOnly indicates O4C but app is only fullscreen.
	O4CFullScreenOnly // 6
	// IsFSToAmacE indicates Fullscreen (no amace) to Amace(after restore)
	IsFSToAmacE // 7
	// IsLockedPAmacE indicates the app is locked to phone
	IsLockedPAmacE // 8
	// IsLockedPAmacE indicates the app is locked to tablet
	IsLockedTAmacE // 9
	// IsAmacE indicates IsAmacE in all windows and modes
	IsAmacE // 10
	// PWA indicates app is PWA (TikTok)
	PWA // 11
)

func (ap *AppStatus) String() string {
	return ap.String()
}

// AppResult stores result of checking app
type AppResult struct {
	App    AppPackage
	RunID  string
	RunTS  string
	AppTS  int64
	Status AppStatus
	IsGame bool
}

func (ar AppResult) String() string {
	return fmt.Sprintf("isAmac-e: %v, Name: %s", ar.Status, ar.App.Aname)
}
