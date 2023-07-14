// Copyright 2023 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package amace

import (
	"bytes"
	"regexp"
)

func GrepLines(input, pattern string) []string {
	re := regexp.MustCompile(pattern)
	lines := bytes.Split([]byte(input), []byte("\n"))

	var matchedLines []string
	for _, line := range lines {
		if re.Match(line) {
			matchedLines = append(matchedLines, string(line))
		}
	}

	return matchedLines
}

func AbsVal(x int8) int8 {
	if x < 0 {
		return 0 - x
	}
	return x
}

func MaxVal(x int8, y int8) int8 {
	if x > y {
		return x
	}
	return y
}
