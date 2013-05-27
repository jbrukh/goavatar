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
	GoavatarVersionSha   = "a84852b0e57e05f14c9c370e3ff3c09b7c02e1f1"
)

func Version() string {
	return fmt.Sprintf("%d.%d.%s", GoavatarVersionMajor, GoavatarVersionMinor, GoavatarVersionSha)
}
