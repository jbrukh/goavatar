//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"crypto/sha1"
)

var hash = sha1.New()

// A real-time recorder of dataframes.
type Recorder interface {
	Start() error
	ProcessFrame(DataFrame) error
	Stop() (id string, err error)
}
