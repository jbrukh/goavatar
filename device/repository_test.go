//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"testing"
)

const testBaseDir = "var/poop"

func TestNewRepository(t *testing.T) {
	r := NewRepository(testBaseDir)
	if r.BaseDir() != testBaseDir {
		t.Errorf("could not create the baseDir")
	}
}
