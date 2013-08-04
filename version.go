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
	GoavatarVersionSha   = "b13ccde02340b74bc35f506441fd7a01a0b7d60a"
)

func Version() string {
	return fmt.Sprintf("%d.%d.%s", GoavatarVersionMajor, GoavatarVersionMinor, GoavatarVersionSha)
}
