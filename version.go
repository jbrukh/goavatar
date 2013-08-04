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
	GoavatarVersionSha   = "414c6701fe534710256c9988329b2da9113911d9"
)

func Version() string {
	return fmt.Sprintf("%d.%d.%s", GoavatarVersionMajor, GoavatarVersionMinor, GoavatarVersionSha)
}
