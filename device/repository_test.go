//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"os"
	"testing"
)

const testBaseDir = "../var/poop"

// Test repository, basedir, and subdir
// creation.
func TestNewRepository(t *testing.T) {
	r, err := NewRepository(testBaseDir)

	// something wrong with constructor
	if err != nil {
		t.Errorf("could not create the directories")
	}

	// check basedir is set
	if r.Basedir() != testBaseDir {
		t.Errorf("basedir not set")
	}

	// check basedir created
	if _, err := os.Stat(testBaseDir); os.IsNotExist(err) {
		t.Errorf("basedir wasn't created")
	}

	// check that there are two subdirs
	if len(subdirSearchPath) != 2 {
		t.Errorf("should be local and cloud subdirs")
	}

	// check that we can retrieve them
	subdirs := r.Subdirs()
	if len(subdirs) != 2 {
		t.Errorf("should be able to get subdirs")
	}

	// check that they were created
	for _, subdir := range subdirs {
		if _, err := os.Stat(subdir); os.IsNotExist(err) {
			t.Errorf("did not create subdir")
		}
	}
}
