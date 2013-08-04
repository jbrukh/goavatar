//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	"fmt"
)

const (
	GoavatarVersionMajor = 0
	GoavatarVersionMinor = 1
	GoavatarVersionSha   = "761340a1f48620fd42180e54ba2ce0e03adbf1d9"
)

func Version() string {
	return fmt.Sprintf("%d.%d.%s", GoavatarVersionMajor, GoavatarVersionMinor, GoavatarVersionSha)
}
