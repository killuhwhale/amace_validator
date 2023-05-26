// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import "fmt"

// AppStatus indicates the final status of checking the app
type AppStatus int

const (
	// Fail indicates failure
	Fail AppStatus = iota // 0
	// O4C indicates O4C
	O4C // 1
	// IsAmacE indicates IsAmacE
	IsAmacE // 2
	// PRICE indicates PRICE
	PRICE // 3
	// OLDVERSION indicates OLDVERSION
	OLDVERSION // 4
	// INSTALLFAIL indicates INSTALLFAIL
	INSTALLFAIL // 5
)

// AppResult stores result of checking app
type AppResult struct {
	App    AppPackage
	RunID  string
	RunTS  int64
	AppTS  int64
	Status AppStatus
}

func (ar AppResult) String() string {
	return fmt.Sprintf("isAmac-e: %v, Name: %s", ar.Status, ar.App.Aname)
}
