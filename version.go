package goavatar

import (
	"fmt"
)

const (
	GoavatarVersionMajor = 0
	GoavatarVersionMinor = 1
)

func Version() string {
	return fmt.Sprintf("%d.%d", GoavatarVersionMajor, GoavatarVersionMinor)
}
