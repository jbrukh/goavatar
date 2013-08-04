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
	GoavatarVersionSha   = "c722471e3344b3bd31e5ae3c7fdcf509fed79c5e"
)

func Version() string {
	return fmt.Sprintf("%d.%d.%s", GoavatarVersionMajor, GoavatarVersionMinor, GoavatarVersionSha)
}
