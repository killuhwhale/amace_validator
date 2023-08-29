// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"fmt"
	"os/exec"
	"io/ioutil"
	"strings"
	"context"

	"go.chromium.org/tast/core/testing"
)

func InstallApkApp(ctx context.Context, s *testing.State, appPack AppPackage, ip string) (AppStatus, error) {
	testing.ContextLogf(ctx, "In apk install")
	myFile := ""
	cmd := exec.Command("adb", "connect", ip+":5555")
	testing.ContextLogf(ctx, "Connecting to adb")
	err := cmd.Run()
	if err != nil {
		testing.ContextLogf(ctx, "FAILED: ", err)
		return Fail, err
	}

	dir := "/usr/local/share/tast/data_pushed/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		testing.ContextLogf(ctx, "FAILED: ", err)
		return Fail, err
	}
	
	for _, file := range files {
		testing.ContextLogf(ctx, file.Name())
		if strings.Contains(file.Name(), appPack.Pname) {
			myFile = file.Name()
		}
	}
	testing.ContextLogf(ctx, "My file: " + myFile)
	
	if strings.Contains(myFile, ".apk") {
		testing.ContextLogf(ctx, dir + "/" + myFile)
		testing.ContextLogf(ctx, "Installing single apk")
		cmd := exec.Command("adb", "install", dir + "/" + myFile)
		err := cmd.Run()
		if err != nil {
			testing.ContextLogf(ctx, fmt.Sprint(err))
			return Fail, err
		}
	} else {

		testing.ContextLogf(ctx, "Unzipping split apk files")
		cmd = exec.Command("unzip", "-o", dir + "/" + myFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			testing.ContextLogf(ctx, fmt.Sprint(err) + ": " + string(output))
			return Fail, err
		}
		
		files, err := ioutil.ReadDir(".")
		apkFiles := []string{}
		for _, file := range files {
			if strings.Contains(file.Name(), ".apk") {
				apkFiles = append(apkFiles, file.Name())
			}
		}

		args := []string{"install-multiple"}
		args = append(args, apkFiles...)
		testing.ContextLogf(ctx, "Installing split apk")
		cmd = exec.Command("adb", args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			testing.ContextLogf(ctx, fmt.Sprint(err) + ": " + string(output))
			return Fail, err
		}
		
		cmd = exec.Command("rm", apkFiles...)
		testing.ContextLogf(ctx, "Removing split apk files")
		output, err = cmd.CombinedOutput()
		if err != nil {
			testing.ContextLogf(ctx, fmt.Sprint(err) + ": " + string(output))
			return Fail, err
		}
	}

	return SKIPPEDAMACE, nil
}